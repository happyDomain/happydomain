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
	"net/http"
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
	m, err := NewManager(Config{Policy: PolicyPlaintext})
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
		if _, err := NewManager(Config{Policy: p}); err == nil {
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

func fieldContext(field string) SecretContext {
	sc := objectContext()
	sc.Field = field
	return sc
}

func TestSealKeepsOpenedSecretOfTheSameContext(t *testing.T) {
	m := plaintextManager(t)

	opened := sealedFromStorage(t, "hds:1:AQ:b3BlbmVk")
	opened.SetOpened(opened.Token(), []byte("opened-value"), fieldContext("other").binding())
	obj := &managedObject{Other: opened}

	if err := m.SealObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}
	if !obj.Other.IsOpened() || obj.Other.Token() != "hds:1:AQ:b3BlbmVk" {
		t.Error("an opened value bound to this very field must be left as is")
	}
}

func TestSealResealsOpenedSecretOfAnotherContext(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: happydns.NewSecret("my-api-key")}

	if err := m.SealObject(context.Background(), objectContext(), obj); err != nil {
		t.Fatalf("SealObject: %v", err)
	}

	// The same object, copied to another owner: its token must be bound to
	// where it now lives, not to where it was sealed first.
	moved := objectContext()
	moved.Owner = happydns.Identifier{0x02}
	if err := m.SealObject(context.Background(), moved, obj); err != nil {
		t.Fatalf("SealObject under another context: %v", err)
	}

	want := moved
	want.Field = "apikey"
	if obj.ApiKey.Binding() != want.binding() {
		t.Error("an opened value moved to another context kept its former binding")
	}
	if obj.ApiKey.Reveal() != "my-api-key" {
		t.Errorf("resealing lost the value: %q", obj.ApiKey.Reveal())
	}
}

func TestSealRefusesSealedSecretItCannotOpen(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{ApiKey: sealedFromStorage(t, "hds:1:AQ:c2VhbGVk")}

	// Kept as is, it would be stored where nobody knows whether it opens.
	// Without safe storage, whether its safe exists is unknown: a fault of
	// the configuration, not a value known lost.
	err := m.SealObject(context.Background(), objectContext(), obj)
	if !errors.Is(err, ErrSafeUnavailable) {
		t.Errorf("SealObject(sealed, no safe storage) = %v, want ErrSafeUnavailable", err)
	}
	if !obj.ApiKey.IsSealed() || obj.ApiKey.Token() != "hds:1:AQ:c2VhbGVk" {
		t.Error("a failed SealObject changed the sealed value")
	}
}

func TestSealObjectIsAllOrNothing(t *testing.T) {
	m := plaintextManager(t)
	obj := &managedObject{
		ApiKey: happydns.NewSecret("k"),
		Other:  redacted(),
	}

	if err := m.SealObject(context.Background(), objectContext(), obj); !errors.Is(err, ErrRedactedSecret) {
		t.Fatalf("SealObject = %v, want ErrRedactedSecret", err)
	}
	if !obj.ApiKey.IsClear() {
		t.Error("a failed SealObject left the object half sealed")
	}
}

