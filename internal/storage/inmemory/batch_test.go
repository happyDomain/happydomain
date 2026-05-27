// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

package inmemory

import (
	"math"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func newStore(t *testing.T) *InMemoryStorage {
	t.Helper()
	s, err := NewInMemoryStorage()
	if err != nil {
		t.Fatalf("NewInMemoryStorage: %v", err)
	}
	return s
}

func TestBatchEmptyCommitIsNoop(t *testing.T) {
	s := newStore(t)
	if err := s.NewBatch().Commit(); err != nil {
		t.Fatalf("empty Commit: %v", err)
	}
	if len(s.data) != 0 {
		t.Fatalf("empty batch left %d entries behind", len(s.data))
	}
}

func TestBatchMixedPutDelete(t *testing.T) {
	s := newStore(t)
	// Seed a value that the batch will delete.
	if err := s.Put("seed", "old"); err != nil {
		t.Fatal(err)
	}

	b := s.NewBatch()
	if err := b.Put("a", "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := b.Put("b", "beta"); err != nil {
		t.Fatal(err)
	}
	b.Delete("seed")
	if err := b.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var got string
	if err := s.Get("a", &got); err != nil || got != "alpha" {
		t.Errorf("a: got (%q, %v), want (\"alpha\", nil)", got, err)
	}
	if err := s.Get("b", &got); err != nil || got != "beta" {
		t.Errorf("b: got (%q, %v), want (\"beta\", nil)", got, err)
	}
	if err := s.Get("seed", &got); err != happydns.ErrNotFound {
		t.Errorf("seed should be deleted, got err=%v", err)
	}
}

func TestBatchPutMarshalFailureKeepsStateUntouched(t *testing.T) {
	s := newStore(t)
	if err := s.Put("untouched", "keep"); err != nil {
		t.Fatal(err)
	}

	b := s.NewBatch()
	if err := b.Put("ok", "fine"); err != nil {
		t.Fatal(err)
	}
	// NaN cannot be JSON-encoded.
	if err := b.Put("bad", math.NaN()); err == nil {
		t.Fatalf("expected Put with NaN to fail, got nil")
	}

	// Even though one Put failed, the others were staged. Caller policy is
	// to abort the batch; verify by NOT calling Commit and confirming the
	// store is unchanged.
	var got string
	if err := s.Get("untouched", &got); err != nil || got != "keep" {
		t.Errorf("untouched: got (%q, %v)", got, err)
	}
	if _, err := s.Has("ok"); err != nil {
		t.Errorf("Has(ok): %v", err)
	} else if exists, _ := s.Has("ok"); exists {
		t.Errorf("ok should not exist before Commit")
	}
}

func TestBatchCommitOrderObservedAtomically(t *testing.T) {
	s := newStore(t)

	first := s.NewBatch()
	if err := first.Put("k", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}

	second := s.NewBatch()
	if err := second.Put("k", "v2"); err != nil {
		t.Fatal(err)
	}
	second.Delete("k")
	if err := second.Commit(); err != nil {
		t.Fatal(err)
	}

	// Within a single batch, ops apply in staging order; the Delete here
	// follows the Put, so k must be absent after Commit.
	var got string
	if err := s.Get("k", &got); err != happydns.ErrNotFound {
		t.Errorf("expected ErrNotFound after Put+Delete in same batch, got err=%v val=%q", err, got)
	}
}
