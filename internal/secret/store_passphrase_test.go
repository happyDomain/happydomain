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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// creation returns the parameters creating a passphrase keystore, p
// confirmed.
func creation(p string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{passphraseField: p, passphraseConfirmField: p})
	return b
}

func newPhraseSafe(t *testing.T, store *phraseStore, p string) *happydns.Safe {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	safe := &happydns.Safe{Id: id, Owner: happydns.Identifier{0x01}, Kind: KindPassphrase}
	if err := store.Create(context.Background(), safe, creation(p)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return safe
}

func TestPassphraseStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newPhraseStore()
	safe := newPhraseSafe(t, store, "open sesame")
	sc := objectContext()

	u, err := store.UnlockWith(ctx, safe, phrase("open sesame"))
	if err != nil {
		t.Fatalf("UnlockWith: %v", err)
	}
	payload, err := u.Seal(ctx, sc, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("secret")) {
		t.Error("the payload holds the secret in clear")
	}

	again, err := store.UnlockWith(ctx, safe, phrase("open sesame"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := again.Open(ctx, sc, payload); err != nil || string(got) != "secret" {
		t.Errorf("Open = %q, %v", got, err)
	}
	other := sc
	other.ObjectId = "other"
	if _, err := again.Open(ctx, other, payload); err == nil {
		t.Error("a payload opened for another place")
	}
}

func TestPassphraseStoreRefusesWhatDoesNotOpen(t *testing.T) {
	ctx := context.Background()
	store := newPhraseStore()
	safe := newPhraseSafe(t, store, "open sesame")

	for name, material := range map[string]json.RawMessage{
		"wrong":     phrase("close sesame"),
		"empty":     phrase(""),
		"malformed": json.RawMessage(`[1`),
		"missing":   json.RawMessage(`{}`),
	} {
		if _, err := store.UnlockWith(ctx, safe, material); !errors.Is(err, ErrWrongUnlockMaterial) {
			t.Errorf("UnlockWith(%s) = %v, want ErrWrongUnlockMaterial", name, err)
		}
	}
	if _, err := store.Unlock(ctx, safe); err == nil {
		t.Error("Unlock opened an interactive safe without its owner")
	}
}

func TestPassphraseStoreRefusesASafeOfAnotherKind(t *testing.T) {
	store := newPhraseStore()
	safe := newPhraseSafe(t, store, "open sesame")
	safe.Kind = KindInstance
	if _, err := store.UnlockWith(context.Background(), safe, phrase("open sesame")); err == nil {
		t.Error("a safe of another kind opened")
	}
}

func TestPassphraseStoreCreateNeedsAConfirmedPassphrase(t *testing.T) {
	store := newPhraseStore()
	for name, params := range map[string]json.RawMessage{
		"empty":       creation(""),
		"short":       creation("7 chars"),
		"unconfirmed": phrase("open sesame"),
		"mistyped":    json.RawMessage(`{"passphrase":"open sesame","passphrase-confirm":"open sesam"}`),
		"malformed":   json.RawMessage(`nope`),
		"missing":     json.RawMessage(`{}`),
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x07}, Owner: happydns.Identifier{0x01}, Kind: KindPassphrase}
		if err := store.Create(context.Background(), safe, params); err == nil {
			t.Errorf("Create(%s) succeeded", name)
		}
		if len(safe.Keyring) != 0 {
			t.Errorf("Create(%s) left a keyring", name)
		}
	}
}

// Each safe has its own salt and key, and a key wrapped for a safe does not
// open another.
func TestPassphraseStoreBindsKeyToSafe(t *testing.T) {
	ctx := context.Background()
	store := newPhraseStore()
	a := newPhraseSafe(t, store, "same phrase")
	b := newPhraseSafe(t, store, "same phrase")

	if bytes.Equal(a.Keyring[0].Blob, b.Keyring[0].Blob) || bytes.Equal(a.Keyring[0].Params, b.Keyring[0].Params) {
		t.Error("two safes share their salt or wrapped key")
	}
	moved := *b
	moved.Keyring = a.Keyring
	if _, err := store.UnlockWith(ctx, &moved, phrase("same phrase")); err == nil {
		t.Error("the keyring of a safe opened another safe")
	}
}

