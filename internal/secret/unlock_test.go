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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

// passphraseKind is a personal keystore opening with a passphrase its owner
// gives: its keyring keeps the digest of the passphrase, as only a test can
// afford.
const passphraseKind = "test-passphrase"

// passwordKind is a personal keystore opening with the login password of
// its owner.
const passwordKind = "test-password"

func digest(s string) []byte {
	d := sha256.Sum256([]byte(s))
	return d[:]
}

type passphraseStore struct{ kind, field string }

func (s passphraseStore) Unlock(context.Context, *happydns.Safe) (UnlockedSafe, error) {
	return nil, errors.New("an interactive safe is not opened without its owner")
}

func (s passphraseStore) UnlockForm() []happydns.Field {
	return []happydns.Field{{Id: s.field, Type: "string", Secret: true, Required: true}}
}

func (s passphraseStore) CreateForm() []happydns.Field { return s.UnlockForm() }

func (s passphraseStore) read(raw json.RawMessage) (string, error) {
	var p map[string]string
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if p[s.field] == "" {
		return "", errors.New(s.field + " is required")
	}
	return p[s.field], nil
}

func (s passphraseStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	v, err := s.read(params)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.kind, Blob: digest(v)}}
	return nil
}

func (s passphraseStore) open(safe *happydns.Safe, v string) (UnlockedSafe, error) {
	for _, w := range safe.Keyring {
		if w.KEK == s.kind && string(w.Blob) == string(digest(v)) {
			return fakeUnlocked{id: safe.Id}, nil
		}
	}
	return nil, ErrWrongUnlockMaterial
}

func (s passphraseStore) UnlockWith(_ context.Context, safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error) {
	v, err := s.read(material)
	if err != nil {
		return nil, ErrWrongUnlockMaterial
	}
	return s.open(safe, v)
}

// passwordStore is no Creator: the Manager creates its keystores from the
// login password alone, through Bind.
type passwordStore struct{ inner passphraseStore }

func (s passwordStore) Unlock(ctx context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	return s.inner.Unlock(ctx, safe)
}

func (s passwordStore) UnlockForm() []happydns.Field { return s.inner.UnlockForm() }

func (s passwordStore) UnlockWith(ctx context.Context, safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error) {
	return s.inner.UnlockWith(ctx, safe, material)
}

func (s passwordStore) Bind(_ context.Context, safe *happydns.Safe, password string) error {
	safe.Keyring = []happydns.WrappedKeyset{{KEK: s.inner.kind, Blob: digest(password)}}
	return nil
}

func (s passwordStore) UnlockWithPassword(_ context.Context, safe *happydns.Safe, password string) (UnlockedSafe, error) {
	return s.inner.open(safe, password)
}

func (s passwordStore) Rebind(_ context.Context, safe *happydns.Safe, old, new string) error {
	if _, err := s.inner.open(safe, old); err != nil {
		return err
	}
	for i := range safe.Keyring {
		if safe.Keyring[i].KEK == s.inner.kind {
			safe.Keyring[i].Blob = digest(new)
		}
	}
	return nil
}

func interactiveFixture(t *testing.T) keystoreFixture {
	t.Helper()
	f := keystoreFixture{safes: newMemSafeStorage(), defaults: newMemDefaults()}
	f.m = newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: f.safes, Defaults: f.defaults})
	return f
}

func scoped(scope string) context.Context {
	return WithUnlockScope(context.Background(), scope)
}

func phrase(p string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"passphrase": p})
	return b
}

func isLocked(err error, ref string) bool {
	var le happydns.KeystoreLockedError
	return errors.As(err, &le) && le.Ref == ref && len(le.Form) == 1
}

