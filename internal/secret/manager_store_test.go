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
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// fakeKind is a store kept apart from any cryptography, to check that the
// Manager goes through the Store interface only.
const fakeKind = "test-fake"

// noProvisionKind is a store that does not create safes.
const noProvisionKind = "test-no-provision"

var fakeUnlocks atomic.Int64

type fakeStore struct{}

func (fakeStore) Provision(_ context.Context, safe *happydns.Safe) error {
	safe.Keyring = []happydns.WrappedKeyset{{KEK: fakeKind}}
	return nil
}

func (fakeStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != fakeKind {
		return nil, errors.New("not a fake safe")
	}
	fakeUnlocks.Add(1)
	return fakeUnlocked{id: safe.Id}, nil
}

type fakeUnlocked struct{ id happydns.Identifier }

// The payload carries the associated data in clear, then the value: enough
// to tell a secret opened at the wrong place.
func (u fakeUnlocked) Seal(_ context.Context, sc SecretContext, clear []byte) ([]byte, error) {
	return append([]byte(hex.EncodeToString(AssociatedData(u.id, sc))+":"), clear...), nil
}

func (u fakeUnlocked) Open(_ context.Context, sc SecretContext, payload []byte) ([]byte, error) {
	ad, clear, ok := bytes.Cut(payload, []byte(":"))
	if !ok || string(ad) != hex.EncodeToString(AssociatedData(u.id, sc)) {
		return nil, errors.New("fake payload sealed elsewhere")
	}
	return clear, nil
}

func init() {
	builtinStores[fakeKind] = func(Config) (Store, error) { return fakeStore{}, nil }
	builtinStores[noProvisionKind] = nopFactory
}

func newTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager(%s): %v", cfg.Policy, err)
	}
	return m
}

func TestStorePolicySealsThroughItsStore(t *testing.T) {
	store := newMemSafeStorage()
	m := newTestManager(t, Config{Policy: Policy(fakeKind), Safes: store})
	sc := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("a"), Other: happydns.NewSecret("b")}
	before := fakeUnlocks.Load()
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	if n := fakeUnlocks.Load() - before; n != 1 {
		t.Errorf("the safe was unlocked %d times to seal one object, want 1", n)
	}

	safe, err := store.GetSafeByOwner(sc.Owner, fakeKind)
	if err != nil {
		t.Fatalf("no %s safe created: %v", fakeKind, err)
	}
	if len(safe.Keyring) != 1 || safe.Keyring[0].KEK != fakeKind || safe.CreatedAt.IsZero() {
		t.Errorf("created safe = %+v", safe)
	}

	for _, s := range []happydns.Secret{obj.ApiKey, obj.Other} {
		sv, err := ParseSealed(s.Token())
		if err != nil || !sv.SafeId.Equals(safe.Id) || !strings.HasSuffix(string(sv.Payload), ":"+s.Reveal()) {
			t.Errorf("token %q is not the fake payload in safe %s: %v", s.Token(), safe.Id.String(), err)
		}
	}

	// Opened by what the safe is, whatever the policy.
	plaintext := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store})
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token()), Other: sealedFromStorage(t, obj.Other.Token())}
	before = fakeUnlocks.Load()
	if err := plaintext.OpenObject(context.Background(), sc, stored); err != nil {
		t.Fatalf("OpenObject: %v", err)
	}
	if n := fakeUnlocks.Load() - before; n != 1 {
		t.Errorf("the safe was unlocked %d times to open one object, want 1", n)
	}
	if stored.ApiKey.Reveal() != "a" || stored.Other.Reveal() != "b" {
		t.Errorf("opened %q and %q, want a and b", stored.ApiKey.Reveal(), stored.Other.Reveal())
	}

	// Bound to its place, as an encrypting store binds it.
	moved := &managedObject{ApiKey: sealedFromStorage(t, obj.Other.Token())}
	if err := plaintext.OpenObject(context.Background(), sc, moved); err == nil {
		t.Error("a secret moved to another field opened")
	}
}

