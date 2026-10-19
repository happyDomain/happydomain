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
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

func TestTidySafesRemovesOnlyOrphans(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	alive := &happydns.User{Id: happydns.Identifier("alive"), Email: "alive@example.com"}
	if err := db.CreateOrUpdateUser(alive); err != nil {
		t.Fatal(err)
	}

	aliveSafe := &happydns.Safe{Id: happydns.Identifier{0x01}, Owner: alive.Id, Kind: "instance"}
	orphanSafe := &happydns.Safe{Id: happydns.Identifier{0x02}, Owner: happydns.Identifier("gone"), Kind: "instance"}
	for _, s := range []*happydns.Safe{aliveSafe, orphanSafe} {
		if err := db.CreateSafe(s); err != nil {
			t.Fatal(err)
		}
	}

	if err := usecase.NewTidyUpUsecase(db).TidySafes(true); err != nil {
		t.Fatalf("TidySafes: %v", err)
	}

	if _, err := db.GetSafe(aliveSafe.Id); err != nil {
		t.Errorf("the safe of an existing user was removed: %v", err)
	}
	if _, err := db.GetSafe(orphanSafe.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the orphan safe is still there: %v", err)
	}
	if _, err := db.GetSafeByOwner(orphanSafe.Owner, "instance"); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the orphan safe index is still there: %v", err)
	}
}

// A safe record that does not decode names no owner: tidy keeps it, as
// deleting it would lose its secrets for good, and still deletes the orphans.
func TestTidySafesKeepsADamagedSafe(t *testing.T) {
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := kvdb.Put("safe-damaged", "not a safe"); err != nil {
		t.Fatal(err)
	}
	orphan := &happydns.Safe{Id: happydns.Identifier{0x02}, Owner: happydns.Identifier("gone"), Kind: "instance"}
	if err := db.CreateSafe(orphan); err != nil {
		t.Fatal(err)
	}

	if err := usecase.NewTidyUpUsecase(db).TidySafes(true); err != nil {
		t.Fatalf("TidySafes: %v", err)
	}

	if ok, err := kvdb.Has("safe-damaged"); err != nil || !ok {
		t.Errorf("the damaged safe record was dropped: %v, %v", ok, err)
	}
	if _, err := db.GetSafe(orphan.Id); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("the orphan safe is still there: %v", err)
	}
}