// The key is derived as RFC 9106 recommends when memory is constrained.
func TestPassphraseStoreDerivationParameters(t *testing.T) {
	safe := newPhraseSafe(t, newPhraseStore(), "open sesame")
	var p phraseParams
	if err := json.Unmarshal(safe.Keyring[0].Params, &p); err != nil {
		t.Fatal(err)
	}
	if p.Time != 3 || p.Memory != 64*1024 || p.Threads != 4 || len(p.Salt) != 16 {
		t.Errorf("parameters = %+v, want t=3, m=64 MiB, p=4 and a 16 bytes salt", p)
	}
}

// Only the entry of its own kind opens a safe; a corrupt or missing one is
// not taken for a wrong passphrase.
func TestPassphraseStoreReadsOnlyItsOwnEntry(t *testing.T) {
	ctx := context.Background()
	store := newPhraseStore()

	safe := newPhraseSafe(t, store, "open sesame")
	own := safe.Keyring[0]
	safe.Keyring = []happydns.WrappedKeyset{{KEK: KindInstance, Blob: []byte("other")}, own}
	if _, err := store.UnlockWith(ctx, safe, phrase("open sesame")); err != nil {
		t.Errorf("UnlockWith behind an entry of another kind = %v", err)
	}

	for name, keyring := range map[string][]happydns.WrappedKeyset{
		"empty":          nil,
		"other kind":     {{KEK: KindInstance, Params: own.Params, Blob: own.Blob}},
		"corrupt params": {{KEK: KindPassphrase, Params: json.RawMessage(`[1`), Blob: own.Blob}},
	} {
		safe.Keyring = keyring
		if _, err := store.UnlockWith(ctx, safe, phrase("open sesame")); err == nil || errors.Is(err, ErrWrongUnlockMaterial) {
			t.Errorf("UnlockWith(%s) = %v, want an error other than ErrWrongUnlockMaterial", name, err)
		}
	}
}

// The length is counted in characters, not bytes: 8 is enough, 7 accented
// ones (14 bytes) are not.
func TestPassphraseStoreLengthCountsCharacters(t *testing.T) {
	store := newPhraseStore()
	for p, ok := range map[string]bool{
		"12345678": true,
		"ééééééé":  false,
		"éééééééé": true,
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x07}, Owner: happydns.Identifier{0x01}, Kind: KindPassphrase}
		if err := store.Create(context.Background(), safe, creation(p)); (err == nil) != ok {
			t.Errorf("Create(%q) = %v, want accepted=%v", p, err, ok)
		}
	}
}

// Parameters read from storage are bounded: a record tampered with must not
// make an unlock exhaust the memory or time of the server.
func TestPassphraseStoreRefusesUnboundedParameters(t *testing.T) {
	store := newPhraseStore()
	for name, change := range map[string]func(*phraseParams){
		"huge memory":  func(p *phraseParams) { p.Memory = 64 * 1024 * 1024 },
		"tiny memory":  func(p *phraseParams) { p.Memory = 1024 },
		"no time":      func(p *phraseParams) { p.Time = 0 },
		"long time":    func(p *phraseParams) { p.Time = 1000 },
		"no thread":    func(p *phraseParams) { p.Threads = 0 },
		"many threads": func(p *phraseParams) { p.Threads = 255 },
		"short salt":   func(p *phraseParams) { p.Salt = p.Salt[:8] },
	} {
		safe := newPhraseSafe(t, store, "open sesame")
		var p phraseParams
		if err := json.Unmarshal(safe.Keyring[0].Params, &p); err != nil {
			t.Fatal(err)
		}
		change(&p)
		safe.Keyring[0].Params, _ = json.Marshal(p)

		start := time.Now()
		_, err := store.UnlockWith(context.Background(), safe, phrase("open sesame"))
		if err == nil || errors.Is(err, ErrWrongUnlockMaterial) {
			t.Errorf("%s: UnlockWith = %v, want unreadable parameters", name, err)
		}
		if time.Since(start) > time.Second {
			t.Errorf("%s: the key was derived before the parameters were checked", name)
		}
	}
}

