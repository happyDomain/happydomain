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
	"testing"

	"git.happydns.org/happyDomain/model"
)

func TestInspectCounts(t *testing.T) {
	m, _, _ := instanceManagers(t)
	sc := objectContext()

	sealed := sealOne(t, m, sc, "v")
	unknownSafe := FormatSealed(happydns.Identifier{0x42}, []byte("payload"))

	objects := []*managedObject{
		{ApiKey: happydns.NewSecret("legacy"), Other: sealedFromStorage(t, sealed)},
		{ApiKey: sealedFromStorage(t, unknownSafe)},
		{},
	}

	var c Counts
	for _, obj := range objects {
		if err := m.Inspect(context.Background(), sc, obj, &c); err != nil {
			t.Fatalf("Inspect: %v", err)
		}
	}

	// The sealed value sits in "other", but was sealed for "apikey": it does
	// not open there, which is what unreadable means.
	if c.Clear != 1 || c.Unreadable != 2 || c.Sealed[KindInstance] != 0 {
		t.Errorf("counts = %+v, want 1 clear, 2 unreadable", c)
	}

	c = Counts{}
	obj := &managedObject{ApiKey: sealedFromStorage(t, sealed)}
	if err := m.Inspect(context.Background(), sc, obj, &c); err != nil {
		t.Fatal(err)
	}
	if c.Sealed[KindInstance] != 1 || c.Clear != 0 || c.Unreadable != 0 {
		t.Errorf("counts = %+v, want 1 sealed in an instance safe", c)
	}
	if !obj.ApiKey.IsSealed() {
		t.Error("Inspect changed the object")
	}
}

func TestResealObject(t *testing.T) {
	instance, plaintext, _ := instanceManagers(t)
	sc := objectContext()

	// Plaintext to instance: legacy values are sealed.
	var obj managedObject
	if err := json.Unmarshal([]byte(`{"apikey":"legacy","other":""}`), &obj); err != nil {
		t.Fatal(err)
	}
	changed, err := instance.ResealObject(context.Background(), sc, &obj)
	if err != nil || !changed {
		t.Fatalf("ResealObject(clear, instance) = %v, %v; want changed", changed, err)
	}
	if !IsSealed(obj.ApiKey.Token()) || obj.ApiKey.Reveal() != "legacy" {
		t.Errorf("not sealed: %q", obj.ApiKey.Token())
	}

	// Already sealed: nothing to do.
	stored, _ := json.Marshal(&obj)
	var again managedObject
	_ = json.Unmarshal(stored, &again)
	if changed, err := instance.ResealObject(context.Background(), sc, &again); err != nil || changed {
		t.Errorf("ResealObject(sealed, instance) = %v, %v; want unchanged", changed, err)
	}

	// Instance to plaintext: sealed values are stored in clear again.
	var back managedObject
	_ = json.Unmarshal(stored, &back)
	changed, err = plaintext.ResealObject(context.Background(), sc, &back)
	if err != nil || !changed {
		t.Fatalf("ResealObject(sealed, plaintext) = %v, %v; want changed", changed, err)
	}
	b, err := json.Marshal(&back)
	if err != nil || string(b) != `{"host":"","apikey":"legacy","other":""}` {
		t.Errorf("stored after reseal to plaintext = %s, %v", b, err)
	}

	// Clear under plaintext: stored the same, nothing to do.
	var legacy managedObject
	_ = json.Unmarshal([]byte(`{"apikey":"legacy"}`), &legacy)
	if changed, err := plaintext.ResealObject(context.Background(), sc, &legacy); err != nil || changed {
		t.Errorf("ResealObject(clear, plaintext) = %v, %v; want unchanged", changed, err)
	}
}

// An object partly written under another policy, one secret sealed and one in
// clear, is completed in both directions, each secret bound to its own field.
func TestResealObjectMixed(t *testing.T) {
	instance, plaintext, _ := instanceManagers(t)
	ctx := context.Background()
	sc := objectContext()
	token := sealOne(t, instance, sc, "sealed")
	mixed := func() *managedObject {
		return &managedObject{ApiKey: sealedFromStorage(t, token), Other: happydns.NewSecret("clear")}
	}
	const inClear = `{"host":"","apikey":"sealed","other":"clear"}`

	obj := mixed()
	if changed, err := plaintext.ResealObject(ctx, sc, obj); err != nil || !changed {
		t.Fatalf("ResealObject(mixed, plaintext) = %v, %v; want changed", changed, err)
	}
	if b, err := json.Marshal(obj); err != nil || string(b) != inClear {
		t.Errorf("stored under plaintext = %s, %v; want %s", b, err, inClear)
	}

	obj = mixed()
	if changed, err := instance.ResealObject(ctx, sc, obj); err != nil || !changed {
		t.Fatalf("ResealObject(mixed, instance) = %v, %v; want changed", changed, err)
	}
	if obj.ApiKey.Token() != token || !IsSealed(obj.Other.Token()) {
		t.Fatalf("apikey %q, other %q; want the first kept, the second sealed", obj.ApiKey.Token(), obj.Other.Token())
	}

	stored, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	var back managedObject
	if err := json.Unmarshal(stored, &back); err != nil {
		t.Fatal(err)
	}
	if _, err := plaintext.ResealObject(ctx, sc, &back); err != nil {
		t.Fatalf("the newly sealed field does not open in its place: %v", err)
	}
	if b, _ := json.Marshal(&back); string(b) != inClear {
		t.Errorf("round trip = %s, want %s", b, inClear)
	}
}

func TestResealObjectFailsOnUnreadable(t *testing.T) {
	_, plaintext, _ := instanceManagers(t)
	obj := &managedObject{ApiKey: sealedFromStorage(t, FormatSealed(happydns.Identifier{0x42}, []byte("payload")))}

	if _, err := plaintext.ResealObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("an unreadable secret was resealed")
	}
	if !obj.ApiKey.IsSealed() {
		t.Error("a failed reseal changed the object")
	}
}