func TestInteractiveKeystoreOpensOnceUnlocked(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, passphraseKind, "Mine", phrase("open sesame"))
	if err != nil {
		t.Fatal(err)
	}
	s1 := scoped("session-1")

	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
		t.Fatalf("SealObjectInto(locked) = %v, want a KeystoreLockedError naming %s", err, ks.Ref)
	}

	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("wrong")); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("UnlockKeystore(wrong) = %v, want ErrWrongUnlockMaterial", err)
	}
	if err := f.m.UnlockKeystore(context.Background(), sc.Owner, ks.Ref, phrase("open sesame")); err == nil {
		t.Error("UnlockKeystore without a scope succeeded")
	}
	if err := f.m.UnlockKeystore(s1, happydns.Identifier{0x02}, ks.Ref, phrase("open sesame")); !errors.Is(err, ErrUnknownKeystore) {
		t.Errorf("UnlockKeystore(other owner) = %v, want ErrUnknownKeystore", err)
	}
	if err := f.m.UnlockKeystore(s1, sc.Owner, KindInstance, phrase("x")); !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("UnlockKeystore(instance) = %v, want ErrInvalidKeystore", err)
	}
	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("open sesame")); err != nil {
		t.Fatalf("UnlockKeystore = %v", err)
	}

	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto(unlocked) = %v", err)
	}
	token := obj.ApiKey.Token()

	back := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := f.m.OpenObject(s1, sc, back); err != nil || back.ApiKey.Reveal() != "v" {
		t.Errorf("OpenObject(unlocked scope) = %q, %v", back.ApiKey.Reveal(), err)
	}
	for name, ctx := range map[string]context.Context{"another session": scoped("session-2"), "background": context.Background()} {
		back := &managedObject{ApiKey: sealedFromStorage(t, token)}
		if err := f.m.OpenObject(ctx, sc, back); !isLocked(err, ks.Ref) {
			t.Errorf("OpenObject(%s) = %v, want locked", name, err)
		}
	}

	list, _ := f.m.Keystores(s1, sc.Owner)
	if got := findKeystore(list, ks.Ref); got == nil || !got.Interactive || got.Locked || len(got.UnlockForm) != 1 {
		t.Errorf("Keystores(unlocked scope) = %+v", got)
	}
	list, _ = f.m.Keystores(scoped("session-2"), sc.Owner)
	if got := findKeystore(list, ks.Ref); got == nil || !got.Locked {
		t.Errorf("Keystores(other scope) = %+v, want locked", got)
	}
	if got := findKeystore(list, KindInstance); got == nil || got.Interactive || got.Locked {
		t.Errorf("instance keystore = %+v, never locked", got)
	}
}

// Its owner just gave what opens it: a keystore created during a session is
// unlocked for that session, and described as such.
func TestInteractiveKeystoreUnlockedForTheSessionCreatingIt(t *testing.T) {
	for kind, params := range map[string]json.RawMessage{
		passphraseKind: phrase("open sesame"),
		passwordKind:   json.RawMessage(`{"password":"hunter2"}`),
	} {
		t.Run(kind, func(t *testing.T) {
			f := interactiveFixture(t)
			sc := objectContext()
			s1 := scoped("session-1")

			ks, err := f.m.CreateKeystore(s1, sc.Owner, kind, "Mine", params)
			if err != nil {
				t.Fatal(err)
			}
			if !ks.Interactive || ks.Locked || len(ks.UnlockForm) != 1 {
				t.Errorf("created keystore = %+v, want it interactive, unlocked, with its unlock form", ks)
			}
			if err := f.m.SealObjectInto(s1, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); err != nil {
				t.Errorf("SealObjectInto(session creating it) = %v", err)
			}
			if err := f.m.SealObjectInto(scoped("session-2"), sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
				t.Errorf("SealObjectInto(another session) = %v, want locked", err)
			}

			other, err := f.m.CreateKeystore(context.Background(), sc.Owner, kind, "Without a session", params)
			if err != nil {
				t.Fatal(err)
			}
			if !other.Interactive || !other.Locked {
				t.Errorf("keystore created without a session = %+v, want it interactive and locked", other)
			}
		})
	}
}

// A copy of its key under another store does not open an interactive safe:
// the passphrase would be of no use.
func TestInteractiveKeystoreIgnoresCopies(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, passphraseKind, "Mine", phrase("p"))
	if err != nil {
		t.Fatal(err)
	}
	safe, _ := f.safes.GetSafe(ks.SafeId)
	dek, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		t.Fatal(err)
	}
	safe.Keyring = append(safe.Keyring, escrowEntry(t, dek))
	if err := f.safes.UpdateSafe(safe); err != nil {
		t.Fatal(err)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(scoped("s"), sc, obj, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
		t.Errorf("SealObjectInto = %v, want locked despite the copy", err)
	}
}