func TestPassphraseStoreForms(t *testing.T) {
	store := newPhraseStore()
	create := store.CreateForm()
	if len(create) != 2 || create[0].Id != passphraseField || create[1].Id != passphraseConfirmField {
		t.Fatalf("create form = %+v", create)
	}
	for _, f := range create {
		if !f.Secret || !f.Required {
			t.Errorf("create field %s is not a required secret", f.Id)
		}
	}
	if create[0].Description == "" {
		t.Error("the create form does not warn that a forgotten passphrase loses the credentials")
	}
	if unlock := store.UnlockForm(); len(unlock) != 1 || unlock[0].Id != passphraseField || !unlock[0].Secret || !unlock[0].Required {
		t.Errorf("unlock form = %+v", unlock)
	}
}

func TestPassphraseKindIsBuiltin(t *testing.T) {
	factory, ok := builtinStores[KindPassphrase]
	if !ok {
		t.Fatalf("no builtin store %q", KindPassphrase)
	}
	store, err := factory(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(Interactive); !ok {
		t.Error("passphrase is not Interactive")
	}
	if _, ok := store.(Creator); !ok {
		t.Error("passphrase is not a Creator")
	}
	if opensUnattended(store) {
		t.Error("passphrase opens unattended")
	}
}

func TestPassphraseKeystoreThroughManager(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, KindPassphrase, "Mine", creation("open sesame"))
	if err != nil {
		t.Fatal(err)
	}
	s1 := scoped("session-1")
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}

	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
		t.Fatalf("SealObjectInto(locked) = %v, want locked", err)
	}
	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("close sesame")); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Fatalf("UnlockKeystore(wrong) = %v, want ErrWrongUnlockMaterial", err)
	}
	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, phrase("open sesame")); err != nil {
		t.Fatal(err)
	}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto(unlocked) = %v", err)
	}
	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.OpenObject(s1, sc, back); err != nil || back.ApiKey.Reveal() != "v" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
	other := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.OpenObject(scoped("session-2"), sc, other); !isLocked(err, ks.Ref) {
		t.Errorf("OpenObject(another session) = %v, want locked", err)
	}
}

func newLoginSafe(t *testing.T, store *loginStore, password string) *happydns.Safe {
	t.Helper()
	safe := &happydns.Safe{Id: happydns.Identifier{0x09}, Owner: happydns.Identifier{0x01}, Kind: KindLoginPassword}
	if err := store.Bind(context.Background(), safe, password); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	return safe
}

func TestLoginStoreFollowsThePassword(t *testing.T) {
	ctx := context.Background()
	store := newLoginStore()
	safe := newLoginSafe(t, store, "hunter2")
	sc := objectContext()

	u, err := store.UnlockWithPassword(ctx, safe, "hunter2")
	if err != nil {
		t.Fatalf("UnlockWithPassword: %v", err)
	}
	payload, err := u.Seal(ctx, sc, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UnlockWithPassword(ctx, safe, "hunter3"); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("UnlockWithPassword(wrong) = %v, want ErrWrongUnlockMaterial", err)
	}

	if err := store.Rebind(ctx, safe, "wrong", "new"); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("Rebind(wrong old) = %v, want ErrWrongUnlockMaterial", err)
	}
	if _, err := store.UnlockWithPassword(ctx, safe, "hunter2"); err != nil {
		t.Errorf("a refused Rebind changed the keyring: %v", err)
	}
	if err := store.Rebind(ctx, safe, "hunter2", ""); err == nil {
		t.Error("Rebind to an empty password succeeded")
	}

	if err := store.Rebind(ctx, safe, "hunter2", "hunter3"); err != nil {
		t.Fatalf("Rebind: %v", err)
	}
	if len(safe.Keyring) != 1 {
		t.Fatalf("keyring = %d entries, want 1", len(safe.Keyring))
	}
	if _, err := store.UnlockWithPassword(ctx, safe, "hunter2"); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("the old password still opens: %v", err)
	}
	u, err = store.UnlockWithPassword(ctx, safe, "hunter3")
	if err != nil {
		t.Fatalf("UnlockWithPassword(new) = %v", err)
	}
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "secret" {
		t.Errorf("a secret sealed before Rebind = %q, %v", got, err)
	}
}

