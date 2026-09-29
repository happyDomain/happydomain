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
	"errors"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
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

// newInstanceKey returns the key of a fresh instance keyset.
func newInstanceKey(t *testing.T) *secret.InstanceKey {
	t.Helper()
	h, err := secret.GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	key, err := secret.NewInstanceKey(h)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// keyedSecrets returns two Managers sharing a fresh instance keyset and the
// safes of db: one sealing under the instance policy, one back to plaintext.
func keyedSecrets(t *testing.T, db storage.Storage) (instance, plaintext *secret.Manager) {
	t.Helper()
	key := newInstanceKey(t)
	var err error
	if instance, err = secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db}); err != nil {
		t.Fatal(err)
	}
	if plaintext, err = secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db}); err != nil {
		t.Fatal(err)
	}
	return
}

// sealedFor returns value sealed by m for the provider id of owner.
func sealedFor(t *testing.T, m *secret.Manager, owner happydns.Identifier, id byte, value string) string {
	t.Helper()
	body := &BackupSecretProvider{Host: "h", ApiKey: happydns.NewSecret(value)}
	p := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{id}, Owner: owner}}
	if err := m.SealObject(context.Background(), providerUC.SecretContext(p), body); err != nil {
		t.Fatal(err)
	}
	return body.ApiKey.Token()
}

