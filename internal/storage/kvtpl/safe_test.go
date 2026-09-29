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
	"time"

	happydns "git.happydns.org/happyDomain/model"
)

func newSafe(owner happydns.Identifier, kind string) *happydns.Safe {
	id, _ := happydns.NewRandomIdentifier()
	return &happydns.Safe{
		Id:        id,
		Owner:     owner,
		Kind:      kind,
		Keyring:   []happydns.WrappedKeyset{{KEK: "instance", Blob: []byte{1, 2, 3}}},
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
}

func TestSafeCRUD(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	safe := newSafe(owner, "instance")

	if err := s.CreateSafe(safe); err != nil {
		t.Fatalf("CreateSafe: %v", err)
	}
	if err := s.CreateSafe(safe); err == nil {
		t.Error("CreateSafe twice with the same id succeeded")
	}

	got, err := s.GetSafe(safe.Id)
	if err != nil {
		t.Fatalf("GetSafe: %v", err)
	}
	if !got.Owner.Equals(owner) || got.Kind != "instance" || len(got.Keyring) != 1 || got.Keyring[0].KEK != "instance" || !got.CreatedAt.Equal(safe.CreatedAt) {
		t.Errorf("GetSafe = %+v, want %+v", got, safe)
	}

	safe.Keyring = append(safe.Keyring, happydns.WrappedKeyset{KEK: "other", Blob: []byte{4}})
	if err := s.UpdateSafe(safe); err != nil {
		t.Fatalf("UpdateSafe: %v", err)
	}
	if got, _ := s.GetSafe(safe.Id); len(got.Keyring) != 2 {
		t.Errorf("UpdateSafe not applied: %+v", got)
	}

	if err := s.DeleteSafe(safe.Id); err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if _, err := s.GetSafe(safe.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafe after delete = %v, want ErrSafeNotFound", err)
	}
	if _, err := s.GetSafeByOwner(owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner after delete = %v, want ErrSafeNotFound", err)
	}
}

func TestSafeOwnerIndex(t *testing.T) {
	s := newStorage(t)
	alice, _ := happydns.NewRandomIdentifier()
	bob, _ := happydns.NewRandomIdentifier()

	aliceSafe := newSafe(alice, "instance")
	bobSafe := newSafe(bob, "instance")
	for _, safe := range []*happydns.Safe{aliceSafe, bobSafe} {
		if err := s.CreateSafe(safe); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.GetSafeByOwner(alice, "instance")
	if err != nil || !got.Id.Equals(aliceSafe.Id) {
		t.Errorf("GetSafeByOwner(alice) = %v, %v; want alice's safe", got, err)
	}
	if _, err := s.GetSafeByOwner(alice, "other"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("GetSafeByOwner(alice, other) = %v, want ErrSafeNotFound", err)
	}

	// One safe of a kind per owner.
	if err := s.CreateSafe(newSafe(alice, "instance")); err == nil {
		t.Error("a second instance safe for the same owner was created")
	}

	iter, err := s.ListAllSafes()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllSafes = %d safes, want 2 (index entries must not be listed)", n)
	}
}
