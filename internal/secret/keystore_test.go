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
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// creatorKind is a store whose safes its owner creates, giving a passphrase,
// and that does not open without them.
const creatorKind = "test-creator"

// unattendedCreatorKind is a store whose safes its owner creates, and that
// opens without them.
const unattendedCreatorKind = "test-unattended-creator"

type creatorStore struct{ kind string }

func (s creatorStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != s.kind {
		return nil, errors.New("not a safe of this store")
	}
	return fakeUnlocked{id: safe.Id}, nil
}

func (creatorStore) CreateForm() []happydns.Field {
	return []happydns.Field{{Id: "passphrase", Type: "string", Secret: true, Required: true}}
}

func (s creatorStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p struct {
		Passphrase string `json:"passphrase"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if p.Passphrase == "" {
		return errors.New("a passphrase is required")
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.kind}}
	return nil
}

// updaterKind is a store whose safes their owner updates, replacing the blob
// of their keyring entry.
const updaterKind = "test-updater"

type updaterStore struct{ creatorStore }

func (updaterStore) UpdateForm() []happydns.Field {
	return []happydns.Field{{Id: "blob", Type: "string", Required: true}}
}

func (s updaterStore) Update(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p struct {
		Blob string `json:"blob"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	keyring := slices.Clone(safe.Keyring)
	keyring[0].Blob = []byte(p.Blob)
	safe.Keyring = keyring
	return nil
}

type unattendedCreatorStore struct{ creatorStore }

func (unattendedCreatorStore) OpensUnattended() bool { return true }

// provisioningCreatorKind is a store that creates a safe for each owner, and
// whose owners create personal ones too.
const provisioningCreatorKind = "test-provisioning-creator"

type provisioningCreatorStore struct{ unattendedCreatorStore }

func (s provisioningCreatorStore) Provision(_ context.Context, safe *happydns.Safe) error {
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.kind}}
	return nil
}

// memDefaults keeps the default keystores in memory.
type memDefaults struct {
	mu   sync.Mutex
	refs map[string]string
}

func newMemDefaults() *memDefaults { return &memDefaults{refs: map[string]string{}} }

func (d *memDefaults) GetDefaultKeystore(owner happydns.Identifier) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	ref, ok := d.refs[owner.String()]
	if !ok {
		return "", happydns.ErrNotFound
	}
	return ref, nil
}

func (d *memDefaults) PutDefaultKeystore(owner happydns.Identifier, ref string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.refs[owner.String()] = ref
	return nil
}

func (d *memDefaults) ListDefaultKeystores() (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[string]string, len(d.refs))
	for owner, ref := range d.refs {
		out[owner] = ref
	}
	return out, nil
}

func (d *memDefaults) DeleteDefaultKeystore(owner happydns.Identifier) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.refs, owner.String())
	return nil
}

type keystoreFixture struct {
	m        *Manager
	safes    *memSafeStorage
	defaults *memDefaults
}

func newKeystoreFixture(t *testing.T, policy Policy) keystoreFixture {
	t.Helper()
	f := keystoreFixture{safes: newMemSafeStorage(), defaults: newMemDefaults()}
	f.m = newTestManager(t, Config{
		Policy:      policy,
		InstanceKey: testInstanceKey(t),
		Safes:       f.safes,
		Defaults:    f.defaults,
	})
	return f
}

func passphrase(p string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"passphrase": p})
	return b
}

func (f keystoreFixture) create(t *testing.T, owner happydns.Identifier, kind, name string) Keystore {
	t.Helper()
	ks, err := f.m.CreateKeystore(context.Background(), owner, kind, name, passphrase("p"))
	if err != nil {
		t.Fatalf("CreateKeystore(%s, %s): %v", kind, name, err)
	}
	return ks
}

// safeOf returns the safe s is sealed in.
func safeOf(t *testing.T, s happydns.Secret) string {
	t.Helper()
	sv, err := ParseSealed(s.Token())
	if err != nil {
		t.Fatalf("%q is not sealed: %v", s.Token(), err)
	}
	return sv.SafeId.String()
}

func findKeystore(list []Keystore, ref string) *Keystore {
	for i := range list {
		if list[i].Ref == ref {
			return &list[i]
		}
	}
	return nil
}

