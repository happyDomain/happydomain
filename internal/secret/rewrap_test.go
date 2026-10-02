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
	"errors"
	"strings"
	"testing"

	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

// withoutKey returns h without the key id, as after an operator removed it.
func withoutKey(t *testing.T, h *keyset.Handle, id uint32) *keyset.Handle {
	t.Helper()
	km := keyset.NewManagerFromHandle(h)
	if err := km.Disable(id); err != nil {
		t.Fatal(err)
	}
	if err := km.Delete(id); err != nil {
		t.Fatal(err)
	}
	out, err := km.Handle()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRewrapMovesSafesToPrimary(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := newMemSafeStorage()

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
	if err != nil || report.Processed != 3 || report.Changed != 3 || report.Failed != 0 {
		t.Fatalf("Rewrap = %+v, %v; want 3 rewrapped", report, err)
	}

	usage, _ = rm.KeyUsage()
	if usage[oldKey.PrimaryKeyId()] != 0 || usage[newKey.PrimaryKeyId()] != 3 {
		t.Errorf("usage after rewrap = %v, want 3 safes on the new primary", usage)
	}

	// Idempotent.
	if report, err := rm.Rewrap(context.Background()); err != nil || report.Changed != 0 || report.Failed != 0 {
		t.Errorf("second Rewrap = %+v, %v; want nothing to do", report, err)
	}

	// The data stays readable, even with the old key gone.
	onlyNew, _ := NewInstanceKey(withoutKey(t, rotated, oldKey.PrimaryKeyId()))
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

// A safe that cannot be rewrapped, its key already removed from the keyset,
// is reported and left as it was: the others are rewrapped all the same.
func TestRewrapGoesPastAFailingSafe(t *testing.T) {
	h1, _ := GenerateInstanceKeyset()
	key1, _ := NewInstanceKey(h1)
	store := newMemSafeStorage()
	m1, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key1, Safes: store})
	lost := objectContext()
	lost.Owner = happydns.Identifier{0x01}
	sealOne(t, m1, lost, "v")

	h2, _ := RotateInstanceKeyset(h1)
	key2, _ := NewInstanceKey(h2)
	m2, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key2, Safes: store})
	kept := objectContext()
	kept.Owner = happydns.Identifier{0x02}
	token := sealOne(t, m2, kept, "kept")

	h3, _ := RotateInstanceKeyset(h2)
	key3, _ := NewInstanceKey(withoutKey(t, h3, key1.PrimaryKeyId()))
	m3, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key3, Safes: store})

	report, err := m3.Rewrap(context.Background())
	if err != nil {
		t.Fatalf("Rewrap: %v", err)
	}
	if report.Processed != 2 || report.Changed != 1 || report.Failed != 1 || len(report.Errors) != 1 {
		t.Errorf("Rewrap = %+v, want one rewrapped, one failed", report)
	}

	only3, _ := NewInstanceKey(withoutKey(t, withoutKey(t, h3, key1.PrimaryKeyId()), key2.PrimaryKeyId()))
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: only3, Safes: store})
	obj := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), kept, obj); err != nil || obj.ApiKey.Reveal() != "kept" {
		t.Errorf("the rewrapped safe does not open under the primary alone: %v", err)
	}
}

