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
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	happydns "git.happydns.org/happyDomain/model"
)

func testInstanceKey(t *testing.T) *InstanceKey {
	t.Helper()
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	k, err := NewInstanceKey(h)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func newTestSafe(t *testing.T, owner happydns.Identifier) *happydns.Safe {
	t.Helper()
	id, _ := happydns.NewRandomIdentifier()
	return &happydns.Safe{Id: id, Owner: owner, Kind: KindInstance}
}

func encryptWith(t *testing.T, h *keyset.Handle, pt string) []byte {
	t.Helper()
	a, err := aead.New(h)
	if err != nil {
		t.Fatal(err)
	}
	ct, err := a.Encrypt([]byte(pt), []byte("ad"))
	if err != nil {
		t.Fatal(err)
	}
	return ct
}

func decryptsWith(h *keyset.Handle, ct []byte, pt string) bool {
	a, err := aead.New(h)
	if err != nil {
		return false
	}
	got, err := a.Decrypt(ct, []byte("ad"))
	return err == nil && string(got) == pt
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

func TestInstanceSafeCreatedOnceUnderConcurrency(t *testing.T) {
	store := secrettest.NewSafes()
	r := newSafeRegistry(store, testInstanceKey(t))
	owner := happydns.Identifier{0x01}

	const n = 32
	ids := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			safe, err := r.instanceSafe(owner)
			errs[i] = err
			if err == nil {
				ids[i] = safe.Id.String()
			}
		}()
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("instanceSafe: %v", errs[i])
		}
		if ids[i] != ids[0] {
			t.Fatalf("two safes returned: %s and %s", ids[0], ids[i])
		}
	}
	if store.Creates() != 1 {
		t.Errorf("%d safes created, want 1", store.Creates())
	}

	safe, _ := store.GetSafe(happydns.Identifier(mustDecodeId(t, ids[0])))
	if safe.Kind != KindInstance || len(safe.Keyring) != 1 || safe.Keyring[0].KEK != KEKInstance || safe.CreatedAt.IsZero() {
		t.Errorf("created safe = %+v", safe)
	}
}

func TestInstanceSafeWithoutKey(t *testing.T) {
	r := newSafeRegistry(secrettest.NewSafes(), nil)
	if _, err := r.instanceSafe(happydns.Identifier{0x01}); err == nil {
		t.Error("an instance safe was created without an instance key")
	}
}

func mustDecodeId(t *testing.T, s string) happydns.Identifier {
	t.Helper()
	id, err := happydns.NewIdentifierFromString(s)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// A committed safe, wrapped by the committed instance keyset, opens a
// committed ciphertext: a change to how keysets are wrapped fails here.
func TestWrappedKeysetGolden(t *testing.T) {
	key, err := LoadInstanceKey("testdata/instance-keyset.json")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("testdata/safe.json")
	if err != nil {
		t.Fatal(err)
	}
	var safe happydns.Safe
	if err := json.Unmarshal(raw, &safe); err != nil {
		t.Fatal(err)
	}

	dek, err := key.Unwrap(&safe)
	if err != nil {
		t.Fatalf("Unwrap(committed safe): %v", err)
	}

	ctB64, err := os.ReadFile("testdata/safe-ciphertext.b64")
	if err != nil {
		t.Fatal(err)
	}
	ct, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(ctB64)))
	if err != nil {
		t.Fatal(err)
	}

	if !decryptsWith(dek, ct, "golden field value") {
		t.Error("the committed safe does not open the committed ciphertext")
	}
}

func TestStartupCheckRefusesOrphanedInstanceSafes(t *testing.T) {
	store := newMemStartupStorage()

	if err := StartupCheck(PolicyPlaintext, nil, store); err != nil {
		t.Fatalf("no keyset, no safe: %v", err)
	}

	if err := store.CreateSafe(newTestSafe(t, happydns.Identifier{0x01})); err != nil {
		t.Fatal(err)
	}
	if err := StartupCheck(PolicyPlaintext, nil, store); err == nil {
		t.Error("instance safes exist but no keyset is configured: want a refusal")
	}
}

type memStartupStorage struct {
	*memCheckStorage
	*secrettest.Safes
}

func newMemStartupStorage() *memStartupStorage {
	return &memStartupStorage{memCheckStorage: &memCheckStorage{}, Safes: secrettest.NewSafes()}
}

