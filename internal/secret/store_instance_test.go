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
	"errors"
	"slices"
	"sync"
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestInstanceStoreIsRegistered(t *testing.T) {
	if !slices.Contains(RegisteredStores(), KindInstance) {
		t.Errorf("RegisteredStores() = %v, want it to hold %q", RegisteredStores(), KindInstance)
	}
}

func testInstanceStore(t *testing.T, key *InstanceKey) Provisioner {
	t.Helper()
	f, ok := builtinStores[KindInstance]
	if !ok {
		t.Fatal("the instance store is not known")
	}
	s, err := f(Config{InstanceKey: key, Safes: newMemSafeStorage()})
	if err != nil {
		t.Fatalf("instance store: %v", err)
	}
	p, ok := s.(Provisioner)
	if !ok {
		t.Fatal("the instance store does not create safes")
	}
	return p
}

// Without a keyset, the instance store is not configured: its safes do not
// open, and none is created.
func TestInstanceStoreWithoutKey(t *testing.T) {
	f := builtinStores[KindInstance]
	if _, err := f(Config{Safes: newMemSafeStorage()}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("instance store without a keyset: got %v, want ErrStoreUnavailable", err)
	}
	if _, err := f(Config{InstanceKey: testInstanceKey(t)}); err == nil {
		t.Error("instance store without safe storage accepted")
	}
}

// A safe provisioned, then stored and read again, opens what it sealed, bound
// to the place of each secret.
func TestInstanceStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := testInstanceStore(t, key)
	safe := newTestSafe(t, happydns.Identifier{0x01})

	if err := store.Provision(ctx, safe); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if len(safe.Keyring) != 1 || safe.Keyring[0].KEK != KEKInstance {
		t.Fatalf("keyring = %+v, want one instance entry", safe.Keyring)
	}

	sc := objectContext()
	sc.Field = "apikey"

	u, err := store.Unlock(ctx, safe)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	payload, err := u.Seal(ctx, sc, []byte("v"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	again, err := testInstanceStore(t, key).Unlock(ctx, safe)
	if err != nil {
		t.Fatalf("Unlock again: %v", err)
	}
	if clear, err := again.Open(ctx, sc, payload); err != nil || string(clear) != "v" {
		t.Errorf("Open = %q, %v; want v", clear, err)
	}

	moved := sc
	moved.Field = "other"
	if _, err := again.Open(ctx, moved, payload); err == nil {
		t.Error("a payload opened for another field")
	}
}

func TestInstanceStoreRefusesOtherSafes(t *testing.T) {
	ctx := context.Background()
	store := testInstanceStore(t, testInstanceKey(t))
	safe := newTestSafe(t, happydns.Identifier{0x01})
	if err := store.Provision(ctx, safe); err != nil {
		t.Fatal(err)
	}

	other := *safe
	other.Kind = "password"
	if _, err := store.Unlock(ctx, &other); err == nil {
		t.Error("a safe of another kind unlocked")
	}

	if _, err := testInstanceStore(t, testInstanceKey(t)).Unlock(ctx, safe); err == nil {
		t.Error("another instance keyset unlocked the safe")
	}
}

func TestInstanceSafeCreatedOnceUnderConcurrency(t *testing.T) {
	m, _, store := instanceManagers(t)
	sc := objectContext()
	sc.Field = "apikey"

	const n = 32
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := happydns.NewSecret("v")
			errs[i] = m.SealSecret(context.Background(), sc, &s)
		}()
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("SealSecret: %v", errs[i])
		}
	}
	if store.creates != 1 {
		t.Errorf("%d safes created, want 1", store.creates)
	}

	safe, err := store.GetSafeByOwner(sc.Owner, KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	if len(safe.Keyring) != 1 || safe.Keyring[0].KEK != KEKInstance || safe.CreatedAt.IsZero() {
		t.Errorf("created safe = %+v", safe)
	}
}

func TestWrapUnwrapRoundTrip(t *testing.T) {
	key := testInstanceKey(t)
	safe := newTestSafe(t, happydns.Identifier{0x01})

	dek, _ := GenerateInstanceKeyset()
	ct := encryptWith(t, dek, "field value")

	w, err := key.Wrap(safe, dek)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if w.KEK != KEKInstance {
		t.Errorf("KEK = %q, want %q", w.KEK, KEKInstance)
	}
	safe.Keyring = []happydns.WrappedKeyset{w}

	back, err := key.Unwrap(safe)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !decryptsWith(back, ct, "field value") {
		t.Error("the unwrapped keyset does not open what the original sealed")
	}
}

func TestUnwrapIsBoundToItsSafe(t *testing.T) {
	key := testInstanceKey(t)
	owner := happydns.Identifier{0x01}
	safe := newTestSafe(t, owner)

	dek, _ := GenerateInstanceKeyset()
	w, err := key.Wrap(safe, dek)
	if err != nil {
		t.Fatal(err)
	}

	// Moved to another safe of the same owner.
	other := newTestSafe(t, owner)
	other.Keyring = []happydns.WrappedKeyset{w}
	if _, err := key.Unwrap(other); err == nil {
		t.Error("a keyring copied to another safe opened")
	}

	// Same safe identifier, another owner.
	stolen := *safe
	stolen.Owner = happydns.Identifier{0x02}
	stolen.Keyring = []happydns.WrappedKeyset{w}
	if _, err := key.Unwrap(&stolen); err == nil {
		t.Error("a keyring given to another owner opened")
	}
}

func TestUnwrapWithoutInstanceEntry(t *testing.T) {
	key := testInstanceKey(t)
	safe := newTestSafe(t, happydns.Identifier{0x01})
	safe.Keyring = []happydns.WrappedKeyset{{KEK: "password", Blob: []byte{1}}}

	if _, err := key.Unwrap(safe); err == nil {
		t.Error("Unwrap without an instance entry succeeded")
	}
}

func TestUnwrapRefusesWrongKeyAndTamperedBlob(t *testing.T) {
	key := testInstanceKey(t)
	safe := newTestSafe(t, happydns.Identifier{0x01})

	dek, _ := GenerateInstanceKeyset()
	w, err := key.Wrap(safe, dek)
	if err != nil {
		t.Fatal(err)
	}
	safe.Keyring = []happydns.WrappedKeyset{w}

	if _, err := testInstanceKey(t).Unwrap(safe); err == nil {
		t.Error("another instance keyset unwrapped the safe")
	}

	// Byte 10 is past the protobuf tag and length, inside the ciphertext.
	blob := append([]byte(nil), w.Blob...)
	blob[10] ^= 0x01
	tampered := *safe
	tampered.Keyring = []happydns.WrappedKeyset{{KEK: KEKInstance, Blob: blob}}
	if _, err := key.Unwrap(&tampered); err == nil {
		t.Error("a tampered wrapped keyset unwrapped")
	}
}

func TestUnwrapAfterInstanceRotation(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)
	safe := newTestSafe(t, happydns.Identifier{0x01})

	dek, _ := GenerateInstanceKeyset()
	ct := encryptWith(t, dek, "v")
	w, _ := key.Wrap(safe, dek)
	safe.Keyring = []happydns.WrappedKeyset{w}

	rotated, _ := RotateInstanceKeyset(h)
	rkey, _ := NewInstanceKey(rotated)

	back, err := rkey.Unwrap(safe)
	if err != nil {
		t.Fatalf("Unwrap after rotation: %v", err)
	}
	if !decryptsWith(back, ct, "v") {
		t.Error("the keyset unwrapped after rotation differs")
	}
}