func TestKeyStatus(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := newMemSafeStorage()
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
	onlyNew, _ := NewInstanceKey(withoutKey(t, rotated, oldKey.PrimaryKeyId()))
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

// A safe whose wrapping key cannot be told is counted apart, rather than
// failing the whole report or hiding the key it may need.
func TestKeyStatusCountsUnreadableSafes(t *testing.T) {
	m, _, store := instanceManagers(t)
	sealOne(t, m, objectContext(), "v")
	store.safes["broken"] = happydns.Safe{
		Id:      happydns.Identifier("broken"),
		Owner:   happydns.Identifier{0x09},
		Kind:    KindInstance,
		Keyring: []happydns.WrappedKeyset{{KEK: KEKInstance, Blob: []byte("not a wrapped keyset")}},
	}

	keys, err := m.KeyStatus()
	if err != nil {
		t.Fatalf("KeyStatus: %v", err)
	}
	unreadable := 0
	for _, k := range keys {
		if k.Status == "UNREADABLE" {
			unreadable = k.Safes
		}
	}
	if unreadable != 1 {
		t.Errorf("keys = %+v, want one UNREADABLE safe", keys)
	}
}

// Without an instance keyset, the default deployment, the key report still
// goes through: safes left wrapped are listed MISSING. Rewrap refuses.
func TestKeyStatusAndRewrapWithoutKeyset(t *testing.T) {
	m, err := NewManager(Config{Policy: PolicyPlaintext, Safes: newMemSafeStorage()})
	if err != nil {
		t.Fatal(err)
	}
	if keys, err := m.KeyStatus(); err != nil || len(keys) != 0 {
		t.Errorf("KeyStatus = %+v, %v; want no key and no error", keys, err)
	}
	if _, err := m.Rewrap(context.Background()); err == nil {
		t.Error("Rewrap without an instance keyset succeeded")
	}

	instance, _, store := instanceManagers(t)
	sealOne(t, instance, objectContext(), "v")
	nokey, err := NewManager(Config{Policy: PolicyPlaintext, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	keys, err := nokey.KeyStatus()
	if err != nil || len(keys) != 1 || keys[0].Status != "MISSING" || keys[0].Safes != 1 {
		t.Errorf("KeyStatus = %+v, %v; want the wrapping key listed MISSING with one safe", keys, err)
	}
}

// A safe holding several instance entries keeps only the new one: a stale
// wrapping would leave the old key able to open it.
func TestRewrapCollapsesInstanceEntries(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldKey, _ := NewInstanceKey(h)
	store := newMemSafeStorage()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: oldKey, Safes: store})
	sc := objectContext()
	token := sealOne(t, m, sc, "v")

	safe, err := store.GetSafeByOwner(sc.Owner, KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	safe.Keyring = append(safe.Keyring, safe.Keyring[0])
	if err := store.UpdateSafe(safe); err != nil {
		t.Fatal(err)
	}

	rotated, _ := RotateInstanceKeyset(h)
	newKey, _ := NewInstanceKey(rotated)
	rm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})
	if report, err := rm.Rewrap(context.Background()); err != nil || report.Changed != 1 || report.Failed != 0 {
		t.Fatalf("Rewrap = %+v, %v; want the safe rewrapped", report, err)
	}

	safe, _ = store.GetSafeByOwner(sc.Owner, KindInstance)
	n := 0
	for _, w := range safe.Keyring {
		if w.KEK == KEKInstance {
			n++
		}
	}
	if n != 1 {
		t.Errorf("keyring holds %d instance entries after rewrap, want 1", n)
	}
	if usage, _ := rm.KeyUsage(); usage[oldKey.PrimaryKeyId()] != 0 {
		t.Errorf("usage = %v, the old key still wraps the safe", usage)
	}

	onlyNew, _ := NewInstanceKey(withoutKey(t, rotated, oldKey.PrimaryKeyId()))
	nm, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: onlyNew, Safes: store})
	obj := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := nm.OpenObject(context.Background(), sc, obj); err != nil || obj.ApiKey.Reveal() != "v" {
		t.Errorf("the rewrapped safe does not open without the old key: %v", err)
	}
}

// A safe record that does not decode is reported as failed by Rewrap, and
// counted UNREADABLE by the key report: the other safes go through.
func TestRewrapAndKeyStatusReportDamagedRecords(t *testing.T) {
	m, _, store := instanceManagers(t)
	sealOne(t, m, objectContext(), "v")
	store.damaged["safe-broken"] = errors.New("does not decode")

	report, err := m.Rewrap(context.Background())
	if err != nil {
		t.Fatalf("Rewrap: %v", err)
	}
	if report.Processed != 2 || report.Failed != 1 || len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "safe-broken") {
		t.Errorf("Rewrap = %+v, want the damaged record reported failed by its key", report)
	}

	keys, err := m.KeyStatus()
	if err != nil {
		t.Fatalf("KeyStatus: %v", err)
	}
	primary, unreadable := 0, 0
	for _, k := range keys {
		switch {
		case k.Primary:
			primary = k.Safes
		case k.Status == "UNREADABLE":
			unreadable = k.Safes
		}
	}
	if primary != 1 || unreadable != 1 {
		t.Errorf("keys = %+v, want the safe on the primary and the damaged record UNREADABLE", keys)
	}
}