func TestKeystoresListsInstanceAndPersonal(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	owner := objectContext().Owner
	ctx := context.Background()

	list, err := f.m.Keystores(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	inst := findKeystore(list, KindInstance)
	if inst == nil {
		t.Fatalf("Keystores = %+v, want the instance keystore, even before it is created", list)
	}
	if inst.Scope != ScopeInstance || !inst.Unattended || !inst.Default || !inst.SafeId.IsEmpty() {
		t.Errorf("instance keystore = %+v", inst)
	}
	for _, ks := range list {
		if ks.Kind == creatorKind {
			t.Errorf("Keystores lists a kind without a safe: %+v", ks)
		}
	}

	mine := f.create(t, owner, creatorKind, "Mine")
	if mine.Scope != ScopePersonal || mine.Name != "Mine" || mine.Unattended || mine.Ref != mine.SafeId.String() {
		t.Errorf("created keystore = %+v", mine)
	}

	sealOne(t, f.m, objectContext(), "v")
	list, _ = f.m.Keystores(ctx, owner)
	personal := 0
	for _, ks := range list {
		if ks.Scope == ScopePersonal {
			personal++
		}
	}
	if personal != 1 {
		t.Fatalf("Keystores = %+v, want 1 personal", list)
	}
	if inst := findKeystore(list, KindInstance); inst == nil || inst.SafeId.IsEmpty() {
		t.Errorf("instance keystore once used = %+v, want its safe", inst)
	}
	if ks := findKeystore(list, mine.Ref); ks == nil || ks.Default {
		t.Errorf("personal keystore = %+v, want it listed, not default", ks)
	}

	// Another owner sees none of them.
	other, _ := f.m.Keystores(ctx, happydns.Identifier{0x02})
	if findKeystore(other, mine.Ref) != nil {
		t.Error("the keystore of an owner is listed for another")
	}
}

func TestCreatableKinds(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	kinds := f.m.CreatableKinds()

	byKind := map[string]KindInfo{}
	for _, k := range kinds {
		byKind[k.Kind] = k
	}
	if _, ok := byKind[KindInstance]; ok {
		t.Error("the instance kind is offered for creation")
	}
	if k, ok := byKind[creatorKind]; !ok || len(k.Form) != 1 || k.Unattended {
		t.Errorf("creator kind = %+v, %v", k, ok)
	}
	if k, ok := byKind[unattendedCreatorKind]; !ok || !k.Unattended {
		t.Errorf("unattended creator kind = %+v, %v", k, ok)
	}
}

func TestCreateKeystoreRefusals(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	owner := objectContext().Owner
	ctx := context.Background()
	f.create(t, owner, creatorKind, "Mine")

	for name, c := range map[string]struct {
		kind, name string
		params     json.RawMessage
	}{
		"unknown kind":      {"nope", "A", passphrase("p")},
		"instance kind":     {KindInstance, "A", passphrase("p")},
		"no create":         {fakeKind, "A", passphrase("p")},
		"no name":           {creatorKind, "  ", passphrase("p")},
		"name taken":        {creatorKind, "mine", passphrase("p")},
		"refused by store":  {creatorKind, "B", passphrase("")},
		"params not object": {creatorKind, "B", json.RawMessage(`"x"`)},
	} {
		if _, err := f.m.CreateKeystore(ctx, owner, c.kind, c.name, c.params); err == nil {
			t.Errorf("CreateKeystore(%s) succeeded", name)
		}
	}

	// The same name for another owner is fine.
	f.create(t, happydns.Identifier{0x02}, creatorKind, "Mine")
}

func TestSealIntoChosenKeystore(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	obj := &managedObject{ApiKey: happydns.NewSecret("a"), Other: happydns.NewSecret("b")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: mine.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	if safeOf(t, obj.ApiKey) != mine.Ref || safeOf(t, obj.Other) != mine.Ref {
		t.Errorf("sealed into %s and %s, want %s", safeOf(t, obj.ApiKey), safeOf(t, obj.Other), mine.Ref)
	}

	// Reread from storage, it opens.
	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "a" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}

	// Without a choice, a new secret goes where the object already is.
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token()), Other: happydns.NewSecret("c")}
	if err := f.m.SealObject(ctx, sc, stored); err != nil {
		t.Fatal(err)
	}
	if safeOf(t, stored.Other) != mine.Ref {
		t.Errorf("a new secret of an object in a personal keystore went to %s", safeOf(t, stored.Other))
	}
}