func TestLoginStoreUnlocksWithItsForm(t *testing.T) {
	ctx := context.Background()
	store := newLoginStore()
	safe := newLoginSafe(t, store, "hunter2")

	if form := store.UnlockForm(); len(form) != 1 || form[0].Id != PasswordParam || !form[0].Secret {
		t.Errorf("UnlockForm = %+v", form)
	}
	material, _ := json.Marshal(map[string]string{PasswordParam: "hunter2"})
	if _, err := store.UnlockWith(ctx, safe, material); err != nil {
		t.Errorf("UnlockWith = %v", err)
	}
	if _, err := store.UnlockWith(ctx, safe, phrase("hunter2")); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("UnlockWith(wrong field) = %v, want ErrWrongUnlockMaterial", err)
	}
	if _, err := store.Unlock(ctx, safe); err == nil {
		t.Error("Unlock opened an interactive safe without its owner")
	}

	safe.Kind = KindPassphrase
	if _, err := store.UnlockWithPassword(ctx, safe, "hunter2"); err == nil {
		t.Error("a safe of another kind opened")
	}
}

func TestLoginPasswordKindIsBuiltin(t *testing.T) {
	factory, ok := builtinStores[KindLoginPassword]
	if !ok {
		t.Fatalf("no builtin store %q", KindLoginPassword)
	}
	store, err := factory(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(PasswordBound); !ok {
		t.Error("login-password is not PasswordBound")
	}
	if _, ok := store.(Creator); ok {
		t.Error("login-password is a Creator: the Manager binds it to the login password")
	}
	if opensUnattended(store) {
		t.Error("login-password opens unattended")
	}
}

// Through the Manager: unlocked at login, bound to the new password when it
// changes, and no longer opening with the old one.
func TestLoginPasswordKeystoreThroughManager(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	pw, _ := json.Marshal(map[string]string{PasswordParam: "old password"})
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, KindLoginPassword, "Login", pw)
	if err != nil {
		t.Fatal(err)
	}

	s1 := scoped("s1")
	f.m.UnlockAtLogin(s1, sc.Owner, "old password")
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto after login = %v", err)
	}
	token := obj.ApiKey.Token()

	if err := f.m.PasswordChanged(s1, sc.Owner, "old password", "new password"); err != nil {
		t.Fatalf("PasswordChanged = %v", err)
	}

	s2 := scoped("s2")
	f.m.UnlockAtLogin(s2, sc.Owner, "old password")
	if err := f.m.OpenObject(s2, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); !isLocked(err, ks.Ref) {
		t.Errorf("OpenObject after a login with the old password = %v, want locked", err)
	}
	f.m.UnlockAtLogin(s2, sc.Owner, "new password")
	back := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := f.m.OpenObject(s2, sc, back); err != nil || back.ApiKey.Reveal() != "v" {
		t.Errorf("OpenObject after a login with the new password = %q, %v", back.ApiKey.Reveal(), err)
	}
}

// What creates such a keystore opens it: created during a session, it is
// unlocked for that session.
func TestPhraseKeystoresUnlockedForTheSessionCreatingThem(t *testing.T) {
	pw, _ := json.Marshal(map[string]string{PasswordParam: "login password"})
	for kind, params := range map[string]json.RawMessage{KindPassphrase: creation("open sesame"), KindLoginPassword: pw} {
		f := interactiveFixture(t)
		sc := objectContext()
		s1 := scoped("s1")
		ks, err := f.m.CreateKeystore(s1, sc.Owner, kind, "Mine", params)
		if err != nil {
			t.Fatal(err)
		}
		if ks.Locked {
			t.Errorf("%s: created keystore = %+v, want it unlocked", kind, ks)
		}
		if err := f.m.SealObjectInto(s1, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); err != nil {
			t.Errorf("%s: SealObjectInto(session creating it) = %v", kind, err)
		}
	}
}

