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
	"os"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

func instanceManagers(t *testing.T) (instance, plaintext *Manager, store *secrettest.Safes) {
	t.Helper()
	key := testInstanceKey(t)
	store = secrettest.NewSafes()

	var err error
	instance, err = NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err = NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestNewManagerInstanceNeedsKeyAndStore(t *testing.T) {
	key := testInstanceKey(t)
	if _, err := NewManager(Config{Policy: PolicyInstance, Safes: secrettest.NewSafes()}); err == nil {
		t.Error("the instance policy without a key was accepted")
	}
	if _, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key}); err == nil {
		t.Error("the instance policy without safe storage was accepted")
	}
	if _, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key}); err == nil {
		t.Error("an instance key without safe storage was accepted")
	}
}

func TestInstanceSealOpenRoundTrip(t *testing.T) {
	m, _, store := instanceManagers(t)
	sc := objectContext()

	obj := &managedObject{
		Host:   "h",
		ApiKey: happydns.NewSecret("my-api-key"),
		Nested: &managedInner{Token: happydns.NewSecret("nested-token")},
	}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}

	safe, err := store.GetSafeByOwner(sc.Owner, KindInstance)
	if err != nil {
		t.Fatalf("no instance safe created for the owner: %v", err)
	}

	stored, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(stored), "my-api-key") || strings.Contains(string(stored), "nested-token") {
		t.Fatalf("stored form holds a clear value: %s", stored)
	}
	if !strings.HasPrefix(obj.ApiKey.Token(), "hds:1:"+safe.Id.String()+":") {
		t.Errorf("token = %q, want it to name the safe %s", obj.ApiKey.Token(), safe.Id.String())
	}
	if obj.ApiKey.Reveal() != "my-api-key" {
		t.Error("a sealed secret must stay usable")
	}

	var back managedObject
	if err := json.Unmarshal(stored, &back); err != nil {
		t.Fatal(err)
	}
	if !back.ApiKey.IsSealed() {
		t.Fatal("the stored value does not read back as sealed")
	}
	if err := m.OpenObject(context.Background(), sc, &back); err != nil {
		t.Fatalf("OpenObject: %v", err)
	}
	if back.ApiKey.Reveal() != "my-api-key" || back.Nested.Token.Reveal() != "nested-token" {
		t.Errorf("opened = %q, %q", back.ApiKey.Reveal(), back.Nested.Token.Reveal())
	}
}

func TestInstanceSealIsRandomized(t *testing.T) {
	m, _, _ := instanceManagers(t)
	a := &managedObject{ApiKey: happydns.NewSecret("same")}
	b := &managedObject{ApiKey: happydns.NewSecret("same")}
	_ = m.SealObject(context.Background(), objectContext(), a)
	_ = m.SealObject(context.Background(), objectContext(), b)

	if a.ApiKey.Token() == b.ApiKey.Token() {
		t.Error("two seals of the same value gave the same token")
	}
}

func sealOne(t *testing.T, m *Manager, sc SecretContext, value string) string {
	t.Helper()
	obj := &managedObject{ApiKey: happydns.NewSecret(value)}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatal(err)
	}
	return obj.ApiKey.Token()
}

func TestInstanceOpenIsBoundToItsPlace(t *testing.T) {
	m, _, _ := instanceManagers(t)
	sc := objectContext()
	token := sealOne(t, m, sc, "v")

	// Copied to another field of the same object.
	obj := &managedObject{Other: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), sc, obj); err == nil {
		t.Error("a token copied to another field opened")
	}

	// Copied to another object of the same owner.
	other := sc
	other.ObjectId = "another-provider"
	obj = &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), other, obj); err == nil {
		t.Error("a token copied to another object opened")
	}

	// Copied to an object of another owner.
	stranger := sc
	stranger.Owner = happydns.Identifier{0x99}
	_ = sealOne(t, m, stranger, "their own") // so that they have a safe
	obj = &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), stranger, obj); err == nil {
		t.Error("a token copied to another owner opened")
	}

	// In its place, it opens.
	obj = &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), sc, obj); err != nil || obj.ApiKey.Reveal() != "v" {
		t.Errorf("OpenObject in place = %v", err)
	}
}

func TestInstanceOpenUnknownSafe(t *testing.T) {
	m, _, _ := instanceManagers(t)
	obj := &managedObject{ApiKey: sealedFromStorage(t, FormatSealed(happydns.Identifier{0x42}, []byte("payload")))}

	if err := m.OpenObject(context.Background(), objectContext(), obj); !errors.Is(err, ErrUnknownSafe) {
		t.Errorf("OpenObject(unknown safe) = %v, want ErrUnknownSafe", err)
	}
}

func TestPlaintextPolicyStillOpensSealed(t *testing.T) {
	instance, plaintext, _ := instanceManagers(t)
	sc := objectContext()
	token := sealOne(t, instance, sc, "sealed-earlier")

	// Back to plaintext: what was sealed still opens...
	obj := &managedObject{ApiKey: sealedFromStorage(t, token), Other: happydns.NewSecret("typed-now")}
	if err := plaintext.OpenObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("OpenObject under plaintext: %v", err)
	}
	if obj.ApiKey.Reveal() != "sealed-earlier" {
		t.Error("the sealed value did not open")
	}

	// ...and new values are stored in clear.
	if err := plaintext.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(obj)
	if !strings.Contains(string(b), `"other":"typed-now"`) {
		t.Errorf("new value not stored in clear under plaintext: %s", b)
	}
}

