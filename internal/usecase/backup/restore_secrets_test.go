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
	"encoding/json"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	happydns "git.happydns.org/happyDomain/model"
)

// BackupSecretProvider holds a happydns.Secret, as every provider will.
type BackupSecretProvider struct {
	Host   string          `json:"host"`
	ApiKey happydns.Secret `json:"apikey"`
}

func (*BackupSecretProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

func init() {
	providerReg.RegisterProvider(func() happydns.ProviderBody { return &BackupSecretProvider{} }, happydns.ProviderInfos{Name: "backup secret test"})
}

func plaintextSecrets(t *testing.T) *secret.Manager {
	t.Helper()
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func secretProviderMessage(owner happydns.Identifier, id byte, body string) *happydns.ProviderMessage {
	return &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type:  "BackupSecretProvider",
			Id:    happydns.Identifier{id},
			Owner: owner,
		},
		Provider: json.RawMessage(body),
	}
}

func TestBackupCopiesStoredSecretsVerbatim(t *testing.T) {
	db, user := seed(t)
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	// Restore puts the records in place; Backup must give them back as is.
	in := &happydns.Backup{
		Version: db.SchemaVersion(),
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(user.Id, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(user.Id, 2, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	id1, id2 := happydns.Identifier{1}, happydns.Identifier{2}
	want := map[string]string{
		id1.String(): `{"host":"h","apikey":"legacy"}`,
		id2.String(): `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`,
	}

	ret := uc.Backup()
	found := 0
	for _, pm := range ret.Providers {
		w, ok := want[pm.Id.String()]
		if !ok {
			continue
		}
		found++
		if string(pm.Provider) != w {
			t.Errorf("backup of %s = %s, want %s", pm.Id.String(), pm.Provider, w)
		}
	}
	if found != len(want) {
		t.Errorf("found %d of the %d restored providers in the backup", found, len(want))
	}
}

func TestRestoreSealsClearAndKeepsSealed(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	owner := happydns.Identifier{0xaa}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(owner, 2, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	for id, want := range map[byte]string{
		1: `{"host":"h","apikey":"legacy"}`,
		2: `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`,
	} {
		got, err := db.GetProvider(happydns.Identifier{id})
		if err != nil {
			t.Fatalf("GetProvider(%d): %v", id, err)
		}
		if string(got.Provider) != want {
			t.Errorf("restored %d = %s, want %s", id, got.Provider, want)
		}
	}
}

// A user export carries placeholders instead of secrets: restoring one must
// not store them as if they were credentials.
func TestRestoreRefusesRedactedSecrets(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(happydns.Identifier{0xaa}, 1, `{"host":"h","apikey":"`+happydns.RedactedSecret+`"}`),
		},
	}
	if err := uc.Restore(in); err == nil {
		t.Error("Restore of a redacted secret succeeded, want an error")
	}
	if _, err := db.GetProvider(happydns.Identifier{1}); err == nil {
		t.Error("the provider holding a placeholder was stored")
	}
}

func TestRestoreSkipsUnparsableProvider(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	bad := secretProviderMessage(happydns.Identifier{0xaa}, 1, `{}`)
	bad.Type = "NoSuchProvider"

	if err := uc.Restore(&happydns.Backup{Providers: []*happydns.ProviderMessage{bad}}); err == nil {
		t.Error("Restore of an unknown provider type succeeded, want an error")
	}
}

// BackupUser encodes providers again after redacting them, through
// Provider.ToMessage: a legacy plaintext value must come out redacted rather
// than make the encoding fail.
func TestBackupUserRedactsSecretType(t *testing.T) {
	db, user := seed(t)
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Version: db.SchemaVersion(),
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(user.Id, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(user.Id, 2, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	ret := uc.BackupUser(user)
	if len(ret.Errors) > 0 {
		t.Fatalf("export errors: %v", ret.Errors)
	}

	want := `{"host":"h","apikey":"` + happydns.RedactedSecret + `"}`
	found := 0
	for _, pm := range ret.Providers {
		if pm.Type != "BackupSecretProvider" {
			continue
		}
		found++
		if string(pm.Provider) != want {
			t.Errorf("exported %s = %s, want %s", pm.Id.String(), pm.Provider, want)
		}
	}
	if found != 2 {
		t.Errorf("exported %d secret providers, want 2", found)
	}
}
