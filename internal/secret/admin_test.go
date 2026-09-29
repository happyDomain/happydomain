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
	"slices"
	"testing"

	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
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

// A value that will never open is lost already: a reseal leaves it as it is,
// and stores the other secrets of its object the way the policy does, so that
// one lost value does not keep the rest sealed, or in clear, for ever.
func TestResealObjectGoesPastAValueThatNoLongerOpens(t *testing.T) {
	ctx := context.Background()
	instance, plaintext, _ := instanceManagers(t)
	sc := objectContext()
	lost := FormatSealed(happydns.Identifier{0x42}, []byte("payload"))
	token := sealOne(t, instance, sc, "v")

	// Back to clear: the value that opens goes, the lost one stays.
	obj := &managedObject{ApiKey: sealedFromStorage(t, token), Other: sealedFromStorage(t, lost)}
	changed, err := plaintext.ResealObject(ctx, sc, obj)
	if err != nil || !changed {
		t.Fatalf("ResealObject(plaintext) = %v, %v; want changed", changed, err)
	}
	if b, err := json.Marshal(obj); err != nil || string(b) != `{"host":"","apikey":"v","other":"`+lost+`"}` {
		t.Errorf("stored = %s, %v; want the key in clear, the lost value as is", b, err)
	}

	// To the instance policy: the clear value is sealed, the lost one stays.
	obj = &managedObject{ApiKey: happydns.NewSecret("legacy"), Other: sealedFromStorage(t, lost)}
	changed, err = instance.ResealObject(ctx, sc, obj)
	if err != nil || !changed {
		t.Fatalf("ResealObject(instance) = %v, %v; want changed", changed, err)
	}
	if !IsSealed(obj.ApiKey.Token()) || obj.ApiKey.Reveal() != "legacy" || obj.Other.Token() != lost {
		t.Errorf("after reseal: apikey %q, other %q; want the key sealed, the lost value as is", obj.ApiKey.Token(), obj.Other.Token())
	}

	// Nothing but a lost value: nothing to write.
	for name, m := range map[string]*Manager{"instance": instance, "plaintext": plaintext} {
		obj := &managedObject{ApiKey: sealedFromStorage(t, lost)}
		if changed, err := m.ResealObject(ctx, sc, obj); err != nil || changed {
			t.Errorf("ResealObject(lost only, %s) = %v, %v; want unchanged", name, changed, err)
		}
		if obj.ApiKey.Token() != lost {
			t.Errorf("ResealObject(lost only, %s) changed the object", name)
		}
	}
}

// A value whose safe cannot be read for now is not taken for lost: the
// reseal of its object fails, and leaves it as it was.
func TestResealObjectFailsOnAPassingFailure(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := secrettest.NewSafes()
	instance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	flaky, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: flakySafes{store}})
	if err != nil {
		t.Fatal(err)
	}

	sc := objectContext()
	token := sealOne(t, instance, sc, "v")
	obj := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if _, err := flaky.ResealObject(ctx, sc, obj); err == nil {
		t.Error("ResealObject succeeded with the safe unreadable for now")
	}
	if obj.ApiKey.Token() != token {
		t.Error("a failed reseal changed the object")
	}

	vsc := objectContext()
	vsc.Field = "k"
	vtoken, err := instance.SealValue(ctx, vsc, "v")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := flaky.ResealValue(ctx, vsc, vtoken); err == nil {
		t.Error("ResealValue succeeded with the safe unreadable for now")
	}
}

// A single value that will never open is left as it is by a reseal, under
// either policy.
func TestResealValueLeavesAValueThatNoLongerOpens(t *testing.T) {
	ctx := context.Background()
	instance, plaintext, _ := instanceManagers(t)
	sc := objectContext()
	sc.Field = "k"
	lost := FormatSealed(happydns.Identifier{0x42}, []byte("payload"))

	for name, m := range map[string]*Manager{"instance": instance, "plaintext": plaintext} {
		if out, changed, err := m.ResealValue(ctx, sc, lost); err != nil || changed || out != lost {
			t.Errorf("ResealValue(lost, %s) = %q, %v, %v; want it left as is", name, out, changed, err)
		}
	}
}