func TestSealIntoMovesWhatIsSealedElsewhere(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()

	obj := &managedObject{ApiKey: happydns.NewSecret("a")}
	if err := f.m.SealObject(ctx, sc, obj); err != nil {
		t.Fatal(err)
	}
	instanceSafe := safeOf(t, obj.ApiKey)

	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	// As read from storage: sealed, not opened.
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token()), Other: happydns.NewSecret("b")}
	if err := f.m.SealObjectInto(ctx, sc, stored, Destination{Keystore: mine.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	if safeOf(t, stored.ApiKey) != mine.Ref || safeOf(t, stored.Other) != mine.Ref {
		t.Fatalf("not moved: %s, %s", safeOf(t, stored.ApiKey), safeOf(t, stored.Other))
	}
	if stored.ApiKey.Reveal() != "a" {
		t.Errorf("moved value = %q", stored.ApiKey.Reveal())
	}

	// And back to the instance keystore.
	again := &managedObject{ApiKey: sealedFromStorage(t, stored.ApiKey.Token())}
	if err := f.m.SealObjectInto(ctx, sc, again, Destination{Keystore: KindInstance}); err != nil {
		t.Fatal(err)
	}
	if safeOf(t, again.ApiKey) != instanceSafe {
		t.Errorf("moved back into %s, want the instance safe %s", safeOf(t, again.ApiKey), instanceSafe)
	}
}

func TestDefaultKeystore(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}
	if ref, _ := f.m.DefaultKeystore(sc.Owner); ref != mine.Ref {
		t.Errorf("DefaultKeystore = %q, want %q", ref, mine.Ref)
	}
	list, _ := f.m.Keystores(ctx, sc.Owner)
	if ks := findKeystore(list, mine.Ref); ks == nil || !ks.Default {
		t.Errorf("the default keystore is not flagged: %+v", list)
	}
	if ks := findKeystore(list, KindInstance); ks == nil || ks.Default {
		t.Errorf("the instance keystore is still flagged default: %+v", ks)
	}

	if tok := sealOne(t, f.m, sc, "v"); safeOf(t, sealedFromStorage(t, tok)) != mine.Ref {
		t.Errorf("a new secret did not go to the default keystore")
	}

	// An object used in the background does not go to a default that needs
	// its owner: it falls back to the policy.
	bg := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(ctx, sc, bg, Destination{Unattended: true}); err != nil {
		t.Fatalf("SealObjectInto(unattended): %v", err)
	}
	if safeOf(t, bg.ApiKey) == mine.Ref {
		t.Error("an unattended object went to a keystore that needs its owner")
	}

	// Another owner cannot pick it.
	if err := f.m.SetDefaultKeystore(ctx, happydns.Identifier{0x02}, mine.Ref); !errors.Is(err, ErrUnknownKeystore) {
		t.Errorf("SetDefaultKeystore(other owner) = %v, want ErrUnknownKeystore", err)
	}

	// Cleared, the policy applies again.
	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, ""); err != nil {
		t.Fatal(err)
	}
	if tok := sealOne(t, f.m, sc, "v"); safeOf(t, sealedFromStorage(t, tok)) == mine.Ref {
		t.Error("a cleared default still applies")
	}
}

func TestDestinationRefusals(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")
	theirs := f.create(t, happydns.Identifier{0x02}, creatorKind, "Theirs")
	unknown := happydns.Identifier{0x42}
	unknownSafeRef := unknown.String()

	for name, c := range map[string]struct {
		dst  Destination
		want error
	}{
		"unknown ref":              {Destination{Keystore: "nope"}, ErrUnknownKeystore},
		"another owner's":          {Destination{Keystore: theirs.Ref}, ErrUnknownKeystore},
		"unknown safe":             {Destination{Keystore: unknownSafeRef}, ErrUnknownKeystore},
		"needs owner, unattended":  {Destination{Keystore: mine.Ref, Unattended: true}, ErrKeystoreNotUnattended},
		"plaintext is no keystore": {Destination{Keystore: string(PolicyPlaintext)}, ErrUnknownKeystore},
	} {
		obj := &managedObject{ApiKey: happydns.NewSecret("v")}
		if err := f.m.SealObjectInto(ctx, sc, obj, c.dst); !errors.Is(err, c.want) {
			t.Errorf("SealObjectInto(%s) = %v, want %v", name, err, c.want)
		}
		if !obj.ApiKey.IsClear() {
			t.Errorf("SealObjectInto(%s) changed the object", name)
		}
	}
}

