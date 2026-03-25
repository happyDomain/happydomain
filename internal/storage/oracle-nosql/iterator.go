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
	"log"
	"time"

	"github.com/oracle/nosql-go-sdk/nosqldb"
	"github.com/oracle/nosql-go-sdk/nosqldb/nosqlerr"
	"github.com/oracle/nosql-go-sdk/nosqldb/types"
)

type Iterator struct {
	n          *NoSQLStorage
	req        *nosqldb.QueryRequest
	results    []*types.MapValue
	cur_result int
	started    bool
	err        error
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
	i.err = nil

	// Advance within current batch.
	if i.results != nil {
		i.cur_result++
		if i.cur_result < len(i.results) {
			return true
		}
	}

	// Fetch new batches until we get results or the query is done.
	// The SDK may return empty batches (e.g. during auto-preparation
	// or when the read limit is hit), so we must loop.
	// Note: IsDone() checks continuationKey == nil, which is also true
	// for a fresh QueryRequest that has never been executed. We skip the
	// check only before the first Query() call.
	for {
		if i.started && i.req.IsDone() {
			return false
		}

		res, err := i.n.client.Query(i.req)
		if err != nil {
			// Retry with backoff on rate-limit errors
			if nosqlerr.Is(err, nosqlerr.ReadLimitExceeded, nosqlerr.WriteLimitExceeded, nosqlerr.RequestTimeout) {
				log.Println("rate limited in iterator, backing off:", err.Error())
				time.Sleep(2 * time.Second)
				continue
			}
			i.err = err
			log.Println("error in iterator:", err.Error())
			return false
		}
		i.started = true

		i.results, i.err = res.GetResults()
		if i.err != nil {
			log.Println("error in iterator:", i.err.Error())
			return false
		}

		if len(i.results) > 0 {
			i.cur_result = 0
			return true
		}
	}
}

func (i *Iterator) Valid() bool {
	_, okkey := i.results[i.cur_result].Get("key")
	_, okvalue := i.results[i.cur_result].Get("value")

	return i.err == nil && okkey && okvalue
}

func (i *Iterator) Key() string {
	key, _ := i.results[i.cur_result].Get("key")
	return key.(string)
}

func (i *Iterator) Value() any {
	value, _ := i.results[i.cur_result].Get("value")
	return value
}

func (i *Iterator) Err() error {
	return i.err
}