func TestRewrapMovesSafesToPrimary(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := secrettest.NewSafes()

	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})
	if err != nil {
		t.Fatal(err)
	}

	owners := []happydns.Identifier{{0x01}, {0x02}, {0x03}}
	tokens := map[string]string{}
	for _, o := range owners {
		sc := objectContext()
		sc.Owner = o
		tokens[o.String()] = sealOne(t, m, sc, "value of "+o.String())
	}

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})
	if err != nil {
		t.Fatal(err)
	}

	usage, err := rm.KeyUsage()
	if err != nil {
		t.Fatal(err)
	}
	if usage[oldKey.PrimaryKeyId()] != 3 || usage[newKey.PrimaryKeyId()] != 0 {
		t.Errorf("usage before rewrap = %v, want 3 safes on the old key", usage)
	}

	report, err := rm.Rewrap(context.Background())
	if err != nil || report.Changed != 3 || report.Failed != 0 {
		t.Fatalf("Rewrap = %+v, %v; want 3 rewrapped", report, err)
	}

	usage, _ = rm.KeyUsage()
	if usage[oldKey.PrimaryKeyId()] != 0 || usage[newKey.PrimaryKeyId()] != 3 {
		t.Errorf("usage after rewrap = %v, want 3 safes on the new primary", usage)
	}

	// Idempotent.
	if report, err := rm.Rewrap(context.Background()); err != nil || report.Changed != 0 {
		t.Errorf("second Rewrap = %+v, %v; want nothing to do", report, err)
	}

	// The data stays readable, even with the old key gone.
	onlyNew, _ := NewInstanceKey(onlyPrimary(t, rotated))
	nm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: onlyNew, Safes: store})
	for _, o := range owners {
		sc := objectContext()
		sc.Owner = o
		obj := &managedObject{ApiKey: sealedFromStorage(t, tokens[o.String()])}
		if err := nm.OpenObject(context.Background(), sc, obj); err != nil || obj.ApiKey.Reveal() != "value of "+o.String() {
			t.Errorf("after rewrap, %s does not open without the old key: %v", o.String(), err)
		}
	}
}

// A safe that does not open (its key removed from the keyset too early, a
// damaged blob) is reported, and never keeps the others from being
// rewrapped, now or on a rerun.
func TestRewrapReportsAndContinuesPastASafeThatDoesNotOpen(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := secrettest.NewSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})

	tokens := map[byte]string{}
	for _, o := range []byte{0x01, 0x03} {
		sc := objectContext()
		sc.Owner = happydns.Identifier{o}
		tokens[o] = sealOne(t, m, sc, "value")
	}

	// A safe wrapped by a key the instance keyset does not hold.
	otherH, _ := GenerateInstanceKeyset()
	otherKey, _ := NewInstanceKey(otherH)
	om, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: otherKey, Safes: store})
	lostSc := objectContext()
	lostSc.Owner = happydns.Identifier{0x02}
	_ = sealOne(t, om, lostSc, "lost")
	lost, err := store.GetSafeByOwner(lostSc.Owner, KindInstance)
	if err != nil {
		t.Fatal(err)
	}

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})

	for run := range 2 {
		report, err := rm.Rewrap(context.Background())
		if err != nil {
			t.Fatalf("Rewrap (run %d): %v", run, err)
		}
		wantChanged := 2
		if run > 0 {
			wantChanged = 0
		}
		if report.Changed != wantChanged || report.Failed != 1 || report.Skipped != 0 {
			t.Errorf("Rewrap (run %d) = %+v, want %d rewrapped and 1 failure", run, report, wantChanged)
		}
		containsAll(t, "errors", report.Errors, lost.Id.String())
	}

	usage, _ := rm.KeyUsage()
	if usage[oldKey.PrimaryKeyId()] != 0 || usage[newKey.PrimaryKeyId()] != 2 {
		t.Errorf("usage = %v, want both readable safes on the new primary", usage)
	}
}

