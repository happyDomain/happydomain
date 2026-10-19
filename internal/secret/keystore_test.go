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
