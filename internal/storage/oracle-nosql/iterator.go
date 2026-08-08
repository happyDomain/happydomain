// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"fmt"
	"log"
	"time"

	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/nosqlerr"
	"github.com/oracle/nosql-go-sdk/nosqldb/types"
)

// Fetch pacing for a single iterator.
//
// A query that runs into its read limit must slow down rather than spin. The
// service answers such a fetch with an *empty batch plus a continuation key*,
// not an error, so retrying immediately still consumes read units and keeps
// the table throttled — a query large enough to be throttled once then stays
// throttled forever, entirely on its own. Empty batches and explicit
// throttling errors therefore share a single backoff, and the whole iterator
// runs on one budget so the caller always ends up with an error instead of
// blocking indefinitely.
const (
	iteratorBudget  = 2 * time.Minute
	backoffInitial  = 100 * time.Millisecond
	backoffMax      = 5 * time.Second
	backoffLogFloor = time.Second
)

type Iterator struct {
	n       *NoSQLStorage
	req     *nosqldb.QueryRequest
	results []*types.MapValue
	cur     int
	started bool
	err     error

	// deadline is armed on the first fetch and spans the whole iterator, not
	// a single Next call, so a slow query cannot renew its budget forever.
	deadline time.Time
	backoff  time.Duration
}

func NewIteratorFromRequest(n *NoSQLStorage, req *nosqldb.QueryRequest) *Iterator {
	return &Iterator{
		n:   n,
		req: req,
	}
}

func (i *Iterator) Release() {
	i.req.Close()
}

func (i *Iterator) Next() bool {
	// Once failed, stay failed: never clear a recorded error, and never issue
	// further queries for an iterator whose results are already incomplete.
	if i.err != nil {
		return false
	}

	// Advance within the batch already in hand.
	if i.cur+1 < len(i.results) {
		i.cur++
		return true
	}

	return i.fetch()
}

// fetch pulls batches until one carries results, the query is exhausted, or
// the budget runs out.
func (i *Iterator) fetch() bool {
	if i.deadline.IsZero() {
		i.deadline = time.Now().Add(iteratorBudget)
	}

	for {
		// IsDone reports continuationKey == nil, which is also true of a fresh
		// request that has never run — hence the started guard.
		if i.started && i.req.IsDone() {
			return false
		}

		res, err := i.n.client.Query(i.req)
		if err != nil {
			if nosqlerr.Is(err, nosqlerr.ReadLimitExceeded, nosqlerr.WriteLimitExceeded, nosqlerr.RequestTimeout) {
				if !i.pause("throttled: " + err.Error()) {
					return false
				}
				continue
			}
			i.fail(err)
			return false
		}
		i.started = true

		results, err := res.GetResults()
		if err != nil {
			i.fail(fmt.Errorf("decoding query results: %w", err))
			return false
		}

		if len(results) > 0 {
			i.results = results
			i.cur = 0
			i.backoff = 0
			return true
		}

		// Empty batch with the query still unfinished: the read limit was hit
		// server-side. Treated exactly like a throttling error.
		if i.req.IsDone() {
			return false
		}
		if !i.pause("read limit reached, empty batch") {
			return false
		}
	}
}

// pause sleeps for the current backoff and doubles it, returning false once
// the iterator's budget is exhausted.
func (i *Iterator) pause(reason string) bool {
	switch {
	case i.backoff == 0:
		i.backoff = backoffInitial
	case i.backoff < backoffMax:
		i.backoff = min(i.backoff*2, backoffMax)
	}

	remaining := time.Until(i.deadline)
	if remaining <= 0 {
		i.fail(fmt.Errorf("query still incomplete after %s (%s)", iteratorBudget, reason))
		return false
	}

	// Only report once the backoff is long enough to matter, so a briefly
	// throttled query stays quiet instead of flooding the logs.
	if i.backoff >= backoffLogFloor {
		log.Printf("oracle-nosql: %s, retrying in %s (%s left)",
			reason, i.backoff, remaining.Round(time.Second))
	}

	time.Sleep(min(i.backoff, remaining))
	return true
}

func (i *Iterator) fail(err error) {
	i.err = err
	i.results = nil
	i.cur = 0
	log.Printf("oracle-nosql iterator: %v", err)
}

// Valid reports whether the iterator currently points at a usable row.
func (i *Iterator) Valid() bool {
	if i.err != nil || i.cur >= len(i.results) {
		return false
	}

	_, okkey := i.results[i.cur].Get("key")
	_, okvalue := i.results[i.cur].Get("value")

	return okkey && okvalue
}

func (i *Iterator) Key() string {
	if i.cur >= len(i.results) {
		return ""
	}

	key, _ := i.results[i.cur].Get("key")
	s, _ := key.(string)
	return s
}

func (i *Iterator) Value() any {
	if i.cur >= len(i.results) {
		return nil
	}

	value, _ := i.results[i.cur].Get("value")
	return value
}

func (i *Iterator) Err() error {
	return i.err
}
