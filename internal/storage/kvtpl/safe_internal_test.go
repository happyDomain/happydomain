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

package database

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// An owner index outliving its safe (partial restore, batch not atomic on the
// backend) must not keep the owner from ever getting a safe of that kind.
func TestCreateSafeReplacesDanglingOwnerIndex(t *testing.T) {
	s := &KVStorage{db: newFakeKV()}
	owner, _ := happydns.NewRandomIdentifier()

	newSafe := func() *happydns.Safe {
		id, _ := happydns.NewRandomIdentifier()
		return &happydns.Safe{Id: id, Owner: owner, Kind: "instance"}
	}

	lost := newSafe()
	if err := s.CreateSafe(lost); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Delete(safePrimaryKey(lost.Id)); err != nil {
		t.Fatal(err)
	}

	if _, err := s.GetSafeByOwner(owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Fatalf("GetSafeByOwner through a dangling index = %v, want ErrSafeNotFound", err)
	}

	replacement := newSafe()
	if err := s.CreateSafe(replacement); err != nil {
		t.Fatalf("CreateSafe over a dangling index: %v", err)
	}
	got, err := s.GetSafeByOwner(owner, "instance")
	if err != nil || !got.Id.Equals(replacement.Id) {
		t.Errorf("GetSafeByOwner = %v, %v; want the replacement safe", got, err)
	}

	// A live safe still blocks a second one.
	if err := s.CreateSafe(newSafe()); !errors.Is(err, happydns.ErrAlreadyExists) {
		t.Errorf("CreateSafe next to a live safe = %v, want ErrAlreadyExists", err)
	}
}