// A safe carrying several entries wrapped by old keys is rewrapped, written
// and counted once.
func TestRewrapCountsASafeOnce(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := secrettest.NewSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})
	sc := objectContext()
	token := sealOne(t, m, sc, "value")

	safe, _ := store.GetSafeByOwner(sc.Owner, KindInstance)
	safe.Keyring = append(safe.Keyring, safe.Keyring[0])
	store.Put(*safe)

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})

	report, err := rm.Rewrap(context.Background())
	if err != nil || report.Processed != 1 || report.Changed != 1 {
		t.Fatalf("Rewrap = %+v, %v; want one safe rewrapped once", report, err)
	}
	if store.Replaces() != 1 {
		t.Errorf("safe written %d times, want once", store.Replaces())
	}
	if usage, _ := rm.KeyUsage(); usage[oldKey.PrimaryKeyId()] != 0 {
		t.Errorf("usage = %v, want no entry left on the old key", usage)
	}

	obj := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := rm.OpenObject(context.Background(), sc, obj); err != nil || obj.ApiKey.Reveal() != "value" {
		t.Errorf("after rewrap, the value does not open: %v", err)
	}
}

// conflictingSafes reports every safe of owner as changed meanwhile.
type conflictingSafes struct {
	*secrettest.Safes
	owner happydns.Identifier
}

func (c *conflictingSafes) ReplaceSafe(id happydns.Identifier, update func(*happydns.Safe) (*happydns.Safe, error)) error {
	if s, err := c.GetSafe(id); err == nil && s.Owner.Equals(c.owner) {
		return happydns.ErrChangedMeanwhile
	}
	return c.Safes.ReplaceSafe(id, update)
}

func TestRewrapSkipsASafeChangedMeanwhile(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := secrettest.NewSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})
	for _, o := range []byte{0x01, 0x02} {
		sc := objectContext()
		sc.Owner = happydns.Identifier{o}
		_ = sealOne(t, m, sc, "value")
	}

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: &conflictingSafes{Safes: store, owner: happydns.Identifier{0x02}}})

	report, err := rm.Rewrap(context.Background())
	if err != nil || report.Changed != 1 || report.Skipped != 1 || report.Failed != 0 {
		t.Errorf("Rewrap = %+v, %v; want one rewrapped, one skipped", report, err)
	}
}

// A placeholder stored by mistake is a secret lost: counted as unreadable,
// not a reason to fail the whole report.
func TestInspectCountsAStoredPlaceholderAsUnreadable(t *testing.T) {
	m, _, _ := instanceManagers(t)

	var c Counts
	if err := m.Inspect(context.Background(), objectContext(), &managedObject{ApiKey: redacted(), Other: happydns.NewSecret("legacy")}, &c); err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if c.Unreadable != 1 || c.Clear != 1 {
		t.Errorf("counts = %+v, want 1 unreadable, 1 clear", c)
	}
}

