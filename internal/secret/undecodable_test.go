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
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// undecodableSafes lists, after the safes it holds, a record that does not
// decode, as a corrupted one.
type undecodableSafes struct {
	*memStartupStorage
}

func newUndecodableSafes() undecodableSafes {
	return undecodableSafes{newMemStartupStorage()}
}

func (u undecodableSafes) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	inner, err := u.Safes.ListAllSafes()
	if err != nil {
		return nil, err
	}
	var entries []iterEntry[happydns.Safe]
	for inner.Next() {
		entries = append(entries, iterEntry[happydns.Safe]{key: "safe-" + inner.Item().Id.String(), item: inner.Item()})
	}
	entries = append(entries, iterEntry[happydns.Safe]{key: "safe-corrupt", err: errors.New("invalid character")})
	return newEntriesIterator(entries...), nil
}

// sealFor seals a value for owner, creating their instance safe.
func sealFor(t *testing.T, m *Manager, owner happydns.Identifier) {
	t.Helper()
	sc := objectContext()
	sc.Owner = owner
	_ = sealOne(t, m, sc, "v")
}

// One damaged safe must not keep another user's account from being deleted.
func TestDeleteOwnerSafesGoesPastAnUndecodableSafe(t *testing.T) {
	store := newUndecodableSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store})
	gone, kept := happydns.Identifier{0x01}, happydns.Identifier{0x02}
	sealFor(t, m, gone)
	sealFor(t, m, kept)

	if err := m.DeleteOwnerSafes(gone); err != nil {
		t.Fatalf("DeleteOwnerSafes: %v", err)
	}
	if _, err := store.GetSafeByOwner(gone, KindInstance); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the deleted user's safe is still there: %v", err)
	}
	if _, err := store.GetSafeByOwner(kept, KindInstance); err != nil {
		t.Errorf("another user's safe was deleted: %v", err)
	}
}

// One damaged safe must not keep happyDomain from starting.
func TestStartupCheckGoesPastAnUndecodableSafe(t *testing.T) {
	store := newUndecodableSafes()
	if err := StartupCheck(PolicyPlaintext, nil, store); err != nil {
		t.Errorf("StartupCheck without keyset = %v", err)
	}

	key := testInstanceKey(t)
	if err := StartupCheck(PolicyInstance, key, store); err != nil {
		t.Errorf("StartupCheck with a keyset = %v", err)
	}

	// Readable instance safes still require their keyset.
	if _, err := newSafeRegistry(store, key).instanceSafe(happydns.Identifier{0x01}); err != nil {
		t.Fatal(err)
	}
	if err := StartupCheck(PolicyPlaintext, nil, store); err == nil {
		t.Error("StartupCheck accepted instance safes without their keyset")
	}
}

// damagedOwnSafe reads the safe of one owner as a record that does not decode,
// both when listing the safes and when looking it up, directly or through
// the owner index, as the real storage does.
type damagedOwnSafe struct {
	*memStartupStorage
	id happydns.Identifier
}

var errDoesNotDecode = errors.New("invalid character")

func (d *damagedOwnSafe) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	inner, err := d.Safes.ListAllSafes()
	if err != nil {
		return nil, err
	}
	var entries []iterEntry[happydns.Safe]
	for inner.Next() {
		e := iterEntry[happydns.Safe]{key: "safe-" + inner.Item().Id.String(), item: inner.Item()}
		if inner.Item().Id.Equals(d.id) {
			e.item, e.err = nil, errDoesNotDecode
		}
		entries = append(entries, e)
	}
	return newEntriesIterator(entries...), nil
}

func (d *damagedOwnSafe) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	if id.Equals(d.id) {
		return nil, errDoesNotDecode
	}
	return d.Safes.GetSafe(id)
}

func (d *damagedOwnSafe) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	safe, err := d.Safes.GetSafeByOwner(owner, kind)
	if err == nil && safe.Id.Equals(d.id) {
		return nil, errDoesNotDecode
	}
	return safe, err
}

// Deleting an account shreds what was sealed for its owner. When the safe of
// that owner is the damaged record, it cannot be deleted for certain: the
// deletion is refused, before anything is deleted, rather than reported as
// done while the key stays behind.
func TestDeleteOwnerSafesRefusesWhenTheOwnersSafeIsDamaged(t *testing.T) {
	store := &damagedOwnSafe{memStartupStorage: newMemStartupStorage()}
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store})
	gone, kept := happydns.Identifier{0x01}, happydns.Identifier{0x02}
	sealFor(t, m, gone)
	sealFor(t, m, kept)

	own, err := store.Safes.GetSafeByOwner(gone, KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	store.id = own.Id

	if err := m.DeleteOwnerSafes(gone); err == nil {
		t.Fatal("DeleteOwnerSafes succeeded while the owner's safe does not decode")
	}
	if store.Len() != 2 {
		t.Errorf("%d safes left, want both: a refused deletion deletes nothing", store.Len())
	}
}
