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

package secret

import (
	"context"
	"slices"
	"testing"

	"git.happydns.org/happyDomain/model"
)

type nopStore struct{}

func (nopStore) Unlock(context.Context, *happydns.Safe) (UnlockedSafe, error) { return nil, nil }

func nopFactory(Config) (Store, error) { return nopStore{}, nil }

// The kind is persisted in every safe, and the policy names a kind of safe
// or plaintext: no store can be of an empty or the plaintext kind.
func TestBuiltinStoresKinds(t *testing.T) {
	for kind, factory := range builtinStores {
		if kind == "" || kind == string(PolicyPlaintext) {
			t.Errorf("builtin store of the reserved kind %q", kind)
		}
		if factory == nil {
			t.Errorf("builtin store %q without a factory", kind)
		}
	}
}

func TestKnownStores(t *testing.T) {
	kinds := KnownStores()
	if len(kinds) != len(builtinStores) || !slices.IsSorted(kinds) {
		t.Errorf("KnownStores() = %v, want every builtin kind, sorted", kinds)
	}
	for _, kind := range kinds {
		if _, ok := builtinStores[kind]; !ok {
			t.Errorf("KnownStores() holds %q, which is not builtin", kind)
		}
	}
}

func TestKnownStoresIsACopy(t *testing.T) {
	before := KnownStores()
	kinds := KnownStores()
	for i := range kinds {
		kinds[i] = "changed"
	}
	if !slices.Equal(KnownStores(), before) {
		t.Error("changing the returned list changed the known stores")
	}
}

func TestDefaultStores(t *testing.T) {
	got := DefaultStores()

	if !slices.Contains(got, string(PolicyPlaintext)) {
		t.Errorf("DefaultStores() = %v, want it to hold %q", got, PolicyPlaintext)
	}

	// Those reaching the files or programs of the server are enabled on
	// purpose only.
	optIn := []string{KindFile, KindProgram}
	for _, kind := range KnownStores() {
		if !slices.Contains(optIn, kind) && !slices.Contains(got, kind) {
			t.Errorf("DefaultStores() = %v, want it to hold the store %q", got, kind)
		}
	}
	for _, kind := range optIn {
		if slices.Contains(got, kind) {
			t.Errorf("DefaultStores() = %v, must not hold %q: it needs a configuration, and reaches the server", got, kind)
		}
	}
}

// A test adding a store wants it enabled, whatever stores cfg lists.
func TestWithStoresEnablesWhatItAdds(t *testing.T) {
	cfg := Config{Stores: []string{KindInstance}}.WithStores(map[string]StoreFactory{"test-nop": nopFactory})
	if !slices.Equal(cfg.Stores, []string{KindInstance, "test-nop"}) {
		t.Errorf("Stores = %v, want the instance store and the one added", cfg.Stores)
	}
	if again := cfg.WithStores(map[string]StoreFactory{"test-nop": nopFactory}); !slices.Equal(again.Stores, cfg.Stores) {
		t.Errorf("adding a store twice listed it twice: %v", again.Stores)
	}
	if _, ok := cfg.storeFactories()["test-nop"]; !ok {
		t.Error("the store added is not known")
	}

	if all := (Config{}).WithStores(map[string]StoreFactory{"test-nop": nopFactory}); all.Stores != nil {
		t.Errorf("Stores = %v, want nil, which already enables everything", all.Stores)
	}
}

// Configs derived from one base do not enable each other's stores.
func TestWithStoresLeavesTheBaseAlone(t *testing.T) {
	stores := make([]string, 1, 4)
	stores[0] = KindInstance
	base := Config{Stores: stores}

	a := base.WithStores(map[string]StoreFactory{"test-a": nopFactory})
	b := base.WithStores(map[string]StoreFactory{"test-b": nopFactory})

	if !slices.Equal(base.Stores, []string{KindInstance}) {
		t.Errorf("base Stores = %v, want it unchanged", base.Stores)
	}
	if !slices.Equal(a.Stores, []string{KindInstance, "test-a"}) {
		t.Errorf("Stores = %v, want the instance store and test-a only", a.Stores)
	}
	if !slices.Equal(b.Stores, []string{KindInstance, "test-b"}) {
		t.Errorf("Stores = %v, want the instance store and test-b only", b.Stores)
	}
}
