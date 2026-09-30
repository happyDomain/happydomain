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
	"testing"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

func safeOf(t *testing.T, token string) string {
	t.Helper()
	sv, err := ParseSealed(token)
	if err != nil {
		t.Fatal(err)
	}
	return sv.SafeId.String()
}

// Every sealed value is counted by the safe it names, whether it opens or
// not: that is how the status tells what depends on a damaged safe.
func TestInspectCountsSealedValuesBySafe(t *testing.T) {
	ctx := context.Background()
	m, _, _ := instanceManagers(t)

	a := objectContext()
	both := &managedObject{ApiKey: happydns.NewSecret("v"), Other: happydns.NewSecret("w")}
	if err := m.SealObject(ctx, a, both); err != nil {
		t.Fatal(err)
	}
	b := objectContext()
	b.Owner = happydns.Identifier{0x02}
	tokenB := sealOne(t, m, b, "v")
	unknown := FormatSealed(happydns.Identifier{0x42}, []byte("payload"))

	var c Counts
	for _, obj := range []struct {
		sc  SecretContext
		obj *managedObject
	}{
		{a, &managedObject{ApiKey: sealedFromStorage(t, both.ApiKey.Token()), Other: sealedFromStorage(t, both.Other.Token())}},
		{b, &managedObject{ApiKey: sealedFromStorage(t, tokenB), Other: sealedFromStorage(t, unknown)}},
		{a, &managedObject{ApiKey: happydns.NewSecret("clear")}},
	} {
		if err := m.Inspect(ctx, obj.sc, obj.obj, &c); err != nil {
			t.Fatalf("Inspect: %v", err)
		}
	}

	vsc := objectContext()
	vsc.Field = "k"
	vtoken, err := m.SealValue(ctx, vsc, "v")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.InspectValue(ctx, vsc, vtoken, &c); err != nil {
		t.Fatal(err)
	}

	want := map[string]int{
		safeOf(t, both.ApiKey.Token()): 3,
		safeOf(t, tokenB):              1,
		safeOf(t, unknown):             1,
	}
	if len(c.SealedIn) != len(want) {
		t.Errorf("SealedIn = %v, want %v", c.SealedIn, want)
	}
	for id, n := range want {
		if c.SealedIn[id] != n {
			t.Errorf("SealedIn[%s] = %d, want %d", id, c.SealedIn[id], n)
		}
	}
}

// An object whose safe cannot be read adds nothing but its failure to the
// counts, except the values it seals, every one of them: they are what a
// damaged safe takes with it.
func TestInspectAllCountsTheSealedValuesOfAnObjectFailing(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := secrettest.NewSafes()
	instance, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	flaky, _ := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: flakySafes{store}})

	sc := objectContext()
	obj := &managedObject{ApiKey: happydns.NewSecret("v"), Other: happydns.NewSecret("w")}
	if err := instance.SealObject(ctx, sc, obj); err != nil {
		t.Fatal(err)
	}
	stored := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token()), Other: sealedFromStorage(t, obj.Other.Token())}

	c, err := InspectAll(newEntriesIterator(iterEntry[managedObject]{key: "o", item: stored}),
		func(*managedObject) string { return "o" },
		func(o *managedObject, c *Counts) error { return flaky.Inspect(ctx, sc, o, c) })
	if err != nil {
		t.Fatal(err)
	}
	if c.Undecodable != 1 || c.Unreadable != 0 || c.Clear != 0 {
		t.Errorf("counts = %+v, want the object undecodable only", c)
	}
	if n := c.SealedIn[safeOf(t, obj.ApiKey.Token())]; n != 2 {
		t.Errorf("SealedIn = %v, want both values of the object", c.SealedIn)
	}
}