func TestSealRefuses(t *testing.T) {
	m := plaintextManager(t)

	for name, tc := range map[string]struct {
		s    happydns.Secret
		want error
	}{
		"redacted": {redacted(), ErrRedactedSecret},
		// A clear value carrying the sealed prefix would read back as sealed.
		"sealed prefix": {happydns.NewSecret("hds:1:AQ:c2VhbGVk"), nil},
		// A clear value equal to the redacted sentinel would read back as redacted.
		"redacted sentinel": {happydns.NewSecret(happydns.RedactedSecret), nil},
	} {
		obj := &managedObject{ApiKey: tc.s}
		err := m.SealObject(context.Background(), objectContext(), obj)
		if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
			t.Errorf("SealObject(%s) = %v, want an error (%v)", name, err, tc.want)
		}
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

func TestOpenRefuses(t *testing.T) {
	m := plaintextManager(t)

	for name, tc := range map[string]struct {
		s    happydns.Secret
		want error
	}{
		"sealed, no safe storage": {sealedFromStorage(t, "hds:1:AQ:c2VhbGVk"), ErrSafeUnavailable},
		"malformed":               {sealedFromStorage(t, "hds:1:no-payload"), ErrMalformedSealed},
		"redacted":                {redacted(), ErrRedactedSecret},
	} {
		obj := &managedObject{ApiKey: tc.s}
		if err := m.OpenObject(context.Background(), objectContext(), obj); !errors.Is(err, tc.want) {
			t.Errorf("OpenObject(%s) = %v, want %v", name, err, tc.want)
		}
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

type managedAuth struct {
	Token happydns.Secret `json:"token"`
}

type managedHolder struct {
	Auth any `json:"auth"`
}

type managedDynamic struct {
	Holder *managedHolder `json:"holder"`
}

func TestOpenCopyCopiesThroughInterfaces(t *testing.T) {
	m := plaintextManager(t)
	auth := &managedAuth{Token: happydns.NewSecret("legacy")}
	orig := &managedDynamic{Holder: &managedHolder{Auth: auth}}

	cp, err := m.OpenCopy(context.Background(), objectContext(), orig)
	if err != nil {
		t.Fatalf("OpenCopy: %v", err)
	}

	c := cp.(*managedDynamic)
	if c.Holder == orig.Holder || c.Holder.Auth.(*managedAuth) == auth {
		t.Fatal("OpenCopy shares with the original a struct reaching a Secret through an interface")
	}
	if c.Holder.Auth.(*managedAuth).Token.Reveal() != "legacy" {
		t.Error("OpenCopy lost the value held through an interface")
	}
}

type managedEmbeddedPointer struct {
	*managedInner
}

func TestOpenCopyCopiesEmbeddedUnexportedPointer(t *testing.T) {
	m := plaintextManager(t)
	orig := &managedEmbeddedPointer{&managedInner{Token: happydns.NewSecret("legacy")}}

	cp, err := m.OpenCopy(context.Background(), objectContext(), orig)
	if err != nil {
		t.Fatalf("OpenCopy: %v", err)
	}

	c := cp.(*managedEmbeddedPointer)
	if c.managedInner == orig.managedInner {
		t.Fatal("OpenCopy shares the embedded struct with the original")
	}
	if c.Token.Reveal() != "legacy" {
		t.Error("OpenCopy lost the value of the embedded struct")
	}
}

func TestOpenCopyRefusesCycleThroughSecrets(t *testing.T) {
	m := plaintextManager(t)
	a := &walkNode{Name: "a", Token: happydns.NewSecret("legacy")}
	a.Next = &walkNode{Name: "b", Next: a}

	// Must return, not overflow the stack of the whole server.
	if _, err := m.OpenCopy(context.Background(), objectContext(), a); !errors.Is(err, ErrUnsupportedSecret) {
		t.Errorf("OpenCopy on a cycle holding secrets = %v, want ErrUnsupportedSecret", err)
	}
}

func TestOpenCopyRefusesSharedSecretHolder(t *testing.T) {
	type aliased struct {
		A *managedInner `json:"a"`
		B *managedInner `json:"b"`
	}

	m := plaintextManager(t)
	shared := &managedInner{Token: happydns.NewSecret("legacy")}

	if _, err := m.OpenCopy(context.Background(), objectContext(), &aliased{A: shared, B: shared}); !errors.Is(err, ErrUnsupportedSecret) {
		t.Errorf("OpenCopy with a secret holder reached twice = %v, want ErrUnsupportedSecret", err)
	}
}

type managedWithClient struct {
	ApiKey happydns.Secret `json:"apikey"`
	Client *http.Client    `json:"client"`
	Any    any             `json:"any"`
	Ring   *walkPlainNode  `json:"ring"`
}

func TestOpenCopySharesWhatHoldsNoSecret(t *testing.T) {
	m := plaintextManager(t)

	ring := &walkPlainNode{Name: "a"}
	ring.Next = &walkPlainNode{Name: "b", Next: ring}

	orig := &managedWithClient{
		ApiKey: happydns.NewSecret("legacy"),
		// Holds locks and connection pools, reached through interfaces:
		// copying it would copy its mutexes.
		Client: &http.Client{Transport: &http.Transport{}},
		Any:    &walkPlainNode{Name: "any"},
		Ring:   ring,
	}

	cp, err := m.OpenCopy(context.Background(), objectContext(), orig)
	if err != nil {
		t.Fatalf("OpenCopy: %v", err)
	}

	c := cp.(*managedWithClient)
	if c.Client != orig.Client || c.Client.Transport != orig.Client.Transport {
		t.Error("OpenCopy copied an http.Client that holds no secret")
	}
	if c.Any != orig.Any || c.Ring != orig.Ring {
		t.Error("OpenCopy copied structs that hold no secret")
	}
	if c.ApiKey.Reveal() != "legacy" {
		t.Error("OpenCopy lost the secret")
	}
}

func TestMarshalIncomingNeverTouchesTheObject(t *testing.T) {
	obj := &managedObject{ApiKey: happydns.NewSecret("typed-by-user")}

	// Another goroutine reading the object while it is encoded must never
	// see it change: under -race, any write is reported.
	done := make(chan struct{})
	changed := make(chan bool, 1)
	go func() {
		defer close(changed)
		for {
			select {
			case <-done:
				return
			default:
				if !obj.ApiKey.IsClear() {
					changed <- true
					return
				}
			}
		}
	}()

	for range 1000 {
		if _, err := MarshalIncoming(obj); err != nil {
			t.Fatalf("MarshalIncoming: %v", err)
		}
	}
	close(done)

	if <-changed {
		t.Error("MarshalIncoming changed the object while encoding it")
	}
}
