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
	"bytes"
	"encoding/base64"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

const keyMaterial = "raw-key-material"

func seed(t *testing.T) (storage.Storage, *happydns.User) {
	t.Helper()

	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("failed to instantiate storage: %v", err)
	}

	user := &happydns.User{
		Id:    happydns.Identifier([]byte("backup-user")),
		Email: "export@example.com",
	}
	if err := db.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	p := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "DDNSServer",
			Owner:   user.Id,
			Comment: "Exported provider",
		},
		Provider: &providers.DDNSServer{
			Server:  "127.0.0.1",
			KeyName: "exportkey",
			KeyAlgo: "hmac-sha256",
			KeyBlob: []byte(keyMaterial),
		},
	}
	if err := db.CreateProvider(p); err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	return db, user
}

// The GDPR export is a file that leaves happyDomain, so it must not carry a
// credential that still opens a door.
func TestBackupUserRedactsProviderSecrets(t *testing.T) {
	db, user := seed(t)

	ret := backup.NewUsecase(db).BackupUser(user)

	if len(ret.Errors) > 0 {
		t.Fatalf("unexpected export errors: %v", ret.Errors)
	}
	if len(ret.Providers) != 1 {
		t.Fatalf("exported %d providers, want 1", len(ret.Providers))
	}

	body := ret.Providers[0].Provider

	// KeyBlob is a []byte, so both the credential and the sentinel travel
	// base64-encoded.
	if bytes.Contains(body, []byte(base64.StdEncoding.EncodeToString([]byte(keyMaterial)))) {
		t.Errorf("export still carries the key material: %s", body)
	}
	if !bytes.Contains(body, []byte(base64.StdEncoding.EncodeToString([]byte(happydns.RedactedSecret)))) {
		t.Errorf("export carries no sentinel in place of the secret: %s", body)
	}

	// Everything the user needs to recognise the provider must survive.
	if !bytes.Contains(body, []byte("exportkey")) {
		t.Errorf("export lost the untagged settings: %s", body)
	}
	if ret.Providers[0].Comment != "Exported provider" {
		t.Errorf("Comment = %q, want it preserved", ret.Providers[0].Comment)
	}
}

// Redacting the export must not touch what is stored, nor the administrative
// Backup() that Restore depends on.
func TestBackupKeepsProviderSecrets(t *testing.T) {
	db, user := seed(t)

	uc := backup.NewUsecase(db)
	_ = uc.BackupUser(user)

	ret := uc.Backup()

	if len(ret.Providers) != 1 {
		t.Fatalf("exported %d providers, want 1", len(ret.Providers))
	}

	if !bytes.Contains(ret.Providers[0].Provider, []byte(base64.StdEncoding.EncodeToString([]byte(keyMaterial)))) {
		t.Errorf("admin backup lost the key material: %s", ret.Providers[0].Provider)
	}
}