func TestLockingInteractiveKeystores(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, passphraseKind, "Mine", phrase("p"))
	if err != nil {
		t.Fatal(err)
	}
	s1 := scoped("s1")
	unlock := func() {
		t.Helper()
		if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("p")); err != nil {
			t.Fatal(err)
		}
	}
	locked := func(step string) {
		t.Helper()
		obj := &managedObject{ApiKey: happydns.NewSecret("v")}
		if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
			t.Errorf("after %s: SealObjectInto = %v, want locked", step, err)
		}
	}

	unlock()
	if err := f.m.LockKeystore(s1, sc.Owner, ks.Ref); err != nil {
		t.Fatal(err)
	}
	locked("LockKeystore")

	unlock()
	f.m.LockScope("s1")
	locked("LockScope")

	unlock()
	f.m.LockOwner(sc.Owner)
	locked("LockOwner")

	unlock()
	f.m.PurgeScopes(func(scope string) bool { return scope != "s1" })
	locked("PurgeScopes")

	unlock()
	f.m.PurgeScopes(func(string) bool { return true })
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Errorf("a live scope was purged: %v", err)
	}
}

func TestPasswordBoundKeystore(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	pw := func(p string) json.RawMessage {
		b, _ := json.Marshal(map[string]string{"password": p})
		return b
	}
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, passwordKind, "Login", pw("old"))
	if err != nil {
		t.Fatal(err)
	}
	if kinds := f.m.CreatableKinds(); !slicesContainsKind(kinds, passwordKind, true) {
		t.Errorf("CreatableKinds = %+v, want %s flagged password bound, asking the password", kinds, passwordKind)
	}
	if _, err := f.m.CreateKeystore(context.Background(), sc.Owner, passwordKind, "No password", json.RawMessage(`{}`)); !errors.Is(err, ErrInvalidKeystore) {
		t.Errorf("CreateKeystore(password bound, no password) = %v, want ErrInvalidKeystore", err)
	}

	s1 := scoped("s1")
	f.m.UnlockAtLogin(s1, sc.Owner, "old")
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto after login = %v", err)
	}
	token := obj.ApiKey.Token()

	if err := f.m.PasswordChanged(s1, sc.Owner, "old", "new"); err != nil {
		t.Fatalf("PasswordChanged = %v", err)
	}

	// The session that unlocked it goes on.
	if err := f.m.OpenObject(s1, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); err != nil {
		t.Errorf("OpenObject(same session) after the change = %v", err)
	}

	s2 := scoped("s2")
	f.m.UnlockAtLogin(s2, sc.Owner, "old")
	if err := f.m.OpenObject(s2, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); !isLocked(err, ks.Ref) {
		t.Errorf("OpenObject after a login with the old password = %v, want locked", err)
	}
	f.m.UnlockAtLogin(s2, sc.Owner, "new")
	back := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := f.m.OpenObject(s2, sc, back); err != nil || back.ApiKey.Reveal() != "v" {
		t.Errorf("OpenObject after a login with the new password = %q, %v", back.ApiKey.Reveal(), err)
	}

	// A password change it was not bound to leaves it as it was.
	if err := f.m.PasswordChanged(s1, sc.Owner, "wrong", "other"); err == nil {
		t.Error("PasswordChanged with a wrong old password succeeded")
	}
	s3 := scoped("s3")
	f.m.UnlockAtLogin(s3, sc.Owner, "other")
	if err := f.m.OpenObject(s3, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); !isLocked(err, ks.Ref) {
		t.Errorf("OpenObject after a refused change, with its new password = %v, want locked", err)
	}
	f.m.UnlockAtLogin(s3, sc.Owner, "new")
	if err := f.m.OpenObject(s3, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); err != nil {
		t.Errorf("OpenObject after a refused change, with the password it is bound to = %v", err)
	}
}