// Without its check record, a keyset is only accepted if it opens the instance
// safes already stored: otherwise startup would record a wrong keyset as the
// right one, and refuse the right one at the next start.
func TestStartupCheckWithoutCheckRecordVerifiesSafes(t *testing.T) {
	right := testInstanceKey(t)
	wrong := testInstanceKey(t)

	store := newMemStartupStorage()
	if _, err := newSafeRegistry(store, right).instanceSafe(happydns.Identifier{0x01}); err != nil {
		t.Fatal(err)
	}

	if err := StartupCheck(PolicyInstance, wrong, store); !errors.Is(err, ErrWrongInstanceKey) {
		t.Fatalf("StartupCheck with a keyset opening no safe = %v, want ErrWrongInstanceKey", err)
	}
	if _, err := store.GetSecretCheck(); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("a check record was written for the wrong keyset: %v", err)
	}

	if err := StartupCheck(PolicyInstance, right, store); err != nil {
		t.Fatalf("StartupCheck with the right keyset: %v", err)
	}
	if _, err := store.GetSecretCheck(); err != nil {
		t.Errorf("no check record after accepting the right keyset: %v", err)
	}
}

// racingSafeStorage has another process create the safe of the owner between
// the lookup and the creation made by instanceSafe.
type racingSafeStorage struct {
	*secrettest.Safes
	winner *happydns.Safe
}

func (r *racingSafeStorage) CreateSafe(safe *happydns.Safe) error {
	if r.winner != nil {
		if err := r.Safes.CreateSafe(r.winner); err != nil {
			return err
		}
		r.winner = nil
	}
	return r.Safes.CreateSafe(safe)
}

func TestInstanceSafeReusesASafeCreatedConcurrently(t *testing.T) {
	owner := happydns.Identifier{0x01}
	winner := newTestSafe(t, owner)
	store := &racingSafeStorage{Safes: secrettest.NewSafes(), winner: winner}

	got, err := newSafeRegistry(store, testInstanceKey(t)).instanceSafe(owner)
	if err != nil {
		t.Fatalf("instanceSafe after losing the creation race: %v", err)
	}
	if !got.Id.Equals(winner.Id) {
		t.Errorf("instanceSafe = %s, want the safe created concurrently %s", got.Id.String(), winner.Id.String())
	}
}

// The key of a safe is unwrapped once, not on every seal or open.
func TestAeadIsCachedPerSafe(t *testing.T) {
	store := secrettest.NewSafes()
	r := newSafeRegistry(store, testInstanceKey(t))
	safe, err := r.instanceSafe(happydns.Identifier{0x01})
	if err != nil {
		t.Fatal(err)
	}

	first, err := r.aead(safe)
	if err != nil {
		t.Fatal(err)
	}
	// As read again from storage, the way open finds it.
	again, _ := store.GetSafe(safe.Id)
	second, err := r.aead(again)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("the key of the safe was unwrapped again")
	}
}

// A safe whose wrapped key changed (restored from a backup, rewrapped) is
// unwrapped again: the cached primitive may hold a key it no longer carries.
func TestAeadCacheFollowsTheWrappedKey(t *testing.T) {
	key := testInstanceKey(t)
	r := newSafeRegistry(secrettest.NewSafes(), key)
	safe, err := r.instanceSafe(happydns.Identifier{0x01})
	if err != nil {
		t.Fatal(err)
	}

	old, err := r.aead(safe)
	if err != nil {
		t.Fatal(err)
	}
	ct, err := old.Encrypt([]byte("value"), []byte("ad"))
	if err != nil {
		t.Fatal(err)
	}

	dek, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		t.Fatal(err)
	}
	w, err := key.Wrap(safe, dek)
	if err != nil {
		t.Fatal(err)
	}
	replaced := *safe
	replaced.Keyring = []happydns.WrappedKeyset{w}

	p, err := r.aead(&replaced)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Decrypt(ct, []byte("ad")); err == nil {
		t.Error("the primitive of the former key was used for the replaced one")
	}
}

// The wrapped key is bound to its owner: the cache must not let a safe whose
// owner changed open with the key unwrapped for the former owner.
func TestAeadCacheChecksTheOwner(t *testing.T) {
	r := newSafeRegistry(secrettest.NewSafes(), testInstanceKey(t))
	safe, err := r.instanceSafe(happydns.Identifier{0x01})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.aead(safe); err != nil {
		t.Fatal(err)
	}

	moved := *safe
	moved.Owner = happydns.Identifier{0x02}
	if _, err := r.aead(&moved); err == nil {
		t.Error("a safe moved to another owner opened with the cached key")
	}
}
