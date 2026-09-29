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
	mu    sync.Mutex
	safes map[string]happydns.Safe

	// damaged holds the records that do not decode, by key: listings
	// report them as the KV storage does.
	damaged map[string]error

	creates int
}

func newMemSafeStorage() *memSafeStorage {
	return &memSafeStorage{safes: map[string]happydns.Safe{}, damaged: map[string]error{}}
}

func (m *memSafeStorage) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []sliceEntry[happydns.Safe]
	for key, s := range m.safes {
		c := s
		all = append(all, sliceEntry[happydns.Safe]{key: key, item: &c})
	}
	for key, err := range m.damaged {
		all = append(all, sliceEntry[happydns.Safe]{key: key, err: err})
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

type sliceEntry[T any] struct {
	key  string
	item *T

	// err makes the entry a record that does not decode.
	err error
}

// sliceIterator behaves as the KV iterator: Next skips the records that do
// not decode, yet keeps their error for Err; NextWithError stops on them.
type sliceIterator[T any] struct {
	items []sliceEntry[T]
	idx   int
	err   error
}

func (it *sliceIterator[T]) Next() bool {
	for it.idx++; it.idx < len(it.items); it.idx++ {
		if it.items[it.idx].err == nil {
			return true
		}
		it.err = it.items[it.idx].err
	}
	return false
}

func (it *sliceIterator[T]) NextWithError() bool {
	it.idx++
	if it.idx >= len(it.items) {
		it.err = nil
		return false
	}
	it.err = it.items[it.idx].err
	return true
}

func (it *sliceIterator[T]) Item() *T        { return it.items[it.idx].item }
func (it *sliceIterator[T]) DropItem() error { return nil }
func (it *sliceIterator[T]) Key() string     { return it.items[it.idx].key }
func (it *sliceIterator[T]) Raw() any        { return nil }
func (it *sliceIterator[T]) Err() error      { return it.err }
func (it *sliceIterator[T]) Close()          {}

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

// A safe fetched again from storage opens what it sealed when created.
func TestSafeRegistryAEAD(t *testing.T) {
	r := newSafeRegistry(newMemSafeStorage(), testInstanceKey(t))
	owner := happydns.Identifier{0x01}

	safe, err := r.instanceSafe(owner)
	if err != nil {
		t.Fatal(err)
	}
	a, err := r.aead(safe)
	if err != nil {
		t.Fatalf("aead: %v", err)
	}
	ct, err := a.Encrypt([]byte("v"), []byte("ad"))
	if err != nil {
		t.Fatal(err)
	}

	again, err := r.instanceSafe(owner)
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.aead(again)
	if err != nil {
		t.Fatalf("aead(stored safe): %v", err)
	}
	if pt, err := b.Decrypt(ct, []byte("ad")); err != nil || string(pt) != "v" {
		t.Errorf("the stored safe does not open what it sealed: %q, %v", pt, err)
	}

	other := *safe
	other.Kind = "password"
	if _, err := r.aead(&other); err == nil {
		t.Error("a safe of an unsupported kind opened")
	}

	if _, err := newSafeRegistry(newMemSafeStorage(), nil).aead(safe); err == nil {
		t.Error("an instance safe opened without an instance keyset")
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

func TestEachSafe(t *testing.T) {
	store := newMemSafeStorage()
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := store.CreateSafe(newTestSafe(t, owner)); err != nil {
			t.Fatal(err)
		}
	}
	decodeErr := errors.New("does not decode")
	store.damaged["safe-broken"] = decodeErr

	var seen []happydns.Identifier
	damaged := map[string]error{}
	err := eachSafe(store, func(s *happydns.Safe) {
		seen = append(seen, s.Owner)
	}, func(key string, err error) {
		damaged[key] = err
	})
	if err != nil {
		t.Fatalf("eachSafe: %v", err)
	}
	if len(seen) != 2 {
		t.Errorf("fn called on %v, want both safes", seen)
	}
	if len(damaged) != 1 || damaged["safe-broken"] != decodeErr {
		t.Errorf("damaged called with %v, want the broken record and its error", damaged)
	}

	// Without a damaged callback, the record is skipped all the same.
	n := 0
	if err := eachSafe(store, func(*happydns.Safe) { n++ }, nil); err != nil || n != 2 {
		t.Errorf("eachSafe without damaged = %d safes, %v; want 2, nil", n, err)
	}
}

type unlistableSafeStorage struct{ *memSafeStorage }

func (unlistableSafeStorage) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	return nil, errors.New("storage down")
}

func TestEachSafeReportsListingError(t *testing.T) {
	store := unlistableSafeStorage{newMemSafeStorage()}
	if err := eachSafe(store, func(*happydns.Safe) { t.Error("fn called") }, nil); err == nil {
		t.Error("eachSafe on a store that cannot list: want its error")
	}
}

// A safe record that does not decode tells no kind: it neither keeps
// happyDomain from starting, nor hides an instance safe listed after it.
func TestStartupCheckSkipsDamagedSafeRecords(t *testing.T) {
	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}
	store.damaged["safe-broken"] = errors.New("does not decode")

	if err := StartupCheck(PolicyPlaintext, nil, store); err != nil {
		t.Fatalf("no keyset, a damaged record only: %v", err)
	}

	if err := store.CreateSafe(newTestSafe(t, happydns.Identifier{0x01})); err != nil {
		t.Fatal(err)
	}
	if err := StartupCheck(PolicyPlaintext, nil, store); err == nil {
		t.Error("an instance safe exists beside a damaged record, but no keyset is configured: want a refusal")
	}
}

type memStartupStorage struct {
	*memCheckStorage
	*memSafeStorage
}