func TestPlaintextPolicyWithKeystores(t *testing.T) {
	f := newKeystoreFixture(t, PolicyPlaintext)
	sc := objectContext()
	ctx := context.Background()

	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObject(ctx, sc, obj); err != nil {
		t.Fatal(err)
	}
	if obj.ApiKey.IsSealed() || IsSealed(obj.ApiKey.Token()) {
		t.Error("the plaintext policy sealed a secret without any choice")
	}

	// The owner may still choose the instance keystore, or one of theirs.
	obj = &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: KindInstance}); err != nil {
		t.Fatal(err)
	}
	if !IsSealed(obj.ApiKey.Token()) {
		t.Error("the instance keystore chosen under the plaintext policy did not seal")
	}

	list, _ := f.m.Keystores(ctx, sc.Owner)
	if ks := findKeystore(list, KindInstance); ks == nil || ks.Default {
		t.Errorf("under the plaintext policy, the instance keystore = %+v, want listed, not default", ks)
	}
}

func TestDeleteKeystore(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")
	sealOne(t, f.m, sc, "v")
	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}

	if err := f.m.DeleteKeystore(ctx, sc.Owner, KindInstance, false); !errors.Is(err, ErrInstanceKeystore) {
		t.Errorf("DeleteKeystore(instance) = %v, want ErrInstanceKeystore", err)
	}
	if err := f.m.DeleteKeystore(ctx, happydns.Identifier{0x02}, mine.Ref, false); !errors.Is(err, ErrUnknownKeystore) {
		t.Errorf("DeleteKeystore(other owner) = %v, want ErrUnknownKeystore", err)
	}

	if err := f.m.DeleteKeystore(ctx, sc.Owner, mine.Ref, false); err != nil {
		t.Fatal(err)
	}
	if _, err := f.safes.GetSafe(mine.SafeId); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the safe is still stored: %v", err)
	}
	if ref, _ := f.m.DefaultKeystore(sc.Owner); ref != "" {
		t.Errorf("the default still names the deleted keystore: %q", ref)
	}

	// The administrator may delete the instance keystore of an owner.
	if err := f.m.DeleteKeystore(ctx, sc.Owner, KindInstance, true); err != nil {
		t.Errorf("DeleteKeystore(instance, admin) = %v", err)
	}
	if _, err := f.safes.GetSafeByOwner(sc.Owner, KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the instance safe is still stored: %v", err)
	}
}

func TestUpdateKeystoreRefusals(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")
	// Sealing creates the instance keystore, so the refusal below is for an
	// existing one, not for one still to be provisioned.
	sealOne(t, f.m, sc, "v")

	// Its store has nothing to update: updating it would take a new key.
	if _, err := f.m.UpdateKeystore(ctx, sc.Owner, mine.Ref, passphrase("q")); !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("UpdateKeystore(no Updater) = %v, want ErrInvalidKeystore", err)
	}
	if mine.UpdateForm != nil {
		t.Errorf("UpdateForm = %+v, want none", mine.UpdateForm)
	}
	if _, err := f.m.UpdateKeystore(ctx, sc.Owner, KindInstance, json.RawMessage(`{}`)); !errors.Is(err, ErrInstanceKeystore) {
		t.Errorf("UpdateKeystore(instance) = %v, want ErrInstanceKeystore", err)
	}
	if _, err := f.m.UpdateKeystore(ctx, happydns.Identifier{0x02}, mine.Ref, passphrase("q")); !errors.Is(err, ErrUnknownKeystore) {
		t.Errorf("UpdateKeystore(other owner) = %v, want ErrUnknownKeystore", err)
	}
}

// sharedSafeStorage hands out the very safes it holds, as a cache would, and
// cannot write them.
type sharedSafeStorage struct {
	*memSafeStorage
	shared map[string]*happydns.Safe
}

