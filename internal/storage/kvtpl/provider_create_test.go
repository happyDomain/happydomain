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

package database_test

import (
	"errors"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	happydns "git.happydns.org/happyDomain/model"
)

func TestCreateProviderWithPresetId(t *testing.T) {
	s := newStorage(t)

	owner, id := newIdentifier(t), newIdentifier(t)
	prvd := stubProvider(id, owner, "value")

	if err := s.CreateProvider(prvd); err != nil {
		t.Fatalf("CreateProvider with a preset id: %v", err)
	}
	if !prvd.Id.Equals(id) {
		t.Errorf("CreateProvider replaced the preset id %s by %s", id.String(), prvd.Id.String())
	}

	got, err := s.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider(preset id): %v", err)
	}
	if !got.Owner.Equals(owner) {
		t.Errorf("stored owner = %s, want %s", got.Owner.String(), owner.String())
	}

	providers, err := s.ListProviders(&happydns.User{Id: owner})
	if err != nil || len(providers) != 1 {
		t.Errorf("ListProviders = %d, %v; want 1 provider", len(providers), err)
	}
}

// Checking that the identifier is free, then writing, leaves a window where
// a concurrent creation of the same identifier lands: the write itself has to
// refuse an identifier already taken, even when racingKV hides it from Has.
func TestCreateProviderRefusesTakenId(t *testing.T) {
	for name, newS := range map[string]func(*testing.T) storage.Storage{
		"plain": newStorage,
		"racing": func(t *testing.T) storage.Storage {
			s, _ := storageOver(t, func(raw storage.KVStorage) storage.KVStorage { return racingKV{raw} })
			return s
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := newS(t)

			owner, other, id := newIdentifier(t), newIdentifier(t), newIdentifier(t)

			first := stubProvider(id, owner, "first")
			if err := s.CreateProvider(first); err != nil {
				t.Fatalf("CreateProvider: %v", err)
			}

			second := stubProvider(append(happydns.Identifier(nil), id...), other, "second")
			if err := s.CreateProvider(second); !errors.Is(err, happydns.ErrAlreadyExists) {
				t.Fatalf("CreateProvider with an id already used = %v, want ErrAlreadyExists", err)
			}

			got, err := s.GetProvider(id)
			if err != nil {
				t.Fatalf("GetProvider: %v", err)
			}
			if !got.Owner.Equals(owner) {
				t.Error("the first provider was overwritten")
			}
			if providers, _ := s.ListProviders(&happydns.User{Id: other}); len(providers) != 0 {
				t.Error("a failed creation left an owner index entry")
			}
		})
	}
}

// The storage never chooses the identifier, and refuses one that is missing or
// malformed: it cannot tell a client-supplied identifier from a generated one,
// but it can at least keep keys well formed.
func TestCreateProviderRefusesInvalidId(t *testing.T) {
	for name, id := range map[string]happydns.Identifier{
		"missing":   nil,
		"too short": {0x01},
		"too long":  make(happydns.Identifier, happydns.IDENTIFIER_LEN+1),
	} {
		t.Run(name, func(t *testing.T) {
			s := newStorage(t)
			owner := newIdentifier(t)
			prvd := stubProvider(id, owner, "value")

			if err := s.CreateProvider(prvd); !errors.Is(err, happydns.ErrInvalidIdentifier) {
				t.Fatalf("CreateProvider = %v, want ErrInvalidIdentifier", err)
			}
			if providers, _ := s.ListProviders(&happydns.User{Id: owner}); len(providers) != 0 {
				t.Error("a refused creation stored something")
			}
		})
	}
}

// racingKV answers Has as if the key did not exist yet, as it would for a
// creation checking right before a concurrent one stores the same key.
type racingKV struct {
	storage.KVStorage
}

func (racingKV) Has(string) (bool, error) { return false, nil }

// failingIndexKV refuses to write any key starting with prefix.
type failingIndexKV struct {
	storage.KVStorage
	prefix string
}

var errIndexWrite = errors.New("index write refused")

func (f failingIndexKV) Put(key string, v any) error {
	if strings.HasPrefix(key, f.prefix) {
		return errIndexWrite
	}
	return f.KVStorage.Put(key, v)
}

func (f failingIndexKV) NewBatch() storage.Batch {
	return failingIndexBatch{f.KVStorage.NewBatch(), f.prefix}
}

type failingIndexBatch struct {
	storage.Batch
	prefix string
}

func (b failingIndexBatch) Put(key string, v any) error {
	if strings.HasPrefix(key, b.prefix) {
		return errIndexWrite
	}
	return b.Batch.Put(key, v)
}

func storageOver(t *testing.T, wrap func(storage.KVStorage) storage.KVStorage) (storage.Storage, storage.KVStorage) {
	t.Helper()
	raw, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	s, err := kv.NewKVDatabase(wrap(raw))
	if err != nil {
		t.Fatal(err)
	}
	return s, raw
}

func newIdentifier(t *testing.T) happydns.Identifier {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func stubProvider(id, owner happydns.Identifier, field string) *happydns.Provider {
	return &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: id, Owner: owner},
		Provider:     stubProviderBody{Field: field},
	}
}

// A provider whose owner index could not be written would be stored but
// listed nowhere: the creation fails and leaves nothing behind.
func TestCreateProviderLeavesNothingWhenIndexFails(t *testing.T) {
	s, raw := storageOver(t, func(raw storage.KVStorage) storage.KVStorage {
		return failingIndexKV{raw, "provider.owner|"}
	})

	owner, id := newIdentifier(t), newIdentifier(t)
	prvd := stubProvider(id, owner, "value")
	if err := s.CreateProvider(prvd); !errors.Is(err, errIndexWrite) {
		t.Fatalf("CreateProvider = %v, want the index write error", err)
	}

	iter := raw.Search("provider")
	defer iter.Release()
	for iter.Next() {
		t.Errorf("key %q left behind", iter.Key())
	}
}