// onlyPrimary returns h without its non-primary keys, as after an operator
// removed them.
func onlyPrimary(t *testing.T, h *keyset.Handle) *keyset.Handle {
	t.Helper()
	km := keyset.NewManagerFromHandle(h)
	primary := h.KeysetInfo().GetPrimaryKeyId()
	for _, k := range h.KeysetInfo().GetKeyInfo() {
		if k.GetKeyId() != primary {
			if err := km.Delete(k.GetKeyId()); err != nil {
				t.Fatal(err)
			}
		}
	}
	out, err := km.Handle()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestKeyStatus(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := secrettest.NewSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})
	_ = sealOne(t, m, objectContext(), "v")

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})

	keys, err := rm.KeyStatus()
	if err != nil {
		t.Fatal(err)
	}
	byId := map[uint32]KeyStatus{}
	for _, k := range keys {
		byId[k.Id] = k
	}
	if k := byId[oldKey.PrimaryKeyId()]; k.Safes != 1 || k.Primary {
		t.Errorf("old key = %+v, want 1 safe, not primary", k)
	}
	if k := byId[newKey.PrimaryKeyId()]; k.Safes != 0 || !k.Primary {
		t.Errorf("new key = %+v, want the primary, no safe yet", k)
	}

	// A safe wrapped by a key removed from the keyset is reported.
	onlyNew, _ := NewInstanceKey(onlyPrimary(t, rotated))
	om, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: onlyNew, Safes: store})
	keys, _ = om.KeyStatus()
	missing := false
	for _, k := range keys {
		if k.Id == oldKey.PrimaryKeyId() && k.Status == "MISSING" && k.Safes == 1 {
			missing = true
		}
	}
	if !missing {
		t.Errorf("keys = %+v, want the removed key reported missing with its safe", keys)
	}
}

// A damaged safe does not keep the administrator from learning which key
// wraps the others; keys missing from the keyset come in a stable order.
func TestKeyStatusSurvivesADamagedSafe(t *testing.T) {
	store := secrettest.NewSafes()
	var missing []uint32
	for _, o := range []byte{0x01, 0x02} {
		h, _ := GenerateInstanceKeyset()
		k, _ := NewInstanceKey(h)
		m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: k, Safes: store})
		sc := objectContext()
		sc.Owner = happydns.Identifier{o}
		_ = sealOne(t, m, sc, "v")
		missing = append(missing, k.PrimaryKeyId())
	}
	slices.Sort(missing)

	damaged := happydns.Safe{Id: happydns.Identifier{0x42}, Owner: happydns.Identifier{0x03}, Kind: KindInstance,
		Keyring: []happydns.WrappedKeyset{{KEK: KEKInstance, Blob: []byte("garbage")}}}
	store.Put(damaged)

	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store})
	for range 5 {
		keys, err := m.KeyStatus()
		if err != nil {
			t.Fatalf("KeyStatus: %v", err)
		}

		var gotMissing []uint32
		unreadable := 0
		for _, k := range keys {
			switch k.Status {
			case "MISSING":
				gotMissing = append(gotMissing, k.Id)
			case "UNREADABLE":
				unreadable += k.Safes
			}
		}
		if !slices.Equal(gotMissing, missing) {
			t.Fatalf("missing keys = %v, want %v in this order", gotMissing, missing)
		}
		if unreadable != 1 {
			t.Errorf("keys = %+v, want the damaged safe reported as unreadable", keys)
		}
	}
}

