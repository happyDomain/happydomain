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
	"errors"
	"testing"
	"time"

	"github.com/oracle/nosql-go-sdk/nosqldb/types"
)

func row(key, value string) *types.MapValue {
	m := &types.MapValue{}
	m.Put("key", key)
	m.Put("value", value)
	return m
}

// Walking a batch already in hand must not issue any query — n is left nil so
// an unexpected fetch would panic rather than pass silently.
func TestIteratorAdvancesWithinBatch(t *testing.T) {
	it := &Iterator{results: []*types.MapValue{row("a", "1"), row("b", "2")}}

	if got := it.Key(); got != "a" {
		t.Fatalf("first key = %q, want %q", got, "a")
	}
	if !it.Next() {
		t.Fatal("Next() = false, want true on the second row of the batch")
	}
	if got := it.Key(); got != "b" {
		t.Fatalf("second key = %q, want %q", got, "b")
	}
}

// A failed iterator must never query again: reporting a partial result set as
// complete would silently truncate a backup.
func TestIteratorStaysFailed(t *testing.T) {
	it := &Iterator{
		results: []*types.MapValue{row("a", "1"), row("b", "2")},
		err:     errors.New("boom"),
	}

	if it.Next() {
		t.Error("Next() = true, want false once the iterator has failed")
	}
	if it.Valid() {
		t.Error("Valid() = true, want false once the iterator has failed")
	}
}

// The budget spans the iterator, so an exhausted deadline turns into an error
// instead of another retry.
func TestIteratorPauseGivesUpOnExhaustedBudget(t *testing.T) {
	it := &Iterator{deadline: time.Now().Add(-time.Second)}

	if it.pause("throttled") {
		t.Fatal("pause() = true, want false past the deadline")
	}
	if it.Err() == nil {
		t.Fatal("Err() = nil, want an error after the budget ran out")
	}
}

// The sleep is capped by the remaining budget, so a near deadline keeps this
// test fast while still exercising the cap on the backoff itself.
func TestIteratorPauseBacksOffUpToMax(t *testing.T) {
	it := &Iterator{deadline: time.Now().Add(10 * time.Millisecond), backoff: backoffMax}

	if !it.pause("throttled") {
		t.Fatal("pause() = false, want true while budget remains")
	}
	if it.backoff != backoffMax {
		t.Errorf("backoff = %s, want it capped at %s", it.backoff, backoffMax)
	}
}

func TestIteratorPauseGrowsBackoff(t *testing.T) {
	it := &Iterator{deadline: time.Now().Add(time.Millisecond)}

	it.pause("throttled")
	if it.backoff != backoffInitial {
		t.Fatalf("first backoff = %s, want %s", it.backoff, backoffInitial)
	}

	it.pause("throttled")
	if it.backoff != 2*backoffInitial {
		t.Fatalf("second backoff = %s, want %s", it.backoff, 2*backoffInitial)
	}
}

// Accessors are called by kvtpl around Valid(); none may panic on an iterator
// holding no row.
func TestIteratorAccessorsOnEmptyResults(t *testing.T) {
	it := &Iterator{}

	if it.Valid() {
		t.Error("Valid() = true, want false with no results")
	}
	if got := it.Key(); got != "" {
		t.Errorf("Key() = %q, want empty", got)
	}
	if got := it.Value(); got != nil {
		t.Errorf("Value() = %v, want nil", got)
	}
}
