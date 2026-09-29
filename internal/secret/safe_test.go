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
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

// memSafeStorage keeps safes in memory, enforcing one safe of a kind per
// owner as the real storage does.
type memSafeStorage struct {
	mu      sync.Mutex
	safes   map[string]happydns.Safe
	creates int
}

func newMemSafeStorage() *memSafeStorage {
	return &memSafeStorage{safes: map[string]happydns.Safe{}}
}

func (m *memSafeStorage) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*happydns.Safe
	for _, s := range m.safes {
		c := s
		all = append(all, &c)
	}
	return &sliceIterator[happydns.Safe]{items: all, idx: -1}, nil
}

func (m *memSafeStorage) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.safes[id.String()]
	if !ok {
		return nil, happydns.ErrSafeNotFound
	}
	return &s, nil
}

func (m *memSafeStorage) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.safes {
		if s.Owner.Equals(owner) && s.Kind == kind {
			c := s
			return &c, nil
		}
	}
	return nil, happydns.ErrSafeNotFound
}

func (m *memSafeStorage) CreateSafe(safe *happydns.Safe) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.safes {
		if s.Owner.Equals(safe.Owner) && s.Kind == safe.Kind {
			return fmt.Errorf("owner already has a %s safe", safe.Kind)
		}
	}
	m.safes[safe.Id.String()] = *safe
	m.creates++
	return nil
}

func (m *memSafeStorage) UpdateSafe(safe *happydns.Safe) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.safes[safe.Id.String()]; !ok {
		return happydns.ErrSafeNotFound
	}
	m.safes[safe.Id.String()] = *safe
	return nil
}

func (m *memSafeStorage) DeleteSafe(id happydns.Identifier) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.safes, id.String())
	return nil
}

type sliceIterator[T any] struct {
	items []*T
	idx   int
}

func (it *sliceIterator[T]) Next() bool          { it.idx++; return it.idx < len(it.items) }
func (it *sliceIterator[T]) NextWithError() bool { return it.Next() }
func (it *sliceIterator[T]) Item() *T            { return it.items[it.idx] }
func (it *sliceIterator[T]) DropItem() error     { return nil }
func (it *sliceIterator[T]) Key() string         { return "" }
func (it *sliceIterator[T]) Raw() any            { return nil }
func (it *sliceIterator[T]) Err() error          { return nil }
func (it *sliceIterator[T]) Close()              {}

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

	rotated, _ := rotateInstanceKeyset(h)
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
	store := newMemSafeStorage()
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
	if store.creates != 1 {
		t.Errorf("%d safes created, want 1", store.creates)
	}

	safe, _ := store.GetSafe(happydns.Identifier(mustDecodeId(t, ids[0])))
	if safe.Kind != KindInstance || len(safe.Keyring) != 1 || safe.Keyring[0].KEK != KEKInstance || safe.CreatedAt.IsZero() {
		t.Errorf("created safe = %+v", safe)
	}
}

func TestInstanceSafeWithoutKey(t *testing.T) {
	r := newSafeRegistry(newMemSafeStorage(), nil)
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
	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}

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
	*memSafeStorage
}