func TestValueHelpers(t *testing.T) {
	instance, plaintext, _ := instanceManagers(t)
	ctx := context.Background()
	sc := objectContext()
	sc.Field = "k"

	token, err := instance.SealValue(ctx, sc, "v")
	if err != nil || !IsSealed(token) {
		t.Fatalf("SealValue = %q, %v", token, err)
	}
	if same, err := instance.SealValue(ctx, sc, token); err != nil || same != token {
		t.Errorf("SealValue(sealed) = %q, %v; want it unchanged", same, err)
	}

	if v, err := instance.OpenValue(ctx, sc, token); err != nil || v != "v" {
		t.Errorf("OpenValue = %q, %v", v, err)
	}
	if v, err := instance.OpenValue(ctx, sc, "legacy"); err != nil || v != "legacy" {
		t.Errorf("OpenValue(clear) = %q, %v", v, err)
	}

	var c Counts
	for _, v := range []string{token, "legacy", FormatSealed(happydns.Identifier{0x42}, []byte("x")), ""} {
		if err := instance.InspectValue(ctx, sc, v, &c); err != nil {
			t.Fatal(err)
		}
	}
	if c.Clear != 1 || c.Sealed[KindInstance] != 1 || c.Unreadable != 1 {
		t.Errorf("counts = %+v", c)
	}

	if out, changed, err := instance.ResealValue(ctx, sc, "legacy"); err != nil || !changed || !IsSealed(out) {
		t.Errorf("ResealValue(clear, instance) = %q, %v, %v", out, changed, err)
	}
	if out, changed, err := instance.ResealValue(ctx, sc, token); err != nil || changed || out != token {
		t.Errorf("ResealValue(sealed, instance) = %q, %v, %v", out, changed, err)
	}
	if out, changed, err := plaintext.ResealValue(ctx, sc, token); err != nil || !changed || out != "v" {
		t.Errorf("ResealValue(sealed, plaintext) = %q, %v, %v", out, changed, err)
	}
	if out, changed, err := plaintext.ResealValue(ctx, sc, "legacy"); err != nil || changed || out != "legacy" {
		t.Errorf("ResealValue(clear, plaintext) = %q, %v, %v", out, changed, err)
	}
}

func TestDropInstanceSafes(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := newMemStartupStorage()

	instance, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	plaintext, _ := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: store})
	noKey, _ := NewManager(Config{Policy: PolicyPlaintext, Safes: store})

	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []happydns.Identifier{{0x01}, InstanceOwner()} {
		sc := objectContext()
		sc.Owner = owner
		sealOne(t, instance, sc, "v")
	}

	if _, err := instance.DropInstanceSafes(ctx, store); err == nil {
		t.Error("safes dropped under the instance policy")
	}
	if _, err := noKey.DropInstanceSafes(ctx, store); err == nil {
		t.Error("safes dropped without the keyset")
	}
	if store.Creates() != 2 || store.record == nil {
		t.Fatal("a refused drop changed something")
	}

	report, err := plaintext.DropInstanceSafes(ctx, store)
	if err != nil || report.Changed != 2 || report.Failed != 0 {
		t.Fatalf("DropInstanceSafes = %+v, %v; want 2 dropped", report, err)
	}
	if store.record != nil {
		t.Error("the check record is still there")
	}
	if err := StartupCheck(PolicyPlaintext, nil, store); err != nil {
		t.Errorf("StartupCheck without keyset after the drop: %v", err)
	}

	// Idempotent.
	if report, err := plaintext.DropInstanceSafes(ctx, store); err != nil || report.Changed != 0 || report.Failed != 0 {
		t.Errorf("second DropInstanceSafes = %+v, %v", report, err)
	}

	// Another keyset can be configured afterwards.
	okey := testInstanceKey(t)
	if err := StartupCheck(PolicyInstance, okey, store); err != nil {
		t.Errorf("StartupCheck with a new keyset: %v", err)
	}
}

// The placeholder the API sends in place of a secret is never a credential.
// Stored by mistake, it is reported as unreadable, and neither sealed, nor
// resealed, nor handed out as the value it stands for.
func TestValueHelpersRefuseAStoredPlaceholder(t *testing.T) {
	instance, plaintext, _ := instanceManagers(t)
	ctx := context.Background()
	sc := objectContext()
	sc.Field = "k"

	if _, err := instance.SealValue(ctx, sc, happydns.RedactedSecret); !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("SealValue(placeholder) = %v, want ErrRedactedSecret", err)
	}
	if _, err := plaintext.SealValue(ctx, sc, happydns.RedactedSecret); !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("SealValue(placeholder, plaintext) = %v, want ErrRedactedSecret", err)
	}
	if _, err := instance.OpenValue(ctx, sc, happydns.RedactedSecret); !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("OpenValue(placeholder) = %v, want ErrRedactedSecret", err)
	}
	for name, m := range map[string]*Manager{"instance": instance, "plaintext": plaintext} {
		if _, changed, err := m.ResealValue(ctx, sc, happydns.RedactedSecret); !errors.Is(err, ErrRedactedSecret) || changed {
			t.Errorf("ResealValue(placeholder, %s) = %v, %v; want ErrRedactedSecret, unchanged", name, changed, err)
		}
	}

	var c Counts
	if err := instance.InspectValue(ctx, sc, happydns.RedactedSecret, &c); err != nil {
		t.Fatal(err)
	}
	if c.Unreadable != 1 {
		t.Errorf("counts = %+v, want the placeholder unreadable", c)
	}
}