func (s *sharedSafeStorage) ListOwnerSafes(owner happydns.Identifier) ([]*happydns.Safe, error) {
	safes, err := s.memSafeStorage.ListOwnerSafes(owner)
	for i, safe := range safes {
		if kept, ok := s.shared[safe.Id.String()]; ok {
			safes[i] = kept
		} else {
			s.shared[safe.Id.String()] = safe
		}
	}
	return safes, err
}

func (s *sharedSafeStorage) UpdateSafe(*happydns.Safe) error { return errors.New("storage is down") }

func TestUpdateKeystoreLeavesTheSafeOnStorageError(t *testing.T) {
	safes := &sharedSafeStorage{memSafeStorage: newMemSafeStorage(), shared: map[string]*happydns.Safe{}}
	m := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: safes, Defaults: newMemDefaults()})
	sc := objectContext()
	ctx := context.Background()
	ks, err := m.CreateKeystore(ctx, sc.Owner, updaterKind, "Mine", passphrase("p"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.UpdateKeystore(ctx, sc.Owner, ks.Ref, json.RawMessage(`{"blob":"new"}`)); err == nil {
		t.Fatal("UpdateKeystore succeeded though the safe was not stored")
	}
	list, err := safes.ListOwnerSafes(sc.Owner)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListOwnerSafes = %v, %v", list, err)
	}
	if string(list[0].Keyring[0].Blob) == "new" {
		t.Error("the safe handed out by the storage carries an update that was not stored")
	}
}

func TestKeystoreSafe(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	if _, err := f.m.KeystoreSafe(ctx, sc.Owner, KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("KeystoreSafe(instance, not created yet) = %v, want ErrSafeNotFound", err)
	}
	tok := sealOne(t, f.m, sc, "v")
	if id, err := f.m.KeystoreSafe(ctx, sc.Owner, KindInstance); err != nil || id.String() != safeOf(t, sealedFromStorage(t, tok)) {
		t.Errorf("KeystoreSafe(instance) = %v, %v", id, err)
	}
	if id, err := f.m.KeystoreSafe(ctx, sc.Owner, mine.Ref); err != nil || !id.Equals(mine.SafeId) {
		t.Errorf("KeystoreSafe(mine) = %v, %v", id, err)
	}
	if _, err := f.m.KeystoreSafe(ctx, happydns.Identifier{0x02}, mine.Ref); !errors.Is(err, ErrUnknownKeystore) {
		t.Errorf("KeystoreSafe(other owner) = %v, want ErrUnknownKeystore", err)
	}
}

func TestSafesOf(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	obj := &managedObject{ApiKey: happydns.NewSecret("a"), Other: happydns.NewSecret("b")}
	if ids, err := SafesOf(obj); err != nil || len(ids) != 0 {
		t.Errorf("SafesOf(clear) = %v, %v", ids, err)
	}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: mine.Ref}); err != nil {
		t.Fatal(err)
	}
	ids, err := SafesOf(obj)
	if err != nil || len(ids) != 1 || !ids[0].Equals(mine.SafeId) {
		t.Errorf("SafesOf = %v, %v; want only %s", ids, err, mine.Ref)
	}
}

func TestResealLeavesPersonalKeystoresAlone(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")

	obj := &managedObject{ApiKey: happydns.NewSecret("a")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: mine.Ref}); err != nil {
		t.Fatal(err)
	}
	token := obj.ApiKey.Token()

	plaintext := newTestManager(t, Config{Policy: PolicyPlaintext, InstanceKey: testInstanceKey(t), Safes: f.safes, Defaults: f.defaults})
	stored := &managedObject{ApiKey: sealedFromStorage(t, token)}
	changed, err := plaintext.ResealObject(ctx, sc, stored)
	if err != nil || changed {
		t.Errorf("ResealObject(plaintext) of a personal keystore = %v, %v; want left alone", changed, err)
	}
	if stored.ApiKey.Token() != token {
		t.Error("the object changed")
	}

	// Clear values go to the policy, not to the default of the owner.
	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}
	legacy := &managedObject{ApiKey: happydns.NewSecret("a")}
	changed, err = f.m.ResealObject(ctx, sc, legacy)
	if err != nil || !changed {
		t.Fatalf("ResealObject(clear) = %v, %v", changed, err)
	}
	if safeOf(t, legacy.ApiKey) == mine.Ref {
		t.Error("a reseal followed the default of the owner rather than the policy")
	}
}

