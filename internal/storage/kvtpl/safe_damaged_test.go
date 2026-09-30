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

package database_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	happydns "git.happydns.org/happyDomain/model"
)

// newRawStorage returns the storage along with the key-value store under it,
// to write records the storage would never write, such as damaged ones.
func newRawStorage(t *testing.T) (*inmemory.InMemoryStorage, storage.Storage) {
	t.Helper()
	raw, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	s, err := kv.NewKVDatabase(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, s
}

// damage replaces the record of safe by something that does not decode.
func damage(t *testing.T, raw *inmemory.InMemoryStorage, safe *happydns.Safe) {
	t.Helper()
	if err := raw.Put("safe-"+safe.Id.String(), "not a safe"); err != nil {
		t.Fatal(err)
	}
}

func TestListDamagedSafes(t *testing.T) {
	raw, s := newRawStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	indexed := newSafe(owner, "instance")
	unindexed := newSafe(owner, "instance")
	fine := newSafe(owner, "other")
	if err := s.CreateSafe(indexed); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreSafe(unindexed); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSafe(fine); err != nil {
		t.Fatal(err)
	}
	damage(t, raw, indexed)
	damage(t, raw, unindexed)
	if err := raw.Put("safe-not!an!id", "not a safe"); err != nil {
		t.Fatal(err)
	}

	damaged, err := s.ListDamagedSafes()
	if err != nil {
		t.Fatalf("ListDamagedSafes: %v", err)
	}
	if len(damaged) != 3 {
		t.Fatalf("ListDamagedSafes = %d records, want 3: %+v", len(damaged), damaged)
	}

	byKey := map[string]*happydns.DamagedSafe{}
	for _, d := range damaged {
		byKey[d.Key] = d
	}

	d := byKey["safe-"+indexed.Id.String()]
	if d == nil || !d.Id.Equals(indexed.Id) || !d.Owner.Equals(owner) || d.Kind != "instance" {
		t.Errorf("indexed damaged safe = %+v, want its id, owner and kind", d)
	}
	d = byKey["safe-"+unindexed.Id.String()]
	if d == nil || !d.Id.Equals(unindexed.Id) || d.Owner != nil || d.Kind != "" {
		t.Errorf("unindexed damaged safe = %+v, want its id only: nothing tells its owner", d)
	}
	d = byKey["safe-not!an!id"]
	if d == nil || d.Id != nil {
		t.Errorf("damaged record under a key holding no identifier = %+v, want it listed without id", d)
	}
}

func TestDeleteDamagedSafe(t *testing.T) {
	raw, s := newRawStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	safe := newSafe(owner, "instance")
	fine := newSafe(owner, "other")
	for _, sf := range []*happydns.Safe{safe, fine} {
		if err := s.CreateSafe(sf); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteDamagedSafe(fine.Id); !errors.Is(err, happydns.ErrSafeNotDamaged) {
		t.Errorf("DeleteDamagedSafe on a safe that decodes = %v, want ErrSafeNotDamaged", err)
	}
	if _, err := s.GetSafe(fine.Id); err != nil {
		t.Errorf("a safe that decodes was deleted: %v", err)
	}

	missing, _ := happydns.NewRandomIdentifier()
	if err := s.DeleteDamagedSafe(missing); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("DeleteDamagedSafe on no record = %v, want ErrSafeNotFound", err)
	}

	damage(t, raw, safe)
	if err := s.DeleteDamagedSafe(safe.Id); err != nil {
		t.Fatalf("DeleteDamagedSafe: %v", err)
	}
	if has, _ := raw.Has("safe-" + safe.Id.String()); has {
		t.Error("the damaged record is still there")
	}
	if damaged, _ := s.ListDamagedSafes(); len(damaged) != 0 {
		t.Errorf("ListDamagedSafes after the deletion = %+v", damaged)
	}

	// The owner index went with it: the owner can get a new safe of that kind.
	if err := s.CreateSafe(newSafe(owner, "instance")); err != nil {
		t.Errorf("CreateSafe after the damaged one was deleted: %v", err)
	}
	if _, err := s.GetSafe(fine.Id); err != nil {
		t.Errorf("another safe of the owner was deleted: %v", err)
	}
}

func TestRepairSafe(t *testing.T) {
	raw, s := newRawStorage(t)
	owner, _ := happydns.NewRandomIdentifier()

	safe := newSafe(owner, "instance")
	if err := s.CreateSafe(safe); err != nil {
		t.Fatal(err)
	}

	if err := s.RepairSafe(safe); !errors.Is(err, happydns.ErrSafeNotDamaged) {
		t.Errorf("RepairSafe over a safe that decodes = %v, want ErrSafeNotDamaged", err)
	}
	if err := s.RepairSafe(newSafe(owner, "instance")); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("RepairSafe with no record to repair = %v, want ErrSafeNotFound", err)
	}

	damage(t, raw, safe)
	if err := s.RepairSafe(safe); err != nil {
		t.Fatalf("RepairSafe: %v", err)
	}
	got, err := s.GetSafeByOwner(owner, "instance")
	if err != nil || !got.Id.Equals(safe.Id) || len(got.Keyring) != 1 {
		t.Errorf("GetSafeByOwner after the repair = %+v, %v; want the repaired safe", got, err)
	}

	// A damaged safe the index does not point to gets the index, when the
	// owner has no safe of that kind any more.
	lone := newSafe(owner, "other")
	if err := s.CreateSafe(lone); err != nil {
		t.Fatal(err)
	}
	damage(t, raw, lone)
	if err := s.DeleteDamagedSafe(lone.Id); err != nil {
		t.Fatal(err)
	}
	if err := raw.Put("safe-"+lone.Id.String(), "not a safe"); err != nil {
		t.Fatal(err)
	}
	if err := s.RepairSafe(lone); err != nil {
		t.Fatalf("RepairSafe of an unindexed safe: %v", err)
	}
	if got, err := s.GetSafeByOwner(owner, "other"); err != nil || !got.Id.Equals(lone.Id) {
		t.Errorf("GetSafeByOwner after repairing an unindexed safe = %+v, %v", got, err)
	}
}

func TestRepairSafeRefusesAnIncompleteSafe(t *testing.T) {
	raw, s := newRawStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	safe := newSafe(owner, "instance")
	if err := s.CreateSafe(safe); err != nil {
		t.Fatal(err)
	}
	damage(t, raw, safe)

	noOwner := *safe
	noOwner.Owner = nil
	if err := s.RepairSafe(&noOwner); err == nil {
		t.Error("RepairSafe accepted a safe without owner")
	}
	if damaged, _ := s.ListDamagedSafes(); len(damaged) != 1 {
		t.Errorf("a refused repair changed the record: %+v", damaged)
	}
}
