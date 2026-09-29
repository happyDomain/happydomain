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
	instance, _, _ := instanceManagers(t)
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
}

// The placeholder the API sends in place of a secret is never a credential.
// Stored by mistake, it is neither sealed nor handed out as the value it
// stands for.
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