// Locking closes only what it names: other sessions, keystores and owners stay
// unlocked.
func TestLockingLeavesTheOthersOpen(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	other := sc
	other.Owner = happydns.Identifier{0x02}
	create := func(owner happydns.Identifier, name, p string) Keystore {
		t.Helper()
		ks, err := f.m.CreateKeystore(context.Background(), owner, passphraseKind, name, phrase(p))
		if err != nil {
			t.Fatal(err)
		}
		return ks
	}
	mine, second, theirs := create(sc.Owner, "Mine", "p"), create(sc.Owner, "Second", "p"), create(other.Owner, "Theirs", "q")

	unlockAll := func() {
		t.Helper()
		for _, s := range []string{"s1", "s2"} {
			for _, u := range []struct {
				owner happydns.Identifier
				ref   string
				p     string
			}{{sc.Owner, mine.Ref, "p"}, {sc.Owner, second.Ref, "p"}, {other.Owner, theirs.Ref, "q"}} {
				if err := f.m.UnlockKeystore(scoped(s), u.owner, u.ref, phrase(u.p)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	open := func(step, scope string, owner SecretContext, ref string) {
		t.Helper()
		obj := &managedObject{ApiKey: happydns.NewSecret("v")}
		if err := f.m.SealObjectInto(scoped(scope), owner, obj, Destination{Keystore: ref}); err != nil {
			t.Errorf("after %s: SealObjectInto(%s, %s) = %v, want it still open", step, scope, ref, err)
		}
	}

	unlockAll()
	if err := f.m.LockKeystore(scoped("s1"), sc.Owner, mine.Ref); err != nil {
		t.Fatal(err)
	}
	open("LockKeystore", "s2", sc, mine.Ref)
	open("LockKeystore", "s1", sc, second.Ref)

	unlockAll()
	f.m.LockScope("s1")
	open("LockScope", "s2", sc, mine.Ref)
	open("LockScope", "s2", other, theirs.Ref)

	unlockAll()
	f.m.LockOwner(sc.Owner)
	open("LockOwner", "s1", other, theirs.Ref)
	open("LockOwner", "s2", other, theirs.Ref)
}

// The login opens only the keystores bound to the login password, and only for
// a session.
func TestUnlockAtLoginOnlyOpensPasswordBound(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	same, err := f.m.CreateKeystore(context.Background(), sc.Owner, passphraseKind, "Same as login", phrase("pw"))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := f.m.CreateKeystore(context.Background(), sc.Owner, passwordKind, "Login", json.RawMessage(`{"password":"pw"}`))
	if err != nil {
		t.Fatal(err)
	}

	f.m.UnlockAtLogin(context.Background(), sc.Owner, "pw")
	if len(f.m.unlocked.scopes) != 0 {
		t.Errorf("UnlockAtLogin without a session unlocked %v", f.m.unlocked.scopes)
	}

	s1 := scoped("s1")
	f.m.UnlockAtLogin(s1, sc.Owner, "pw")
	if err := f.m.SealObjectInto(s1, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: same.Ref}); !isLocked(err, same.Ref) {
		t.Errorf("SealObjectInto(passphrase keystore after login) = %v, want locked", err)
	}
	if err := f.m.SealObjectInto(s1, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: bound.Ref}); err != nil {
		t.Errorf("SealObjectInto(password-bound keystore after login) = %v", err)
	}
}

func TestUnlockCacheChecksTheOwner(t *testing.T) {
	c := newUnlockCache()
	safe := &happydns.Safe{Id: happydns.Identifier{0xa0}, Owner: happydns.Identifier{0x01}}
	c.put("s1", safe, fakeUnlocked{id: safe.Id})
	if _, ok := c.get("s1", safe); !ok {
		t.Fatal("get did not find the safe just put")
	}
	other := *safe
	other.Owner = happydns.Identifier{0x02}
	if _, ok := c.get("s1", &other); ok {
		t.Error("get found a safe unlocked by another owner")
	}
}

func slicesContainsKind(kinds []KindInfo, kind string, passwordBound bool) bool {
	for _, k := range kinds {
		if k.Kind == kind {
			return k.PasswordBound == passwordBound && (!passwordBound || len(k.Form) == 1 && k.Form[0].Id == PasswordParam)
		}
	}
	return false
}

// A safe of a backup that only its owner opens is restored without being
// opened.
func TestCheckSafeAcceptsInteractive(t *testing.T) {
	f := interactiveFixture(t)
	ks, err := f.m.CreateKeystore(context.Background(), objectContext().Owner, passphraseKind, "Mine", phrase("p"))
	if err != nil {
		t.Fatal(err)
	}
	safe, _ := f.safes.GetSafe(ks.SafeId)
	if err := f.m.CheckSafe(context.Background(), safe); err != nil {
		t.Errorf("CheckSafe(interactive) = %v", err)
	}
}

func TestInspectCountsLocked(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, passphraseKind, "Mine", phrase("p"))
	if err != nil {
		t.Fatal(err)
	}
	s1 := scoped("s1")
	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("p")); err != nil {
		t.Fatal(err)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatal(err)
	}

	var c Counts
	if err := f.m.Inspect(context.Background(), sc, &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Locked != 1 || c.Unreadable != 0 {
		t.Errorf("Inspect = %+v, want 1 locked", c)
	}
}
