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

package usecase_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// The domains a deleted user watched are no longer looked up.
func TestTidyRemovesOrphanDomainWatches(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	alive := &happydns.User{Id: newTestId(t), Email: "alive@example.com"}
	if err := db.CreateOrUpdateUser(alive); err != nil {
		t.Fatal(err)
	}

	kept := &happydns.DomainWatch{UserId: alive.Id, DomainName: "kept.example"}
	orphan := &happydns.DomainWatch{UserId: newTestId(t), DomainName: "orphan.example"}
	for _, w := range []*happydns.DomainWatch{kept, orphan} {
		if err := db.CreateDomainWatch(w); err != nil {
			t.Fatal(err)
		}
	}

	if err := usecase.NewTidyUpUsecase(db).TidyAll(true); err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetDomainWatch(kept.Id); err != nil {
		t.Errorf("the watch of a user still there was removed: %v", err)
	}
	if _, err := db.GetDomainWatch(orphan.Id); !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		t.Errorf("the watch of a deleted user is still there: %v", err)
	}
	if list, _ := db.ListDomainWatchesByUser(orphan.UserId); len(list) != 0 {
		t.Errorf("the index of the deleted user still lists %d watches", len(list))
	}
}