func TestInstanceSealsLegacyPlaintext(t *testing.T) {
	m, _, _ := instanceManagers(t)

	// As read from storage: an unprefixed value is a clear Secret.
	var obj managedObject
	if err := json.Unmarshal([]byte(`{"apikey":"legacy"}`), &obj); err != nil {
		t.Fatal(err)
	}
	if err := m.SealObject(context.Background(), objectContext(), &obj); err != nil {
		t.Fatal(err)
	}
	if !IsSealed(obj.ApiKey.Token()) {
		t.Errorf("the legacy value was not sealed: %q", obj.ApiKey.Token())
	}
}

func TestOpenWithoutInstanceKey(t *testing.T) {
	instance, _, store := instanceManagers(t)
	token := sealOne(t, instance, objectContext(), "v")

	noKey, err := NewManager(Config{Policy: PolicyPlaintext, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	obj := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := noKey.OpenObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("an instance safe opened without the instance keyset")
	}
}

// The committed safe (testdata/safe.json), wrapped by the committed instance
// keyset, opens a committed token to a committed value, in a fixed context:
// a change to the sealed format, the associated data or the wrapping fails
// here.
func TestSealedTokenGolden(t *testing.T) {
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
	// Already in the database: its identifier predates the length rule
	// CreateSafe enforces, and is part of the associated data.
	store := secrettest.NewSafes()
	store.Put(safe)

	m, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}

	token, err := os.ReadFile("testdata/sealed-token.txt")
	if err != nil {
		t.Fatal(err)
	}

	obj := &managedObject{ApiKey: sealedFromStorage(t, strings.TrimSpace(string(token)))}
	sc := SecretContext{Owner: safe.Owner, ObjectType: "provider", ObjectId: "golden-provider"}
	if err := m.OpenObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("OpenObject(golden token): %v", err)
	}
	if obj.ApiKey.Reveal() != "golden api key" {
		t.Errorf("golden token opens to %q", obj.ApiKey.Reveal())
	}
}

// storedAndBack encodes obj as storage would, and decodes it back sealed.
func storedAndBack(t *testing.T, obj *managedObject) *managedObject {
	t.Helper()
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var back managedObject
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	return &back
}

func TestInstanceSealMovedObjectOpensWhereItLives(t *testing.T) {
	m, _, _ := instanceManagers(t)
	from := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(context.Background(), from, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}

	// Copied, opened, to another object of another owner, then stored.
	for name, to := range map[string]SecretContext{
		"object": {Owner: from.Owner, ObjectType: from.ObjectType, ObjectId: "YW5vdGhlcg"},
		"owner":  {Owner: happydns.Identifier{0x02}, ObjectType: from.ObjectType, ObjectId: from.ObjectId},
	} {
		moved := *obj
		if err := m.SealObject(context.Background(), to, &moved); err != nil {
			t.Fatalf("%s: SealObject: %v", name, err)
		}

		back := storedAndBack(t, &moved)
		if err := m.OpenObject(context.Background(), to, back); err != nil {
			t.Fatalf("%s: the moved secret does not open where it now lives: %v", name, err)
		}
		if back.ApiKey.Reveal() != "my-api-key" {
			t.Errorf("%s: opened = %q", name, back.ApiKey.Reveal())
		}
	}
}

func TestInstanceSealKeepsSealedThatOpensHere(t *testing.T) {
	m, _, _ := instanceManagers(t)
	sc := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	back := storedAndBack(t, obj)

	if err := m.SealObject(context.Background(), sc, back); err != nil {
		t.Fatalf("SealObject on its own sealed value: %v", err)
	}
	if !back.ApiKey.IsSealed() || back.ApiKey.Token() != obj.ApiKey.Token() {
		t.Error("a sealed value that opens here must be kept as is, still sealed")
	}
}

func TestInstanceSealRefusesSealedFromElsewhere(t *testing.T) {
	m, _, _ := instanceManagers(t)
	from := objectContext()

	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}
	if err := m.SealObject(context.Background(), from, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	back := storedAndBack(t, obj)

	// Sealed, never opened, moved to another object: its value is unknown
	// and it cannot open there.
	to := from
	to.ObjectId = "YW5vdGhlcg"
	if err := m.SealObject(context.Background(), to, back); err == nil {
		t.Fatal("SealObject kept a sealed value that does not open where it is stored")
	}
	if !back.ApiKey.IsSealed() || back.ApiKey.Token() != obj.ApiKey.Token() {
		t.Error("a failed SealObject changed the sealed value")
	}
}

func TestInstanceOpenObjectIsAllOrNothing(t *testing.T) {
	m, _, _ := instanceManagers(t)
	sc := objectContext()

	obj := &managedObject{
		ApiKey: happydns.NewSecret("my-api-key"),
		Other:  happydns.NewSecret("other"),
	}
	if err := m.SealObject(context.Background(), sc, obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	back := storedAndBack(t, obj)

	// ApiKey opens, then Other holds the token of ApiKey: it does not open
	// there.
	back.Other = back.ApiKey

	if err := m.OpenObject(context.Background(), sc, back); err == nil {
		t.Fatal("OpenObject opened a token bound to another field")
	}
	if !back.ApiKey.IsSealed() || back.ApiKey.Reveal() != "" {
		t.Error("a failed OpenObject left the object half opened")
	}
}
