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

// Package storagetest holds the tests every storage backend must pass.
package storagetest

import (
	"encoding/json"
	"sync"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
)

// TestConditionalWrites checks PutIfAbsent and PutIfUnchanged on fresh
// stores returned by open.
func TestConditionalWrites(t *testing.T, open func(t *testing.T) storage.KVStorage) {
	newStore := func(t *testing.T) storage.KVStorage {
		s := open(t)
		t.Cleanup(func() { s.Close() })
		return s
	}

	t.Run("PutIfAbsent", func(t *testing.T) {
		s := newStore(t)

		if stored, err := s.PutIfAbsent("k", "first"); err != nil || !stored {
			t.Fatalf("PutIfAbsent on a free key = %v, %v; want true, nil", stored, err)
		}
		if stored, err := s.PutIfAbsent("k", "second"); err != nil || stored {
			t.Fatalf("PutIfAbsent on a taken key = %v, %v; want false, nil", stored, err)
		}

		var got string
		if err := s.Get("k", &got); err != nil || got != "first" {
			t.Errorf("Get = %q, %v; want the first value kept", got, err)
		}
	})

	t.Run("PutIfAbsentUnderConcurrency", func(t *testing.T) {
		s := newStore(t)

		onlyOneStores(t, "PutIfAbsent", func(i int) (bool, error) {
			return s.PutIfAbsent("k", i)
		})
	})

	t.Run("PutIfUnchanged", func(t *testing.T) {
		s := newStore(t)

		if err := s.Put("k", map[string]string{"v": "first"}); err != nil {
			t.Fatal(err)
		}
		previous := read(t, s, "k")

		if stored, err := s.PutIfUnchanged("k", previous, map[string]string{"v": "second"}); err != nil || !stored {
			t.Fatalf("PutIfUnchanged on an unchanged key = %v, %v; want true, nil", stored, err)
		}
		var got map[string]string
		if err := s.Get("k", &got); err != nil || got["v"] != "second" {
			t.Errorf("Get = %v, %v; want the new value", got, err)
		}

		// previous is now outdated.
		if stored, err := s.PutIfUnchanged("k", previous, map[string]string{"v": "third"}); err != nil || stored {
			t.Fatalf("PutIfUnchanged on a changed key = %v, %v; want false, nil", stored, err)
		}
		if err := s.Get("k", &got); err != nil || got["v"] != "second" {
			t.Errorf("Get = %v, %v; want the concurrent value kept", got, err)
		}
	})

	t.Run("PutIfUnchangedDoesNotBringBackADeletedKey", func(t *testing.T) {
		s := newStore(t)

		if err := s.Put("k", "v"); err != nil {
			t.Fatal(err)
		}
		previous := read(t, s, "k")
		if err := s.Delete("k"); err != nil {
			t.Fatal(err)
		}

		if stored, err := s.PutIfUnchanged("k", previous, "back"); err != nil || stored {
			t.Fatalf("PutIfUnchanged on a deleted key = %v, %v; want false, nil", stored, err)
		}
		if exists, _ := s.Has("k"); exists {
			t.Error("a deleted key was stored again")
		}

		if stored, err := s.PutIfUnchanged("never", json.RawMessage(`"v"`), "new"); err != nil || stored {
			t.Errorf("PutIfUnchanged on a key that never existed = %v, %v; want false, nil", stored, err)
		}
	})

	t.Run("PutIfUnchangedUnderConcurrency", func(t *testing.T) {
		s := newStore(t)

		if err := s.Put("k", 0); err != nil {
			t.Fatal(err)
		}
		previous := read(t, s, "k")

		onlyOneStores(t, "PutIfUnchanged", func(i int) (bool, error) {
			return s.PutIfUnchanged("k", previous, i+1)
		})
	})
}

// read returns what key holds, as PutIfUnchanged expects it.
func read(t *testing.T, s storage.KVStorage, key string) json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	if err := s.Get(key, &raw); err != nil {
		t.Fatalf("Get(%q): %v", key, err)
	}
	return raw
}

// onlyOneStores runs put from many goroutines at once, and checks that
// exactly one of them reports it stored its value.
func onlyOneStores(t *testing.T, name string, put func(i int) (bool, error)) {
	t.Helper()

	const n = 32
	stored := make([]bool, n)
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := put(i)
			if err != nil {
				t.Errorf("%s: %v", name, err)
			}
			stored[i] = ok
		}()
	}
	close(start)
	wg.Wait()

	winners := 0
	for _, ok := range stored {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("%d concurrent %s stored their value, want 1", winners, name)
	}
}