// restoredApiKey opens the provider id stored in db and returns its key.
func restoredApiKey(t *testing.T, db storage.Storage, m *secret.Manager, id byte) string {
	t.Helper()
	msg, err := db.GetProvider(happydns.Identifier{id})
	if err != nil {
		t.Fatalf("provider %d was not restored: %v", id, err)
	}
	p, err := providerUC.ParseProvider(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.OpenObject(context.Background(), providerUC.SecretContext(p), p.Provider); err != nil {
		t.Fatalf("the restored provider does not open: %v", err)
	}
	return p.Provider.(*BackupSecretProvider).ApiKey.Reveal()
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
	instance, plaintext := keyedSecrets(t, db)
	uc := backup.NewUsecase(db, plaintext)
	token := sealedFor(t, instance, user.Id, 2, "sealed-key")

	// Restore puts the records in place; Backup must give them back as is.
	in := &happydns.Backup{
		Version: db.SchemaVersion(),
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(user.Id, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(user.Id, 2, `{"host":"h","apikey":"`+token+`"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	id1, id2 := happydns.Identifier{1}, happydns.Identifier{2}
	want := map[string]string{
		id1.String(): `{"host":"h","apikey":"legacy"}`,
		id2.String(): `{"host":"h","apikey":"` + token + `"}`,
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
	instance, plaintext := keyedSecrets(t, db)
	uc := backup.NewUsecase(db, plaintext)
	token := sealedFor(t, instance, owner, 2, "sealed-key")

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(owner, 2, `{"host":"h","apikey":"`+token+`"}`),
		},
	}
	if err := uc.Restore(in); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	for id, want := range map[byte]string{
		1: `{"host":"h","apikey":"legacy"}`,
		2: `{"host":"h","apikey":"` + token + `"}`,
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
	owner := happydns.Identifier{0xaa}
	instance, plaintext := keyedSecrets(t, db)
	uc := backup.NewUsecase(db, plaintext)

	in := &happydns.Backup{
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(owner, 1, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`),
			// Sealed for provider 3, restored as provider 2.
			secretProviderMessage(owner, 2, `{"host":"h","apikey":"`+sealedFor(t, instance, owner, 3, "sealed-key")+`"}`),
		},
	}
	if err := uc.Restore(in); !errors.Is(err, secret.ErrUnknownSafe) {
		t.Errorf("Restore = %v, want it to report the unknown safe", err)
	}
	for _, id := range []byte{1, 2} {
		if _, err := db.GetProvider(happydns.Identifier{id}); err == nil {
			t.Errorf("provider %d, whose secret does not open, was stored", id)
		}
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
	instance, plaintext := keyedSecrets(t, db)
	uc := backup.NewUsecase(db, plaintext)

	in := &happydns.Backup{
		Version: db.SchemaVersion(),
		Providers: []*happydns.ProviderMessage{
			secretProviderMessage(user.Id, 1, `{"host":"h","apikey":"legacy"}`),
			secretProviderMessage(user.Id, 2, `{"host":"h","apikey":"`+sealedFor(t, instance, user.Id, 2, "sealed-key")+`"}`),
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

// The administrative backup carries the safes with the sealed values: restored
// on an instance holding the same keyset, everything opens; without it, the
// instance refuses to start.
func TestBackupRestoreSafes(t *testing.T) {
	key := newInstanceKey(t)
	dump, srcUC := sealedDump(t, key)

	// Through JSON, as the admin API hands it out.
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
	dstSecrets, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: dst})
	if err := backup.NewUsecase(dst, dstSecrets).Restore(&restored); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if got := restoredApiKey(t, dst, dstSecrets, 1); got != "precious" {
		t.Errorf("restored key = %q", got)
	}

	// The same database without the keyset: refuse to start.
	if err := secret.StartupCheck(secret.PolicyPlaintext, nil, dst); err == nil {
		t.Error("an instance holding restored safes started without the keyset")
	}

	// The user export carries no safe.
	if u := srcUC.BackupUser(&happydns.User{Id: happydns.Identifier{0xaa}}); len(u.Safes) != 0 {
		t.Error("the user export carries safes")
	}
}

// sealedDump returns the administrative backup of an instance holding one
// provider, sealed under the instance keyset key, and the usecase of that
// instance.
func sealedDump(t *testing.T, key *secret.InstanceKey) (*happydns.Backup, *backup.Usecase) {
	t.Helper()
	src, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	srcSecrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: src})
	if err != nil {
		t.Fatal(err)
	}

	owner := happydns.Identifier{0xaa}
	if err := src.CreateOrUpdateUser(&happydns.User{Id: owner, Email: "owner@example.com"}); err != nil {
		t.Fatal(err)
	}
	p, err := providerUC.ParseProvider(secretProviderMessage(owner, 1, `{"host":"h","apikey":"precious"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := srcSecrets.SealObject(context.Background(), providerUC.SecretContext(p), p.Provider); err != nil {
		t.Fatal(err)
	}
	if err := src.UpdateProvider(p); err != nil {
		t.Fatal(err)
	}

	uc := backup.NewUsecase(src, srcSecrets)
	dump := uc.Backup()
	if len(dump.Safes) != 1 || !dump.Safes[0].Owner.Equals(owner) {
		t.Fatalf("backup safes = %+v, want the owner's safe", dump.Safes)
	}
	return &dump, uc
}

// A safe this instance cannot open is not restored: stored, it would keep an
// instance without keyset from starting again, and stay unusable on one with
// another keyset. The providers sealed in it are not restored either.
func TestRestoreSkipsSafesThatDoNotOpen(t *testing.T) {
	key := newInstanceKey(t)
	other := newInstanceKey(t)

	for name, newManager := range map[string]func(storage.Storage) (*secret.Manager, error){
		"without keyset": func(storage.Storage) (*secret.Manager, error) {
			return secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
		},
		"without keyset, with safes": func(db storage.Storage) (*secret.Manager, error) {
			return secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, Safes: db})
		},
		"with another keyset": func(db storage.Storage) (*secret.Manager, error) {
			return secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: other, Safes: db})
		},
	} {
		t.Run(name, func(t *testing.T) {
			dump, _ := sealedDump(t, key)

			dst, err := inmemory.Instantiate()
			if err != nil {
				t.Fatal(err)
			}
			dstSecrets, err := newManager(dst)
			if err != nil {
				t.Fatal(err)
			}

			if err := backup.NewUsecase(dst, dstSecrets).Restore(dump); err == nil {
				t.Error("Restore reported no error for a safe it cannot open")
			}

			iter, err := dst.ListAllSafes()
			if err != nil {
				t.Fatal(err)
			}
			defer iter.Close()
			if iter.Next() {
				t.Errorf("safe %s restored, though it does not open here", iter.Item().Id.String())
			}

			if _, err := dst.GetProvider(happydns.Identifier{1}); err == nil {
				t.Error("a provider sealed in a skipped safe was restored")
			}

			// Without keyset, the instance still starts.
			if err := secret.StartupCheck(secret.PolicyPlaintext, nil, dst); err != nil && name != "with another keyset" {
				t.Errorf("StartupCheck after the restore: %v", err)
			}
		})
	}
}

// A backup can carry a safe other than the one its owner already has here:
// the instance safe, once an administrator set a secret option since, or the
// safe of a user who sealed something since. Both are kept, and what was
// sealed in either still opens. New secrets keep going to the one in place.
func TestRestoreSafeBesideAnExistingOne(t *testing.T) {
	ctx := context.Background()
	key := newInstanceKey(t)
	dump, _ := sealedDump(t, key)

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	dstSecrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: dst})
	if err != nil {
		t.Fatal(err)
	}

	owner := happydns.Identifier{0xaa}
	local := sealedFor(t, dstSecrets, owner, 7, "local-key")
	inPlace, err := dst.GetSafeByOwner(owner, secret.KindInstance)
	if err != nil {
		t.Fatal(err)
	}

	if err := backup.NewUsecase(dst, dstSecrets).Restore(dump); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if got := restoredApiKey(t, dst, dstSecrets, 1); got != "precious" {
		t.Errorf("restored key = %q", got)
	}

	p, err := providerUC.ParseProvider(secretProviderMessage(owner, 7, `{"host":"h","apikey":"`+local+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := dstSecrets.OpenObject(ctx, providerUC.SecretContext(p), p.Provider); err != nil {
		t.Errorf("what was sealed before the restore no longer opens: %v", err)
	}

	if s, err := dst.GetSafeByOwner(owner, secret.KindInstance); err != nil || !s.Id.Equals(inPlace.Id) {
		t.Errorf("new secrets now go to safe %v (%v), want the one in place %s", s, err, inPlace.Id.String())
	}
}
