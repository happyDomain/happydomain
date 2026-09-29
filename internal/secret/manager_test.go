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
	"testing"

	"git.happydns.org/happyDomain/model"
)

type managedInner struct {
	Token happydns.Secret `json:"token"`
}

type managedObject struct {
	Host   string          `json:"host"`
	ApiKey happydns.Secret `json:"apikey"`
	Other  happydns.Secret `json:"other"`
	Nested *managedInner   `json:"nested,omitempty"`
}

func sealedFromStorage(t *testing.T, token string) happydns.Secret {
	t.Helper()
	var s happydns.Secret
	if err := json.Unmarshal([]byte(`"`+token+`"`), &s); err != nil || !s.IsSealed() {
		t.Fatalf("%q does not decode as sealed: %v", token, err)
	}
	return s
}

func redacted() happydns.Secret {
	s := happydns.NewSecret("x")
	s.Redact()
	return s
}

func plaintextManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(PolicyPlaintext)
	if err != nil {
		t.Fatalf("NewManager(plaintext): %v", err)
	}
	return m
}

func objectContext() SecretContext {
	return SecretContext{
		Owner:      happydns.Identifier{0x01},
		ObjectType: "provider",
		ObjectId:   "cHJvdmlkZXI",
	}
}

func TestNewManagerRejectsUnknownPolicy(t *testing.T) {
	for _, p := range []Policy{"", "bogus", "Plaintext"} {
		if _, err := NewManager(p); err == nil {
			t.Errorf("NewManager(%q) succeeded, want an error", p)
		}
	}
}

func TestSealPlaintextKeepsTodaysFormat(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{
		Host:   "dns.example.com",
		ApiKey: happydns.NewSecret("my-api-key"),
		Nested: &managedInner{Token: happydns.NewSecret("nested-token")},
	}

	if err := m.SealObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}

	if !obj.ApiKey.IsOpened() || obj.ApiKey.Reveal() != "my-api-key" {
		t.Error("a sealed secret must stay usable: opened, with its clear value")
	}

	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("json.Marshal after SealObject: %v", err)
	}
	want := `{"host":"dns.example.com","apikey":"my-api-key","other":"","nested":{"token":"nested-token"}}`
	if string(b) != want {
		t.Errorf("stored JSON = %s, want %s", b, want)
	}
}

func TestSealLeavesSealedAndOpenedAlone(t *testing.T) {
	m := plaintextManager(t)

	opened := sealedFromStorage(t, "hds:1:AQ:b3BlbmVk")
	opened.SetOpened(opened.Token(), []byte("opened-value"))

	obj := &managedObject{
		ApiKey: sealedFromStorage(t, "hds:1:AQ:c2VhbGVk"),
		Other:  opened,
	}

	if err := m.SealObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}

	if !obj.ApiKey.IsSealed() || obj.ApiKey.Token() != "hds:1:AQ:c2VhbGVk" {
		t.Error("a sealed value must be left as is")
	}
	if !obj.Other.IsOpened() || obj.Other.Token() != "hds:1:AQ:b3BlbmVk" {
		t.Error("an opened value must be left as is")
	}
}

func TestSealRefusesRedacted(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: redacted()}

	err := m.SealObject(context.Background(), objectContext(), obj)
	if !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("SealObject(redacted) = %v, want ErrRedactedSecret", err)
	}
}

func TestSealPlaintextRefusesPrefixedValue(t *testing.T) {
	m := plaintextManager(t)

	for _, value := range []string{
		// Would read back as sealed.
		"hds:1:AQ:c2VhbGVk",
		// Reserved for the formats to come: stored today, it would be
		// misread by the version that introduces them.
		"hds:2:AQ:c2VhbGVk",
		"hds:",
	} {
		obj := &managedObject{ApiKey: happydns.NewSecret(value)}
		if err := m.SealObject(context.Background(), objectContext(), obj); err == nil {
			t.Errorf("SealObject(%q) under the plaintext policy succeeded, want an error", value)
		}
		if !obj.ApiKey.IsClear() {
			t.Errorf("SealObject(%q) failed but changed the secret", value)
		}
	}
}

func TestSealPlaintextRefusesPlaceholder(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: happydns.NewSecret(happydns.RedactedSecret)}

	if err := m.SealObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("the placeholder stored in clear would read back as redacted: want an error")
	}
}

func TestSealPlaintextKeepsValuesMerelyMentioningThePrefix(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: happydns.NewSecret("key-hds:1:abc")}

	if err := m.SealObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	if obj.ApiKey.Token() != "key-hds:1:abc" {
		t.Errorf("Token() = %q, want the value as is", obj.ApiKey.Token())
	}
}

func TestSealRequiresFullContext(t *testing.T) {
	m := plaintextManager(t)

	for name, change := range map[string]func(*SecretContext){
		"owner":       func(sc *SecretContext) { sc.Owner = nil },
		"object type": func(sc *SecretContext) { sc.ObjectType = "" },
		"object id":   func(sc *SecretContext) { sc.ObjectId = "" },
	} {
		sc := objectContext()
		change(&sc)

		obj := &managedObject{ApiKey: happydns.NewSecret("v")}
		if err := m.SealObject(context.Background(), sc, obj); err == nil {
			t.Errorf("SealObject without %s succeeded, want an error", name)
		}
		if !obj.ApiKey.IsClear() {
			t.Errorf("SealObject without %s changed the object", name)
		}
	}
}

