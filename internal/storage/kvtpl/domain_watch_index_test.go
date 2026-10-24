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
	"testing"

	"git.happydns.org/happyDomain/model"
)

// The user index keeps the entries of a live watch of a live user, and drops
// those whose user or watch is gone.
func TestTidyDomainWatchIndexes(t *testing.T) {
	s := newDiscoveryTestStore()

	alive := &happydns.User{Email: "alive@example.com"}
	if err := s.CreateOrUpdateUser(alive); err != nil {
		t.Fatal(err)
	}
	kept := &happydns.DomainWatch{UserId: alive.Id, DomainName: "kept.example"}
	if err := s.CreateDomainWatch(kept); err != nil {
		t.Fatal(err)
	}

	missingWatch, _ := happydns.NewRandomIdentifier()
	goneUser, _ := happydns.NewRandomIdentifier()
	stale := []string{
		domwatchUserKey(alive.Id, missingWatch),
		domwatchUserKey(goneUser, kept.Id),
		domwatchUserPrefix + "garbage",
	}
	for _, key := range stale {
		if err := s.db.Put(key, ""); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.TidyDomainWatchIndexes(); err != nil {
		t.Fatalf("TidyDomainWatchIndexes: %v", err)
	}

	for _, key := range stale {
		if ok, _ := s.db.Has(key); ok {
			t.Errorf("stale index entry %q is still there", key)
		}
	}
	if got, err := s.ListDomainWatchesByUser(alive.Id); err != nil || len(got) != 1 || !got[0].Id.Equals(kept.Id) {
		t.Errorf("ListDomainWatchesByUser = %v, %v; want the kept watch", got, err)
	}
}