// countingSafes counts the safes read by identifier.
type countingSafes struct {
	*secrettest.Safes
	gets int
}

func (s *countingSafes) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	s.gets++
	return s.Safes.GetSafe(id)
}

// Opening several values in a row reads each safe once.
func TestValueOpenerLooksEachSafeUpOnce(t *testing.T) {
	store := &countingSafes{Safes: secrettest.NewSafes()}
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	tokens := map[string]string{}
	for _, field := range []string{"a", "b", "c"} {
		sc := objectContext()
		sc.Field = field
		if tokens[field], err = m.SealValue(ctx, sc, "value-"+field); err != nil {
			t.Fatal(err)
		}
	}

	store.gets = 0
	opener := m.NewValueOpener()
	for field, token := range tokens {
		sc := objectContext()
		sc.Field = field
		if v, err := opener.Open(ctx, sc, token); err != nil || v != "value-"+field {
			t.Errorf("Open(%s) = %q, %v", field, v, err)
		}
	}
	if v, err := opener.Open(ctx, objectContext(), "legacy"); err != nil || v != "legacy" {
		t.Errorf("Open(clear) = %q, %v", v, err)
	}
	if _, err := opener.Open(ctx, objectContext(), happydns.RedactedSecret); !errors.Is(err, ErrRedactedSecret) {
		t.Errorf("Open(placeholder) = %v, want ErrRedactedSecret", err)
	}
	if store.gets != 1 {
		t.Errorf("the safe was read %d times, want once", store.gets)
	}

	if _, err := (*Manager)(nil).NewValueOpener().Open(ctx, objectContext(), tokens["a"]); err == nil {
		t.Error("Open without a manager succeeded")
	}
}

// flakySafes fails to read a safe by identifier, like a storage briefly down.
type flakySafes struct {
	*secrettest.Safes
}

func (flakySafes) GetSafe(happydns.Identifier) (*happydns.Safe, error) {
	return nil, errors.New("storage unavailable")
}

// A safe that cannot be read for now does not make the values sealed in it
// count as unreadable: they may open a moment later. Inspecting fails
// instead, so that the object is reported as not looked at.
func TestInspectDoesNotCountAPassingFailureAsUnreadable(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := secrettest.NewSafes()

	instance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	flaky, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: flakySafes{store}})
	if err != nil {
		t.Fatal(err)
	}

	sc := objectContext()
	token := sealOne(t, instance, sc, "v")

	var c Counts
	err = flaky.Inspect(ctx, sc, &managedObject{ApiKey: sealedFromStorage(t, token)}, &c)
	if err == nil || errors.Is(err, ErrUnopenable) {
		t.Errorf("Inspect = %v, want an error other than ErrUnopenable", err)
	}
	if c.Unreadable != 0 {
		t.Errorf("Inspect counted %d unreadable, want none", c.Unreadable)
	}

	vsc := objectContext()
	vsc.Field = "k"
	vtoken, err := instance.SealValue(ctx, vsc, "v")
	if err != nil {
		t.Fatal(err)
	}
	c = Counts{}
	if err := flaky.InspectValue(ctx, vsc, vtoken, &c); err == nil || c.Unreadable != 0 {
		t.Errorf("InspectValue = %v, counts %+v; want an error and nothing unreadable", err, c)
	}
}