func TestOpenLeavesLegacyPlaintext(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: happydns.NewSecret("legacy")}

	if err := m.OpenObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("OpenObject: %v", err)
	}
	if obj.ApiKey.Reveal() != "legacy" {
		t.Errorf("Reveal() = %q, want the legacy value", obj.ApiKey.Reveal())
	}
}

func TestOpenFailsOnUnknownSafe(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: sealedFromStorage(t, "hds:1:AQ:c2VhbGVk")}

	err := m.OpenObject(context.Background(), objectContext(), obj)
	if !errors.Is(err, ErrUnknownSafe) {
		t.Errorf("OpenObject(sealed, no safe) = %v, want ErrUnknownSafe", err)
	}
}

func TestOpenFailsOnMalformedSealed(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: sealedFromStorage(t, "hds:1:no-payload")}

	if err := m.OpenObject(context.Background(), objectContext(), obj); !errors.Is(err, ErrMalformedSealed) {
		t.Errorf("OpenObject(malformed) = %v, want ErrMalformedSealed", err)
	}
}

func TestOpenRefusesRedacted(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: redacted()}

	if err := m.OpenObject(context.Background(), objectContext(), obj); !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("OpenObject(redacted) = %v, want ErrRedactedSecret", err)
	}
}

func TestOpenCopyLeavesOriginalUntouched(t *testing.T) {
	m := plaintextManager(t)
	orig := &managedObject{
		Host:   "h",
		ApiKey: happydns.NewSecret("legacy"),
		Nested: &managedInner{Token: happydns.NewSecret("nested")},
	}

	cp, err := m.OpenCopy(context.Background(), objectContext(), orig)
	if err != nil {
		t.Fatalf("OpenCopy: %v", err)
	}

	c, ok := cp.(*managedObject)
	if !ok {
		t.Fatalf("OpenCopy returned %T, want *managedObject", cp)
	}
	if c == orig || c.Nested == orig.Nested {
		t.Fatal("OpenCopy shares structs with the original")
	}
	if c.Host != "h" || c.ApiKey.Reveal() != "legacy" || c.Nested.Token.Reveal() != "nested" {
		t.Error("OpenCopy lost values")
	}

	c.Nested.Token.Redact()
	if !orig.Nested.Token.IsClear() {
		t.Error("changing the copy changed the original")
	}
}

func TestCheckIncoming(t *testing.T) {
	for name, obj := range map[string]*managedObject{
		"clear":    {ApiKey: happydns.NewSecret("v")},
		"empty":    {},
		"redacted": {ApiKey: redacted()},
	} {
		if err := CheckIncoming(obj); err != nil {
			t.Errorf("CheckIncoming(%s) = %v, want nil", name, err)
		}
	}

	obj := &managedObject{Nested: &managedInner{Token: sealedFromStorage(t, "hds:1:AQ:c2VhbGVk")}}
	err := CheckIncoming(obj)
	if !errors.Is(err, ErrSealedFromClient) {
		t.Errorf("CheckIncoming(sealed) = %v, want ErrSealedFromClient", err)
	}
}

func TestMarshalIncoming(t *testing.T) {
	obj := &managedObject{
		Host:   "h",
		ApiKey: happydns.NewSecret("typed-by-user"),
		Other:  redacted(),
		Nested: &managedInner{Token: sealedFromStorage(t, "hds:1:AQ:c2VhbGVk")},
	}

	b, err := MarshalIncoming(obj)
	if err != nil {
		t.Fatalf("MarshalIncoming: %v", err)
	}

	want := `{"host":"h","apikey":"typed-by-user","other":"` + happydns.RedactedSecret + `","nested":{"token":"hds:1:AQ:c2VhbGVk"}}`
	if string(b) != want {
		t.Errorf("MarshalIncoming = %s, want %s", b, want)
	}

	// Decoding it back gives what the client sent.
	var back managedObject
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.ApiKey.IsClear() || !back.Other.IsRedacted() || !back.Nested.Token.IsSealed() {
		t.Error("MarshalIncoming does not round-trip the client's states")
	}

	// The object itself is left as it was: still clear, never storable.
	if !obj.ApiKey.IsClear() {
		t.Error("MarshalIncoming changed the object")
	}
	if _, err := json.Marshal(obj); !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Error("the object must still refuse to be marshaled")
	}
}

func TestNilManagerFailsClosed(t *testing.T) {
	var m *Manager
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}

	if err := m.SealObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("SealObject on a nil Manager succeeded")
	}
	if err := m.OpenObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("OpenObject on a nil Manager succeeded")
	}
	if _, err := m.OpenCopy(context.Background(), objectContext(), obj); err == nil {
		t.Error("OpenCopy on a nil Manager succeeded")
	}
	if !obj.ApiKey.IsClear() {
		t.Error("a nil Manager changed the object")
	}
}
