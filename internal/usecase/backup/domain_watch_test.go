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

package backup_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	"git.happydns.org/happyDomain/model"
)

// The watched domains come back with their state, under the same identifier,
// from the administrative backup as from the export of their user.
func TestBackupRestoreDomainWatches(t *testing.T) {
	src, user := seed(t)

	expiration := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	w := &happydns.DomainWatch{
		UserId:         user.Id,
		DomainName:     "example.com",
		DisplayName:    "example.com",
		Note:           "for the shop",
		Phase:          happydns.DomainWatchExpiring,
		ExpirationDate: &expiration,
	}
	if err := src.CreateDomainWatch(w); err != nil {
		t.Fatal(err)
	}

	uc := backup.NewUsecase(src, plaintextSecrets(t))
	if dump := uc.BackupUser(user); len(dump.DomainWatches) != 1 {
		t.Errorf("user export holds %d watches, want 1", len(dump.DomainWatches))
	}

	dump := uc.Backup()
	if len(dump.Errors) > 0 {
		t.Fatalf("backup errors: %v", dump.Errors)
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	var restored happydns.Backup
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := backup.NewUsecase(dst, plaintextSecrets(t)).Restore(context.Background(), &restored); err != nil {
			t.Fatalf("Restore: %v", err)
		}
	}

	watches, err := dst.ListDomainWatchesByUser(user.Id)
	if err != nil || len(watches) != 1 {
		t.Fatalf("ListDomainWatchesByUser = %v, %v; want the watch, once", watches, err)
	}
	got := watches[0]
	if !got.Id.Equals(w.Id) || got.Note != "for the shop" || got.Phase != happydns.DomainWatchExpiring || got.ExpirationDate == nil || !got.ExpirationDate.Equal(expiration) {
		t.Errorf("restored watch = %+v, want %+v", got, w)
	}
}

// The export of a user carries their watches only.
func TestBackupUserCarriesOnlyTheirWatches(t *testing.T) {
	src, user := seed(t)
	other := happydns.Identifier{0x42}
	for _, w := range []*happydns.DomainWatch{
		{UserId: user.Id, DomainName: "mine.example"},
		{UserId: other, DomainName: "theirs.example"},
	} {
		if err := src.CreateDomainWatch(w); err != nil {
			t.Fatal(err)
		}
	}

	dump := backup.NewUsecase(src, plaintextSecrets(t)).BackupUser(user)
	if len(dump.DomainWatches) != 1 || dump.DomainWatches[0].DomainName != "mine.example" {
		t.Errorf("user export holds %+v, want only the watch of the user", dump.DomainWatches)
	}
}
