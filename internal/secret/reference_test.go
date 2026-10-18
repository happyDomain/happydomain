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
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// referenceKind is a personal keystore keeping no secret: its owner gives,
// at creation, the secrets it resolves, by reference, standing for a file of
// theirs.
const referenceKind = "test-reference"

// referenceOpens counts the secrets a referenceStore resolved.
var referenceOpens atomic.Int64

type referenceStore struct{}

func (referenceStore) OpensUnattended() bool { return true }

func (referenceStore) ReferenceHint() string { return "the key of the secret in your file" }

func (referenceStore) Reference(payload []byte) (string, error) {
	if len(payload) == 0 || string(payload) == "unreadable" {
		return "", errors.New("unreadable reference")
	}
	return string(payload), nil
}

func (referenceStore) CreateForm() []happydns.Field {
	return []happydns.Field{{Id: "values", Type: "string", Required: true}}
}

func (referenceStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p struct {
		Values map[string]string `json:"values"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if len(p.Values) == 0 {
		return errors.New("no values")
	}
	b, err := json.Marshal(p.Values)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: referenceKind, Params: b}}
	return nil
}

func (referenceStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != referenceKind || len(safe.Keyring) == 0 {
		return nil, errors.New("not a reference test safe")
	}
	var values map[string]string
	if err := json.Unmarshal(safe.Keyring[0].Params, &values); err != nil {
		return nil, err
	}
	return referenceUnlocked(values), nil
}

type referenceUnlocked map[string]string

func (u referenceUnlocked) Seal(_ context.Context, _ SecretContext, ref []byte) ([]byte, error) {
	if _, ok := u[string(ref)]; !ok {
		return nil, errors.New("unknown reference")
	}
	return ref, nil
}

func (u referenceUnlocked) Open(_ context.Context, _ SecretContext, payload []byte) ([]byte, error) {
	referenceOpens.Add(1)
	v, ok := u[string(payload)]
	if !ok {
		return nil, errors.New("unknown reference")
	}
	return []byte(v), nil
}

// createReference creates a keystore of referenceKind named name for owner,
// resolving each reference of values to its secret.
func (f keystoreFixture) createReference(t *testing.T, owner happydns.Identifier, name string, values map[string]string) Keystore {
	t.Helper()
	params, _ := json.Marshal(map[string]any{"values": values})
	ks, err := f.m.CreateKeystore(context.Background(), owner, referenceKind, name, params)
	if err != nil {
		t.Fatalf("CreateKeystore(%s): %v", referenceKind, err)
	}
	return ks
}

func TestReferenceSealsAndOpens(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	if safeOf(t, obj.ApiKey) != refs.Ref {
		t.Fatalf("sealed into %s, want %s", safeOf(t, obj.ApiKey), refs.Ref)
	}

	// Its clear value would be the reference: it is left sealed, so that
	// what uses the object right away resolves it.
	if !obj.ApiKey.IsSealed() {
		t.Error("a reference sealed is left opened")
	}
	opened, err := f.m.OpenCopy(ctx, sc, obj)
	if err != nil || opened.(*managedObject).ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenCopy right after sealing = %v; want the secret", err)
	}

	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenObject = %q, %v; want the secret the reference stands for", back.ApiKey.Reveal(), err)
	}

	// What the store refuses is not sealed, and the owner is told why: it
	// is theirs to fix.
	bad := &managedObject{ApiKey: happydns.NewSecret("nope")}
	err = f.m.SealObjectInto(ctx, sc, bad, Destination{Keystore: refs.Ref})
	if !errors.Is(err, ErrUnresolvedReference) || !IsReferenceError(err) || !strings.Contains(err.Error(), "unknown reference") {
		t.Errorf("SealObjectInto(unknown reference) = %v, want ErrUnresolvedReference with the reason of the store", err)
	}
}

func TestReferenceSealsValue(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	sc.Field = "key"
	token, err := f.m.SealValueInto(ctx, sc, "ovh/key", Destination{Keystore: refs.Ref})
	if err != nil {
		t.Fatalf("SealValueInto: %v", err)
	}
	if !IsSealed(token) {
		t.Fatalf("SealValueInto(reference) = %q, want it sealed", token)
	}
	if clear, err := f.m.OpenValue(ctx, sc, token); err != nil || clear != "s3cr3t" {
		t.Errorf("OpenValue = %q, %v; want the secret the reference stands for", clear, err)
	}
}

func TestIsReferenceError(t *testing.T) {
	for _, err := range []error{ErrUnresolvedReference, ErrReferencesOnly, ErrReferenceAsSecret} {
		if !IsReferenceError(fmt.Errorf("field: %w", err)) {
			t.Errorf("IsReferenceError(%v) = false", err)
		}
	}
	if IsReferenceError(errors.New("unable to reach the database")) {
		t.Error("IsReferenceError(another error) = true")
	}
}

func TestReferenceKeystoreDescribed(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	owner := objectContext().Owner
	refs := f.createReference(t, owner, "My file", map[string]string{"k": "v"})

	if !refs.References || refs.ReferenceHint == "" {
		t.Errorf("created keystore = %+v, want it keeping references, with a hint", refs)
	}

	list, err := f.m.Keystores(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, ks := range list {
		if want := ks.Ref == refs.Ref; ks.References != want {
			t.Errorf("keystore %s: References = %v, want %v", ks.Ref, ks.References, want)
		}
	}

	for _, k := range f.m.CreatableKinds() {
		if want := k.Kind == referenceKind; k.References != want || (want && k.ReferenceHint == "") {
			t.Errorf("kind %s = %+v", k.Kind, k)
		}
	}
}

func TestReferenceKeystoreImportsNothing(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"s3cr3t": "other"})

	obj := &managedObject{ApiKey: happydns.NewSecret("s3cr3t")}
	if err := f.m.SealObject(ctx, sc, obj); err != nil {
		t.Fatal(err)
	}

	// The secret itself, sealed in the instance keystore, would be taken
	// for a reference: it does not move.
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	err := f.m.SealObjectInto(ctx, sc, stored, Destination{Keystore: refs.Ref})
	if !errors.Is(err, ErrReferencesOnly) {
		t.Fatalf("SealObjectInto(reference keystore) = %v, want ErrReferencesOnly", err)
	}
	if !stored.ApiKey.IsSealed() || stored.ApiKey.Token() != obj.ApiKey.Token() {
		t.Error("the refused object changed")
	}

	if err := f.m.CheckMigrationTarget(ctx, sc.Owner, refs.Ref); !errors.Is(err, ErrReferencesOnly) {
		t.Errorf("CheckMigrationTarget(reference keystore) = %v, want ErrReferencesOnly", err)
	}
	if err := f.m.CheckMigrationTarget(ctx, sc.Owner, KindInstance); err != nil {
		t.Errorf("CheckMigrationTarget(instance) = %v", err)
	}
}

func TestReferenceKeystoreMigratesOut(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatal(err)
	}

	// Out of it, the secret itself goes: that is what leaving it means.
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.SealObjectInto(ctx, sc, stored, Destination{Keystore: KindInstance}); err != nil {
		t.Fatalf("SealObjectInto(instance) = %v", err)
	}
	back := &managedObject{ApiKey: sealedFromStorage(t, stored.ApiKey.Token())}
	if err := f.m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("moved out = %q, %v; want the secret", back.ApiKey.Reveal(), err)
	}
}

func TestShowReferences(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	inRefs := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, inRefs, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatal(err)
	}
	inInstance := &managedObject{ApiKey: happydns.NewSecret("value")}
	if err := f.m.SealObject(ctx, sc, inInstance); err != nil {
		t.Fatal(err)
	}

	before := referenceOpens.Load()
	shown := &managedObject{
		ApiKey: sealedFromStorage(t, inRefs.ApiKey.Token()),
		Other:  sealedFromStorage(t, inInstance.ApiKey.Token()),
	}
	if err := f.m.ShowReferences(ctx, sc, shown); err != nil {
		t.Fatalf("ShowReferences: %v", err)
	}
	if referenceOpens.Load() != before {
		t.Error("ShowReferences resolved a reference")
	}
	if !shown.ApiKey.IsReference() {
		t.Fatal("the reference is not shown")
	}
	if b, _ := json.Marshal(shown.ApiKey); string(b) != `"ovh/key"` {
		t.Errorf("shown reference encodes to %s", b)
	}
	if !shown.Other.IsSealed() {
		t.Error("a secret of the instance keystore was touched")
	}

	got, err := f.m.ReferencesOf(ctx, sc, &managedObject{ApiKey: sealedFromStorage(t, inRefs.ApiKey.Token())})
	if err != nil || len(got) != 1 || got["apikey"] != "ovh/key" {
		t.Errorf("ReferencesOf = %v, %v; want apikey: ovh/key", got, err)
	}

	// The keystore of another owner shows nothing.
	foreign := &managedObject{ApiKey: sealedFromStorage(t, inRefs.ApiKey.Token())}
	other := sc
	other.Owner = happydns.Identifier{0x02}
	if err := f.m.ShowReferences(ctx, other, foreign); err != nil {
		t.Fatal(err)
	}
	if foreign.ApiKey.IsReference() {
		t.Error("the reference of another owner is shown")
	}
}

func TestReferenceNotSealedAsSecret(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatal(err)
	}
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	current, _ := SafesOf(stored)
	references, err := f.m.ReferencesOf(ctx, sc, stored)
	if err != nil {
		t.Fatal(err)
	}

	// The client sent the reference shown back, as a clear value.
	sent := func() *managedObject { return &managedObject{ApiKey: happydns.NewSecret("ovh/key")} }

	// Staying in the keystore, it is a reference again.
	same := sent()
	if err := f.m.SealObjectInto(ctx, sc, same, Destination{Current: current, References: references}); err != nil {
		t.Fatalf("SealObjectInto(same keystore) = %v", err)
	}
	if safeOf(t, same.ApiKey) != refs.Ref {
		t.Errorf("sent back whole, the object left its keystore for %s", safeOf(t, same.ApiKey))
	}

	// Moving elsewhere, it would become the secret: refused.
	moved := sent()
	err = f.m.SealObjectInto(ctx, sc, moved, Destination{Keystore: KindInstance, Current: current, References: references})
	if !errors.Is(err, ErrReferenceAsSecret) {
		t.Errorf("SealObjectInto(instance) = %v, want ErrReferenceAsSecret", err)
	}

	// A secret typed in its place is fine.
	typed := &managedObject{ApiKey: happydns.NewSecret("s3cr3t")}
	if err := f.m.SealObjectInto(ctx, sc, typed, Destination{Keystore: KindInstance, Current: current, References: references}); err != nil {
		t.Errorf("SealObjectInto(instance, a new secret) = %v", err)
	}
}

func TestReferenceNotStoredInClear(t *testing.T) {
	f := newKeystoreFixture(t, PolicyPlaintext)
	sc := objectContext()
	ctx := context.Background()
	f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	// Its keystore gone, the reference would be stored as the secret.
	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	err := f.m.SealObjectInto(ctx, sc, obj, Destination{References: map[string]string{"apikey": "ovh/key"}})
	if !errors.Is(err, ErrReferenceAsSecret) {
		t.Errorf("SealObjectInto(plaintext) = %v, want ErrReferenceAsSecret", err)
	}
}

func TestInspectDoesNotResolveReferences(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatal(err)
	}

	before := referenceOpens.Load()
	var c Counts
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := f.m.Inspect(ctx, sc, stored, &c); err != nil {
		t.Fatal(err)
	}
	if referenceOpens.Load() != before {
		t.Error("the inspection resolved a reference")
	}
	if c.Sealed[referenceKind] != 1 || c.Unreadable != 0 {
		t.Errorf("Counts = %+v, want 1 sealed of kind %s", c, referenceKind)
	}
}

// A reference that does not read is counted unreadable, and stays withheld.
func TestReferenceUnreadable(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	refs := f.createReference(t, sc.Owner, "My file", map[string]string{"ovh/key": "s3cr3t"})

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := f.m.SealObjectInto(ctx, sc, obj, Destination{Keystore: refs.Ref}); err != nil {
		t.Fatal(err)
	}
	sv, err := ParseSealed(obj.ApiKey.Token())
	if err != nil {
		t.Fatal(err)
	}
	token := FormatSealed(sv.SafeId, []byte("unreadable"))

	var c Counts
	if err := f.m.Inspect(ctx, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}, &c); err != nil {
		t.Fatal(err)
	}
	if c.Unreadable != 1 || c.Sealed[referenceKind] != 0 {
		t.Errorf("Counts = %+v, want 1 unreadable", c)
	}

	shown := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := f.m.ShowReferences(ctx, sc, shown); err != nil {
		t.Fatalf("ShowReferences: %v", err)
	}
	if shown.ApiKey.IsReference() || !shown.ApiKey.IsSealed() {
		t.Error("an unreadable reference is shown")
	}

	if got, err := f.m.ReferencesOf(ctx, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}); err == nil {
		t.Errorf("ReferencesOf(unreadable) = %v, want an error", got)
	}
}

// A secret of a keystore gone is no reference, and fails nothing.
func TestReferenceOfMissingSafe(t *testing.T) {
	f := newKeystoreFixture(t, PolicyInstance)
	sc := objectContext()
	ctx := context.Background()
	token := FormatSealed(happydns.Identifier{0xde, 0xad}, []byte("ovh/key"))

	shown := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := f.m.ShowReferences(ctx, sc, shown); err != nil {
		t.Fatalf("ShowReferences: %v", err)
	}
	if shown.ApiKey.IsReference() {
		t.Error("a secret of an unknown safe is shown as a reference")
	}
	got, err := f.m.ReferencesOf(ctx, sc, &managedObject{ApiKey: sealedFromStorage(t, token)})
	if err != nil || len(got) != 0 {
		t.Errorf("ReferencesOf(unknown safe) = %v, %v; want none", got, err)
	}
}
