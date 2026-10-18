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
