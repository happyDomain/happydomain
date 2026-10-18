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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// escrowedSafe seals value in the instance safe of the owner of sc, then
// keeps a copy of its key under the escrow store. It returns the token and
// the instance keyset.
func escrowedSafe(t *testing.T, store *memSafeStorage, sc SecretContext, value string) (string, *InstanceKey) {
	t.Helper()
	key := testInstanceKey(t)
	m := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	token := sealOne(t, m, sc, value)

	safe, err := store.GetSafeByOwner(sc.Owner, KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	safe.Keyring = append(safe.Keyring, escrowEntry(t, mustUnwrap(t, key, safe)))
	if err := store.UpdateSafe(safe); err != nil {
		t.Fatal(err)
	}
	return token, key
}

func openToken(m *Manager, sc SecretContext, token string) (string, error) {
	var s happydns.Secret
	s.SetSealed(token)
	obj := &managedObject{ApiKey: s}
	err := m.OpenObject(context.Background(), sc, obj)
	return obj.ApiKey.Reveal(), err
}

// The store of its kind not configured here, a safe opens through the copy
// of its key kept by another store.
func TestOpenThroughACopyWhenTheStoreIsMissing(t *testing.T) {
	store := newMemSafeStorage()
	sc := objectContext()
	token, _ := escrowedSafe(t, store, sc, "kept")

	m := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store})
	if got, err := openToken(m, sc, token); err != nil || got != "kept" {
		t.Errorf("open without the instance keyset = %q, %v; want it through the escrow copy", got, err)
	}

	alone := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store, Stores: []string{string(PolicyPlaintext)}})
	if _, err := openToken(alone, sc, token); err == nil || !strings.Contains(err.Error(), KindInstance) {
		t.Errorf("open with no store = %v, want an error naming the store of the safe", err)
	}
}

// The store of its kind failing, as with another keyset than the one that
// wrapped it, a safe still opens through a copy.
func TestOpenThroughACopyWhenTheStoreFails(t *testing.T) {
	store := newMemSafeStorage()
	sc := objectContext()
	token, _ := escrowedSafe(t, store, sc, "kept")

	m := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store})
	if got, err := openToken(m, sc, token); err != nil || got != "kept" {
		t.Errorf("open with another instance keyset = %q, %v; want it through the escrow copy", got, err)
	}

	noEscrow := newTestManager(t, Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store, Stores: []string{KindInstance}})
	if _, err := openToken(noEscrow, sc, token); err == nil {
		t.Error("a safe opened with another instance keyset and no copy")
	}
}

// A copy opens the safe it was made for, bound to the place of each secret
// as through the store of the safe: a value moved elsewhere does not open.
func TestOpenThroughACopyStaysBound(t *testing.T) {
	store := newMemSafeStorage()
	sc := objectContext()
	token, _ := escrowedSafe(t, store, sc, "kept")

	m := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store})
	moved := sc
	moved.ObjectId = "another"
	if _, err := openToken(m, moved, token); err == nil {
		t.Error("a value opened through a copy in another object")
	}
}

// A safe whose kind has no store here, but that a copy opens, does not keep
// happyDomain from starting.
// A safe of a backup whose kind has no store here, but that a copy opens, is
// restored, as it does not keep happyDomain from starting.
func TestCheckSafeAcceptsACopy(t *testing.T) {
	store := newMemSafeStorage()
	sc := objectContext()
	escrowedSafe(t, store, sc, "kept")
	safe, err := store.GetSafeByOwner(sc.Owner, KindInstance)
	if err != nil {
		t.Fatal(err)
	}

	if err := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store}).CheckSafe(context.Background(), safe); err != nil {
		t.Errorf("CheckSafe with the escrow store: %v", err)
	}
	if err := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store, Stores: []string{string(PolicyPlaintext), fakeKind}}).CheckSafe(context.Background(), safe); err == nil {
		t.Error("CheckSafe with neither the instance nor the escrow store succeeded")
	}
}

func TestCheckSafeKindsAcceptsACopy(t *testing.T) {
	store := newMemSafeStorage()
	escrowedSafe(t, store, objectContext(), "kept")

	if err := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store}).checkSafeKinds(); err != nil {
		t.Errorf("checkSafeKinds with the escrow store: %v", err)
	}
	if err := newTestManager(t, Config{Policy: PolicyPlaintext, Safes: store, Stores: []string{string(PolicyPlaintext), fakeKind}}).checkSafeKinds(); err == nil {
		t.Error("checkSafeKinds with neither the instance nor the escrow store succeeded")
	}
}
