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
	"maps"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

// fakeKind is a store kept apart from any cryptography, to check that the
// Manager goes through the Store interface only.
const fakeKind = "test-fake"

// noProvisionKind is a store that does not create safes.
const noProvisionKind = "test-no-provision"

// reportingKind is a store whose keys rotate: its entries hold, as blob, the
// key that made them, and its current key is 2.
const reportingKind = "test-reporting"

type reportingStore struct{ nopStore }

func (reportingStore) Keys() []KeyInfo {
	return []KeyInfo{{Id: 1, Status: "ENABLED"}, {Id: 2, Primary: true, Status: "ENABLED"}}
}

func (reportingStore) WrappingKey(w happydns.WrappedKeyset) (uint32, error) {
	id, err := strconv.ParseUint(string(w.Blob), 10, 32)
	return uint32(id), err
}

func (s reportingStore) Rewrap(_ context.Context, _ *happydns.Safe, w *happydns.WrappedKeyset) (bool, error) {
	id, err := s.WrappingKey(*w)
	if err != nil || id == 2 {
		return false, err
	}
	w.Blob = []byte("2")
	return true, nil
}

// escrowKind is a store keeping, in its entries, copies of the key of safes
// of other kinds: in clear, as only a test can afford.
const escrowKind = "test-escrow"

type escrowStore struct{ nopStore }

func (escrowStore) Unwrap(_ context.Context, _ *happydns.Safe, w happydns.WrappedKeyset) (*keyset.Handle, error) {
	return insecurecleartextkeyset.Read(keyset.NewBinaryReader(bytes.NewReader(w.Blob)))
}

// escrowEntry returns an entry of the escrow store holding dek.
func escrowEntry(t *testing.T, dek *keyset.Handle) happydns.WrappedKeyset {
	t.Helper()
	var buf bytes.Buffer
	if err := insecurecleartextkeyset.Write(dek, keyset.NewBinaryWriter(&buf)); err != nil {
		t.Fatal(err)
	}
	return happydns.WrappedKeyset{KEK: escrowKind, Blob: buf.Bytes()}
}

// failingStartKind is a store whose startup check fails.
const failingStartKind = "test-failing-start"

type failingStartStore struct{ nopStore }

func (failingStartStore) CheckStartup(context.Context) error {
	return errors.New("the store cannot be reached")
}

// unreachableKind is a store of a service that does not answer: it never
// unlocks.
const unreachableKind = "test-unreachable"

type unreachableStore struct{ nopStore }

func (unreachableStore) Unlock(context.Context, *happydns.Safe) (UnlockedSafe, error) {
	return nil, errors.New("the service cannot be reached")
}

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

// withFakeStores returns cfg knowing the fake stores besides the builtin ones,
// unless it already chose its own.
func withFakeStores(cfg Config) Config {
	if cfg.factories == nil {
		cfg.factories = maps.Clone(builtinStores)
		cfg.factories[fakeKind] = func(Config) (Store, error) { return fakeStore{}, nil }
		cfg.factories[noProvisionKind] = nopFactory
		cfg.factories[unreachableKind] = func(Config) (Store, error) { return unreachableStore{}, nil }
		cfg.factories[failingStartKind] = func(Config) (Store, error) { return failingStartStore{}, nil }
		cfg.factories[reportingKind] = func(Config) (Store, error) { return reportingStore{}, nil }
		cfg.factories[escrowKind] = func(Config) (Store, error) { return escrowStore{}, nil }
		cfg.factories[creatorKind] = func(Config) (Store, error) { return creatorStore{kind: creatorKind}, nil }
		cfg.factories[passphraseKind] = func(Config) (Store, error) {
			return passphraseStore{kind: passphraseKind, field: "passphrase"}, nil
		}
		cfg.factories[passwordKind] = func(Config) (Store, error) {
			return passwordStore{passphraseStore{kind: passwordKind, field: PasswordParam}}, nil
		}
		cfg.factories[unattendedCreatorKind] = func(Config) (Store, error) {
			return unattendedCreatorStore{creatorStore{kind: unattendedCreatorKind}}, nil
		}
		cfg.factories[provisioningCreatorKind] = func(Config) (Store, error) {
			return provisioningCreatorStore{unattendedCreatorStore{creatorStore{kind: provisioningCreatorKind}}}, nil
		}
	}
	return cfg
}

func newTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	m, err := NewManager(withFakeStores(cfg))
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

// A safe kept by a store of an external service is accepted while the service
// does not answer: the safe only waits for it, as at startup, which does not
// call the services either.
func TestCheckSafeDoesNotCallExternalStores(t *testing.T) {
	sc := objectContext()
	m := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage()})

	safe := happydns.Safe{Id: happydns.Identifier{0x0d}, Owner: sc.Owner, Kind: unreachableKind}
	if err := m.CheckSafe(context.Background(), &safe); err != nil {
		t.Errorf("CheckSafe refused a safe of a store that does not answer: %v", err)
	}

	// The instance keyset is held here: a safe it does not open is refused.
	instance := happydns.Safe{Id: happydns.Identifier{0x0c}, Owner: sc.Owner, Kind: KindInstance}
	im := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage()})
	if err := im.CheckSafe(context.Background(), &instance); err == nil {
		t.Error("CheckSafe accepted an instance safe holding no key of this keyset")
	}
}

func TestNewManagerChecksThePolicyStore(t *testing.T) {
	for _, cfg := range []Config{
		{Policy: "test-unregistered", Safes: newMemSafeStorage()},
		{Policy: Policy(noProvisionKind), Safes: newMemSafeStorage()},
		{Policy: Policy(fakeKind)},
		{Policy: PolicyInstance, Safes: newMemSafeStorage()},
	} {
		if _, err := NewManager(withFakeStores(cfg)); err == nil {
			t.Errorf("NewManager(%+v) succeeded, want an error", cfg)
		}
	}
}

// The stores a test gives are known to its Manager only: another one knows
// the builtin stores alone.
func TestNewManagerKnowsTheBuiltinStoresOnly(t *testing.T) {
	newTestManager(t, Config{Policy: Policy(fakeKind), Safes: newMemSafeStorage()})

	if _, err := NewManager(Config{Policy: Policy(fakeKind), Safes: newMemSafeStorage()}); err == nil {
		t.Error("a Manager without the fake stores accepted their kind as policy")
	}
	if _, err := NewManager(Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage(), Stores: []string{string(PolicyPlaintext), fakeKind}}); err == nil {
		t.Error("a Manager without the fake stores accepted to enable one")
	}
}

// Only the stores the administrator enabled are built: the safes of the
// others do not open, and none is created.
func TestNewManagerBuildsTheEnabledStoresOnly(t *testing.T) {
	key := testInstanceKey(t)
	store := newMemSafeStorage()
	sc := objectContext()

	inInstance := &managedObject{ApiKey: happydns.NewSecret("a")}
	inFake := &managedObject{ApiKey: happydns.NewSecret("b")}
	if err := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: key, Safes: store}).SealObject(context.Background(), sc, inInstance); err != nil {
		t.Fatal(err)
	}
	if err := newTestManager(t, Config{Policy: Policy(fakeKind), Safes: store}).SealObject(context.Background(), sc, inFake); err != nil {
		t.Fatal(err)
	}

	m := newTestManager(t, Config{Policy: Policy(fakeKind), InstanceKey: key, Safes: store, Stores: []string{fakeKind}})
	if err := m.OpenObject(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, inFake.ApiKey.Token())}); err != nil {
		t.Errorf("a secret of an enabled store does not open: %v", err)
	}
	err := m.OpenObject(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, inInstance.ApiKey.Token())})
	if err == nil || !strings.Contains(err.Error(), KindInstance) {
		t.Errorf("OpenObject of a secret of a disabled store = %v, want an error naming it", err)
	}

	none := newTestManager(t, Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store, Stores: []string{string(PolicyPlaintext)}})
	if err := none.OpenObject(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, inFake.ApiKey.Token())}); err == nil {
		t.Error("a secret opened with no store enabled")
	}
}