// After a password reset, a keystore stays bound to the password it was:
// unlocked with it in the session changing the password, it follows the new
// one, so that its owner no longer needs the previous one.
func TestLoginPasswordKeystoreFollowsFromTheSession(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	pw, _ := json.Marshal(map[string]string{PasswordParam: "bound password"})
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, KindLoginPassword, "Login", pw)
	if err != nil {
		t.Fatal(err)
	}

	// After a reset, the old password is not the one the keystore is bound
	// to: locked in this session, it does not follow.
	s1 := scoped("s1")
	if err := f.m.PasswordChanged(s1, sc.Owner, "reset password", "new password"); err == nil {
		t.Error("PasswordChanged bound to the new password a keystore neither the old one nor the session opens")
	}
	s2 := scoped("s2")
	f.m.UnlockAtLogin(s2, sc.Owner, "new password")
	if err := f.m.SealObjectInto(s2, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
		t.Fatalf("SealObjectInto after a login with the new password = %v, want locked", err)
	}

	unlock, _ := json.Marshal(map[string]string{PasswordParam: "bound password"})
	if err := f.m.UnlockKeystore(s1, sc.Owner, ks.Ref, unlock); err != nil {
		t.Fatal(err)
	}
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := f.m.SealObjectInto(s1, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatal(err)
	}
	if err := f.m.PasswordChanged(s1, sc.Owner, "reset password", "new password"); err != nil {
		t.Fatalf("PasswordChanged(unlocked in the session) = %v", err)
	}
	s3 := scoped("s3")
	f.m.UnlockAtLogin(s3, sc.Owner, "new password")
	if err := f.m.SealObjectInto(s3, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); err != nil {
		t.Errorf("SealObjectInto after a login with the new password = %v", err)
	}
	// The key is kept: what was sealed before still opens.
	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.OpenObject(s3, sc, back); err != nil || back.ApiKey.Reveal() != "v" {
		t.Errorf("a secret sealed before PasswordChanged = %q, %v", back.ApiKey.Reveal(), err)
	}
}

// Unlocked in another session only, a keystore bound to another password
// does not follow: the session changing the password never opened it.
func TestLoginPasswordKeystoreDoesNotFollowFromAnotherSession(t *testing.T) {
	f := interactiveFixture(t)
	sc := objectContext()
	pw, _ := json.Marshal(map[string]string{PasswordParam: "bound password"})
	ks, err := f.m.CreateKeystore(context.Background(), sc.Owner, KindLoginPassword, "Login", pw)
	if err != nil {
		t.Fatal(err)
	}

	if err := f.m.UnlockKeystore(scoped("other"), sc.Owner, ks.Ref, pw); err != nil {
		t.Fatal(err)
	}
	if err := f.m.PasswordChanged(scoped("s1"), sc.Owner, "reset password", "new password"); err == nil {
		t.Error("PasswordChanged rebound a keystore unlocked only in another session")
	}
	s2 := scoped("s2")
	f.m.UnlockAtLogin(s2, sc.Owner, "new password")
	if err := f.m.SealObjectInto(s2, sc, &managedObject{ApiKey: happydns.NewSecret("v")}, Destination{Keystore: ks.Ref}); !isLocked(err, ks.Ref) {
		t.Errorf("SealObjectInto after a login with the new password = %v, want locked", err)
	}
}

func TestLoginStoreRebindUnlocked(t *testing.T) {
	ctx := context.Background()
	store := newLoginStore()
	safe := newLoginSafe(t, store, "bound")
	sc := objectContext()

	u, err := store.UnlockWithPassword(ctx, safe, "bound")
	if err != nil {
		t.Fatalf("UnlockWithPassword: %v", err)
	}
	payload, err := u.Seal(ctx, sc, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.RebindUnlocked(ctx, safe, u, ""); err == nil {
		t.Error("RebindUnlocked to an empty password succeeded")
	}
	if err := store.RebindUnlocked(ctx, safe, struct{ UnlockedSafe }{u}, "new"); err == nil {
		t.Error("RebindUnlocked accepted a safe this store did not open")
	}
	if _, err := store.UnlockWithPassword(ctx, safe, "bound"); err != nil {
		t.Errorf("a refused RebindUnlocked changed the keyring: %v", err)
	}

	if err := store.RebindUnlocked(ctx, safe, u, "new"); err != nil {
		t.Fatalf("RebindUnlocked: %v", err)
	}
	if len(safe.Keyring) != 1 {
		t.Fatalf("keyring = %d entries, want 1", len(safe.Keyring))
	}
	if _, err := store.UnlockWithPassword(ctx, safe, "bound"); !errors.Is(err, ErrWrongUnlockMaterial) {
		t.Errorf("the previous password still opens: %v", err)
	}
	u, err = store.UnlockWithPassword(ctx, safe, "new")
	if err != nil {
		t.Fatalf("UnlockWithPassword(new) = %v", err)
	}
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "secret" {
		t.Errorf("a secret sealed before RebindUnlocked = %q, %v", got, err)
	}
}