// Secrets of one object may sit in safes of several kinds, as when the
// policy changed between two of its updates.
func TestOpenGoesByTheKindOfEachSafe(t *testing.T) {
	key := testInstanceKey(t)
	store := newMemSafeStorage()
	instance := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	fake := newTestManager(t, Config{Policy: Policy(fakeKind), InstanceKey: key, Safes: store})
	sc := objectContext()

	a := &managedObject{ApiKey: happydns.NewSecret("a")}
	b := &managedObject{Other: happydns.NewSecret("b")}
	if err := instance.SealObject(context.Background(), sc, a); err != nil {
		t.Fatal(err)
	}
	if err := fake.SealObject(context.Background(), sc, b); err != nil {
		t.Fatal(err)
	}

	for _, m := range []*Manager{instance, fake} {
		stored := &managedObject{ApiKey: sealedFromStorage(t, a.ApiKey.Token()), Other: sealedFromStorage(t, b.Other.Token())}
		if err := m.OpenObject(context.Background(), sc, stored); err != nil {
			t.Fatalf("OpenObject under %s: %v", m.Policy(), err)
		}
		if stored.ApiKey.Reveal() != "a" || stored.Other.Reveal() != "b" {
			t.Errorf("under %s: opened %q and %q, want a and b", m.Policy(), stored.ApiKey.Reveal(), stored.Other.Reveal())
		}

		var counts Counts
		stored = &managedObject{ApiKey: sealedFromStorage(t, a.ApiKey.Token()), Other: sealedFromStorage(t, b.Other.Token())}
		if err := m.Inspect(context.Background(), sc, stored, &counts); err != nil {
			t.Fatalf("Inspect under %s: %v", m.Policy(), err)
		}
		if counts.Sealed[KindInstance] != 1 || counts.Sealed[fakeKind] != 1 || counts.Unreadable != 0 {
			t.Errorf("under %s: counts = %+v, want one sealed of each kind", m.Policy(), counts)
		}
	}
}

// A safe whose kind has no store here, unregistered or not configured, does
// not open, and says why.
func TestOpenSafeOfAnUnconfiguredKind(t *testing.T) {
	store := newMemSafeStorage()
	safe := happydns.Safe{Id: happydns.Identifier{0x0f}, Owner: objectContext().Owner, Kind: "test-unregistered"}
	if err := store.CreateSafe(&safe); err != nil {
		t.Fatal(err)
	}
	m := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store})
	sc := objectContext()

	token := FormatSealed(safe.Id, []byte("payload"))
	err := m.OpenObject(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, token)})
	if err == nil || !strings.Contains(err.Error(), "test-unregistered") {
		t.Errorf("OpenObject = %v, want an error naming the kind", err)
	}

	var counts Counts
	if err := m.Inspect(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, token)}, &counts); err != nil {
		t.Fatal(err)
	}
	if counts.Unreadable != 1 {
		t.Errorf("counts = %+v, want it unreadable", counts)
	}

	if err := m.CheckSafe(context.Background(), &safe); err == nil {
		t.Error("CheckSafe accepted a safe of an unconfigured kind")
	}

	fakeSafe := happydns.Safe{Id: happydns.Identifier{0x0e}, Owner: sc.Owner, Kind: fakeKind}
	if err := m.CheckSafe(context.Background(), &fakeSafe); err != nil {
		t.Errorf("CheckSafe refused a safe of a configured kind: %v", err)
	}
}

func TestNewManagerChecksThePolicyStore(t *testing.T) {
	for _, cfg := range []Config{
		{Policy: "test-unregistered", Safes: newMemSafeStorage()},
		{Policy: Policy(noProvisionKind), Safes: newMemSafeStorage()},
		{Policy: Policy(fakeKind)},
		{Policy: PolicyInstance, Safes: newMemSafeStorage()},
	} {
		if _, err := NewManager(cfg); err == nil {
			t.Errorf("NewManager(%+v) succeeded, want an error", cfg)
		}
	}
}