func TestNewManagerPlaintextMustBeEnabled(t *testing.T) {
	safes := newMemSafeStorage()
	key := testInstanceKey(t)

	// Left out of the stores, plaintext is refused as the policy...
	if _, err := NewManager(Config{Policy: PolicyPlaintext, Safes: safes, Stores: []string{KindInstance}}); err == nil {
		t.Error("the plaintext policy was accepted with plaintext left out")
	}
	if _, err := NewManager(Config{Policy: PolicyPlaintext, Safes: safes, Stores: []string{}}); err == nil {
		t.Error("the plaintext policy was accepted with no store enabled")
	}

	// ... but not when enabled, nor when no choice was made.
	for _, stores := range [][]string{nil, {string(PolicyPlaintext)}, {string(PolicyPlaintext), KindInstance}} {
		if _, err := NewManager(Config{Policy: PolicyPlaintext, Safes: safes, Stores: stores}); err != nil {
			t.Errorf("NewManager with stores %v => %s", stores, err)
		}
	}

	// Plaintext does not have to be the policy, and being left out does not
	// stop the instance policy.
	if _, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: safes, Stores: []string{KindInstance}}); err != nil {
		t.Errorf("the instance policy without plaintext => %s", err)
	}
	if _, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: safes, Stores: []string{string(PolicyPlaintext)}}); err == nil {
		t.Error("the instance policy was accepted with the instance store left out")
	}
}

func TestNewManagerChecksTheEnabledStores(t *testing.T) {
	for _, cfg := range []Config{
		{Policy: PolicyPlaintext, Safes: newMemSafeStorage(), Stores: []string{"test-unregistered"}},
		{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage(), Stores: []string{fakeKind}},
	} {
		if _, err := NewManager(withFakeStores(cfg)); err == nil {
			t.Errorf("NewManager(%+v) succeeded, want an error", cfg)
		}
	}
}

// Safes of a kind without a store hold secrets that no longer open: better
// refuse to start than fail credential by credential.
func TestCheckSafeKinds(t *testing.T) {
	store := newMemSafeStorage()
	for i, kind := range []string{KindInstance, fakeKind, fakeKind} {
		safe := happydns.Safe{Id: happydns.Identifier{byte(i)}, Owner: happydns.Identifier{byte(i)}, Kind: kind}
		if err := store.CreateSafe(&safe); err != nil {
			t.Fatal(err)
		}
	}
	key := testInstanceKey(t)

	all := newTestManager(t, Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store})
	if err := all.checkSafeKinds(); err != nil {
		t.Errorf("CheckSafeKinds with every store: %v", err)
	}

	m := newTestManager(t, Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store, Stores: []string{string(PolicyPlaintext), KindInstance}})
	err := m.checkSafeKinds()
	if err == nil || !strings.Contains(err.Error(), fakeKind) || !strings.Contains(err.Error(), "2") {
		t.Errorf("CheckSafeKinds = %v, want an error naming the kind and its 2 safes", err)
	}

	empty := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage(), Stores: []string{string(PolicyPlaintext)}})
	if err := empty.checkSafeKinds(); err != nil {
		t.Errorf("CheckSafeKinds without any safe: %v", err)
	}

	var nilManager *Manager
	if err := nilManager.checkSafeKinds(); err == nil {
		t.Error("CheckSafeKinds on a nil Manager succeeded")
	}
}

// A safe record that does not decode tells no kind: it is left to the
// reports, and does not keep happyDomain from starting.
func TestCheckSafeKindsSkipsDamagedRecords(t *testing.T) {
	store := newMemSafeStorage()
	store.damaged["safe-broken"] = errors.New("does not decode")

	m := newTestManager(t, Config{Policy: PolicyPlaintext, InstanceKey: testInstanceKey(t), Safes: store})
	if err := m.checkSafeKinds(); err != nil {
		t.Errorf("CheckSafeKinds beside a damaged record: %v", err)
	}
}

// A store checks itself at startup: when it fails, happyDomain does not
// start, unless the store is left out.
func TestStartRunsTheStoreChecks(t *testing.T) {
	_, err := Start(context.Background(), withFakeStores(Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage()}))
	if err == nil || !strings.Contains(err.Error(), failingStartKind) {
		t.Errorf("Start with a failing store = %v, want an error naming it", err)
	}

	cfg := withFakeStores(Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage(), Stores: []string{string(PolicyPlaintext), fakeKind}})
	if _, err := Start(context.Background(), cfg); err != nil {
		t.Errorf("Start with the failing store left out: %v", err)
	}
}