func TestSealUnderPolicyIgnoresTheDefault(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	mine := f.create(t, sc.Owner, creatorKind, "Mine")
	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("a")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{}.UnderPolicy()); err != nil {
		t.Fatalf("SealObjectInto(under policy): %v", err)
	}
	if safeOf(t, obj.ApiKey) == mine.Ref {
		t.Error("sealed into the default of the owner rather than under the policy")
	}
}

func TestDeleteOwnerSafesForgetsDefault(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	owner := objectContext().Owner
	mine := f.create(t, owner, creatorKind, "Mine")
	if err := f.m.SetDefaultKeystore(context.Background(), owner, mine.Ref); err != nil {
		t.Fatal(err)
	}
	if err := f.m.DeleteOwnerSafes(owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.defaults.GetDefaultKeystore(owner); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("the default survived the owner: %v", err)
	}
}

func TestNoChoiceMovesNothing(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()

	inInstance := &managedObject{ApiKey: happydns.NewSecret("a")}
	if err := f.m.SealObject(ctx, sc, inInstance); err != nil {
		t.Fatal(err)
	}
	instanceSafe := safeOf(t, inInstance.ApiKey)

	mine := f.create(t, sc.Owner, creatorKind, "Mine")
	if err := f.m.SetDefaultKeystore(ctx, sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}

	// An object the instance keeps stays whole there: the default is for
	// new objects.
	stored := &managedObject{ApiKey: sealedFromStorage(t, inInstance.ApiKey.Token()), Other: happydns.NewSecret("b")}
	if err := f.m.SealObject(ctx, sc, stored); err != nil {
		t.Fatal(err)
	}
	if safeOf(t, stored.ApiKey) != instanceSafe || safeOf(t, stored.Other) != instanceSafe {
		t.Errorf("an object of the instance keystore went to %s and %s", safeOf(t, stored.ApiKey), safeOf(t, stored.Other))
	}

	// A value sealed in a safe this instance does not know, as restored from
	// elsewhere, is kept as it is.
	foreign := "hds:1:AQ:c2VhbGVk"
	obj := &managedObject{ApiKey: sealedFromStorage(t, foreign)}
	if err := f.m.SealObject(ctx, sc, obj); err != nil {
		t.Fatalf("SealObject(foreign sealed value) = %v", err)
	}
	if obj.ApiKey.Token() != foreign {
		t.Errorf("a foreign sealed value changed: %q", obj.ApiKey.Token())
	}
	if out, err := f.m.SealValue(ctx, SecretContext{Owner: sc.Owner, ObjectType: "x", ObjectId: "y", Field: "z"}, foreign); err != nil || out != foreign {
		t.Errorf("SealValue(foreign sealed value) = %q, %v", out, err)
	}
}

func TestInstanceKeystoreIsNoPersonalOneOfItsKind(t *testing.T) {
	f := newKeystoreFixture(t, Policy(provisioningCreatorKind))
	sc := objectContext()
	ctx := context.Background()

	// Created first, the personal keystore is the oldest safe of its kind.
	mine := f.create(t, sc.Owner, provisioningCreatorKind, "Mine")

	obj := &managedObject{ApiKey: happydns.NewSecret("a")}
	if err := f.m.SealObject(ctx, sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	instance := safeOf(t, obj.ApiKey)
	if instance == mine.Ref {
		t.Fatalf("sealed under the policy into the personal keystore %s", mine.Ref)
	}

	// Moved into the keystore of the instance, from the personal one.
	moved := &managedObject{ApiKey: happydns.NewSecret("b")}
	if err := f.m.SealObjectInto(ctx, sc, moved, Destination{Keystore: mine.Ref}); err != nil {
		t.Fatal(err)
	}
	if err := f.m.SealObjectInto(ctx, sc, moved, Destination{Keystore: provisioningCreatorKind}); err != nil {
		t.Fatalf("SealObjectInto(instance): %v", err)
	}
	if got := safeOf(t, moved.ApiKey); got != instance {
		t.Errorf("moved into %s, want the keystore of the instance %s", got, instance)
	}
}
