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
	"errors"
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
	m, err := secret.NewManager(secret.PolicyPlaintext)
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
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	id1 := happydns.Identifier{1}
	want := map[string]string{
		id1.String(): `{"host":"h","apikey":"legacy"}`,
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

func TestRestoreSealsClear(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	owner := happydns.Identifier{0xaa}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"legacy"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	for id, want := range map[byte]string{
		1: `{"host":"h","apikey":"legacy"}`,
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

// A sealed value no safe opens is refused rather than stored where nobody
// knows whether it opens.
func TestRestoreRefusesSealedThatDoesNotOpen(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(happydns.Identifier{0xaa}, 1, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
		},
	}
	if err := uc.Restore(in); !errors.Is(err, secret.ErrUnknownSafe) {
		t.Errorf("Restore = %v, want it to report the unknown safe", err)
	}
	if _, err := db.GetProvider(happydns.Identifier{1}); err == nil {
		t.Error("the provider whose secret does not open was stored")
	}
}

// A user export carries placeholders instead of secrets. They stand for no
// value, as on creation: the provider is restored without them, rather than
// storing them as credentials or skipping the provider and its domains.
func TestRestoreClearsRedactedSecrets(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))
	owner := happydns.Identifier{0xaa}

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"`+happydns.RedactedSecret+`"}`),
		},
		Domains: []*happydns.Domain{
			{Id: happydns.Identifier{2}, Owner: owner, ProviderId: happydns.Identifier{1}, DomainName: "example.com."},
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	got, err := db.GetProvider(happydns.Identifier{1})
	if err != nil {
		t.Fatalf("the provider of a user export was not restored: %v", err)
	}
	if want := `{"host":"h","apikey":""}`; string(got.Provider) != want {
		t.Errorf("restored = %s, want %s", got.Provider, want)
	}
	if _, err := db.GetDomain(happydns.Identifier{2}); err != nil {
		t.Errorf("the domain of the restored provider is missing: %v", err)
	}
}

// A provider that cannot be restored takes its domains with it: restored
// alone, they would point at a provider that does not exist.
func TestRestoreSkipsDomainsOfUnrestoredProvider(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))
	owner := happydns.Identifier{0xaa}

	unknownType := secretProviderMessage(owner, 5, `{}`)
	unknownType.Type = "NoSuchProvider"

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
			secretProviderMessage(owner, 3, `{"host":"h","apikey":"fine"}`),
			unknownType,
		},
		Domains: []*happydns.Domain{
			{Id: happydns.Identifier{2}, Owner: owner, ProviderId: happydns.Identifier{1}, DomainName: "sealed.example."},
			{Id: happydns.Identifier{4}, Owner: owner, ProviderId: happydns.Identifier{3}, DomainName: "kept.example."},
			{Id: happydns.Identifier{6}, Owner: owner, ProviderId: happydns.Identifier{5}, DomainName: "unknown.example."},
		},
	}
	if err := uc.Restore(in); err == nil {
		t.Error("Restore succeeded, want the unrestored providers reported")
	}

	for _, id := range []byte{2, 6} {
		if _, err := db.GetDomain(happydns.Identifier{id}); err == nil {
			t.Errorf("domain %d was restored without its provider", id)
		}
	}
	if _, err := db.GetDomain(happydns.Identifier{4}); err != nil {
		t.Errorf("domain 4, whose provider was restored, is missing: %v", err)
	}
}

// A domain whose provider is not part of the backup is restored: its provider
// may already be in the database.
func TestRestoreKeepsDomainsOfProvidersOutsideTheBackup(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	uc := backup.NewUsecase(db, plaintextSecrets(t))

	in := &happydns.Backup{
		Domains: []*happydns.Domain{
			{Id: happydns.Identifier{2}, Owner: happydns.Identifier{0xaa}, ProviderId: happydns.Identifier{9}, DomainName: "example.com."},
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if _, err := db.GetDomain(happydns.Identifier{2}); err != nil {
		t.Errorf("domain not restored: %v", err)
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
	if found != 1 {
		t.Errorf("exported %d secret providers, want 1", found)
	}
}
