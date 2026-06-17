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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

// keyMaterial is the base64 of the key, as KeyBlob holds it.
const keyMaterial = "cmF3LWtleS1tYXRlcmlhbA=="

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
			Id:      happydns.Identifier([]byte("backup-provider")),
			Owner:   user.Id,
			Comment: "Exported provider",
		},
		Provider: &providers.DDNSServer{
			Server:  "127.0.0.1",
			KeyName: "exportkey",
			KeyAlgo: "hmac-sha256",
			KeyBlob: happydns.NewSecret(keyMaterial),
		},
	}
	if err := plaintextSecrets(t).SealObject(context.Background(), providerUC.SecretContext(p), p.Provider); err != nil {
		t.Fatalf("failed to seal provider: %v", err)
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

	ret := dumpUser(t, backup.NewUsecase(db, plaintextSecrets(t)), user)

	if len(ret.Errors) > 0 {
		t.Fatalf("unexpected export errors: %v", ret.Errors)
	}
	if len(ret.Providers) != 1 {
		t.Fatalf("exported %d providers, want 1", len(ret.Providers))
	}

	body := ret.Providers[0].Provider

	if bytes.Contains(body, []byte(keyMaterial)) {
		t.Errorf("export still carries the key material: %s", body)
	}
	if !bytes.Contains(body, []byte(happydns.RedactedSecret)) {
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

	uc := backup.NewUsecase(db, plaintextSecrets(t))
	_ = dumpUser(t, uc, user)

	ret := dumpAll(t, uc)

	if len(ret.Providers) != 1 {
		t.Fatalf("exported %d providers, want 1", len(ret.Providers))
	}

	if !bytes.Contains(ret.Providers[0].Provider, []byte(keyMaterial)) {
		t.Errorf("admin backup lost the key material: %s", ret.Providers[0].Provider)
	}
}

// A provider of a type this build does not know, from a backup taken by
// another version say, is reported and skipped, the others still restored.
func TestRestoreSkipsUnparsableProvider(t *testing.T) {
	src, _ := seed(t)
	dump := dumpAll(t, backup.NewUsecase(src, plaintextSecrets(t)))
	if len(dump.Providers) != 1 {
		t.Fatalf("backup carries %d providers, want 1", len(dump.Providers))
	}
	good := dump.Providers[0]

	bad := *good
	bad.Id = happydns.Identifier([]byte("unknown-provider"))
	bad.Type = "NoSuchProvider"
	dump.Providers = append([]*happydns.ProviderMessage{&bad}, dump.Providers...)

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	if err := restoreDump(t, backup.NewUsecase(dst, plaintextSecrets(t)), context.Background(), &dump); err == nil {
		t.Error("Restore of an unknown provider type succeeded, want an error")
	}

	if _, err := dst.GetProvider(good.Id); err != nil {
		t.Errorf("the known provider was not restored: %v", err)
	}
	if _, err := dst.GetProvider(bad.Id); err == nil {
		t.Error("the unknown provider was stored")
	}
}

var fixedTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// populateStore fills store with a representative dataset spanning the
// user-scoped tables (Users, Domains, DomainsLogs, Sessions), the domain
// watches and zone templates, a positional table (CheckerConfigurations) and
// the global iterator tables (Executions, DiscoveryEntries). Providers and
// Zones are intentionally left out: they round-trip through Parse* helpers
// covered by their own tests, and their re-marshalling is not guaranteed to be
// byte-identical.
func populateStore(t *testing.T, store storage.Storage) {
	t.Helper()

	for _, email := range []string{"alice@example.com", "bob@example.com"} {
		uid := happydns.Identifier([]byte("user-" + email))

		if err := store.CreateOrUpdateUser(&happydns.User{Id: uid, Email: email, CreatedAt: fixedTime, LastSeen: fixedTime}); err != nil {
			t.Fatalf("CreateOrUpdateUser(%s): %v", email, err)
		}

		if err := store.UpdateAuthUser(&happydns.UserAuth{Id: uid, Email: email, CreatedAt: fixedTime, Password: []byte("hash")}); err != nil {
			t.Fatalf("UpdateAuthUser(%s): %v", email, err)
		}

		domain := &happydns.Domain{
			Id:          happydns.Identifier([]byte("dom-" + email)),
			Owner:       uid,
			DomainName:  email + ".example.",
			ZoneHistory: []happydns.Identifier{},
		}
		if err := store.UpdateDomain(domain); err != nil {
			t.Fatalf("UpdateDomain(%s): %v", email, err)
		}

		log := &happydns.DomainLog{
			Id:      happydns.Identifier([]byte("log-" + email)),
			IdUser:  uid,
			Date:    fixedTime,
			Content: "domain created",
			Level:   1,
		}
		if err := store.UpdateDomainLog(domain, log); err != nil {
			t.Fatalf("UpdateDomainLog(%s): %v", email, err)
		}

		session := &happydns.Session{
			Id:          "sess-" + email,
			IdUser:      uid,
			Description: "test session",
			IssuedAt:    fixedTime,
			ExpiresOn:   fixedTime.Add(time.Hour),
			ModifiedOn:  fixedTime,
		}
		if err := store.UpdateSession(session); err != nil {
			t.Fatalf("UpdateSession(%s): %v", email, err)
		}

		watch := &happydns.DomainWatch{
			Id:         happydns.Identifier([]byte("watch-" + email)),
			UserId:     uid,
			DomainName: email + ".example.",
			CreatedAt:  fixedTime,
		}
		if err := store.RestoreDomainWatch(watch); err != nil {
			t.Fatalf("RestoreDomainWatch(%s): %v", email, err)
		}

		tpl := &happydns.ZoneTemplateMessage{
			ZoneTemplateMeta: happydns.ZoneTemplateMeta{
				Id:     happydns.Identifier([]byte("tpl-" + email)),
				UserId: &uid,
				Name:   "template of " + email,
			},
		}
		if err := store.RestoreZoneTemplate(tpl); err != nil {
			t.Fatalf("RestoreZoneTemplate(%s): %v", email, err)
		}

		if err := store.UpdateCheckerConfiguration("dnssec", &uid, nil, nil, happydns.CheckerOptions{"mode": "strict"}); err != nil {
			t.Fatalf("UpdateCheckerConfiguration(%s): %v", email, err)
		}

		exec := &happydns.Execution{
			Id:        happydns.Identifier([]byte("exec-" + email)),
			CheckerID: "dnssec",
			Target:    happydns.CheckTarget{UserId: uid.String()},
			StartedAt: fixedTime,
		}
		if err := store.RestoreExecution(exec); err != nil {
			t.Fatalf("RestoreExecution(%s): %v", email, err)
		}

		entry := &happydns.StoredDiscoveryEntry{
			ProducerID: "producer1",
			Target:     happydns.CheckTarget{UserId: uid.String()},
			Type:       "ns",
			Ref:        "ref-" + email,
		}
		if err := store.RestoreDiscoveryEntry(entry); err != nil {
			t.Fatalf("RestoreDiscoveryEntry(%s): %v", email, err)
		}
	}
}

func countUsers(t *testing.T, store storage.Storage) int {
	t.Helper()
	it, err := store.ListAllUsers()
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	defer it.Close()
	n := 0
	for it.Next() {
		n++
	}
	return n
}

// Test_BackupRoundTrip checks that streaming a backup out and back in is
// lossless: a backup taken from a populated store, restored into a fresh store
// through RestoreReader, then exported again, yields the exact same bytes. The
// in-memory store iterates keys in sorted order, so a byte-for-byte comparison
// is deterministic and transitively verifies that every field of every record
// survived the round trip.
func Test_BackupRoundTrip(t *testing.T) {
	src, _ := inmemory.Instantiate()
	populateStore(t, src)

	var first bytes.Buffer
	if err := backup.NewUsecase(src, plaintextSecrets(t)).WriteBackup(&first); err != nil {
		t.Fatalf("WriteBackup (source): %v", err)
	}

	// Sanity: the backup must actually carry the data we put in, otherwise a
	// trivially empty round trip would pass.
	if !strings.Contains(first.String(), "alice@example.com") || !strings.Contains(first.String(), "template of alice@example.com") {
		t.Fatalf("backup does not contain expected data: %s", first.String())
	}

	dst, _ := inmemory.Instantiate()
	if err := backup.NewUsecase(dst, plaintextSecrets(t)).RestoreReader(context.Background(), bytes.NewReader(first.Bytes())); err != nil {
		t.Fatalf("RestoreReader: %v", err)
	}

	if n := countUsers(t, dst); n != 2 {
		t.Fatalf("restored store has %d users, want 2", n)
	}

	var second bytes.Buffer
	if err := backup.NewUsecase(dst, plaintextSecrets(t)).WriteBackup(&second); err != nil {
		t.Fatalf("WriteBackup (restored): %v", err)
	}

	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("round trip is not lossless:\n--- first ---\n%s\n--- second ---\n%s", first.String(), second.String())
	}
}

// Test_RestoreReader_SkipsExportOnlyErrorsField ensures the trailing "Errors"
// field produced on export is ignored on import rather than treated as a record
// to restore.
func Test_RestoreReader_SkipsExportOnlyErrorsField(t *testing.T) {
	dst, _ := inmemory.Instantiate()
	version := dst.SchemaVersion()

	carolID := happydns.Identifier([]byte("user-carol"))
	doc := fmt.Sprintf(`{
		"Version": %d,
		"Users": [{"id": "%s", "email": "carol@example.com"}],
		"Errors": ["unable to retrieve something", "another failure"]
	}`, version, carolID.String())

	if err := backup.NewUsecase(dst, plaintextSecrets(t)).RestoreReader(context.Background(), strings.NewReader(doc)); err != nil {
		t.Fatalf("RestoreReader: %v", err)
	}

	if n := countUsers(t, dst); n != 1 {
		t.Fatalf("restored store has %d users, want 1", n)
	}
}

// Test_RestoreReader_VersionMismatch rejects a backup whose schema version does
// not match the target database.
func Test_RestoreReader_VersionMismatch(t *testing.T) {
	dst, _ := inmemory.Instantiate()

	doc := fmt.Sprintf(`{"Version": %d}`, dst.SchemaVersion()+1)

	err := backup.NewUsecase(dst, plaintextSecrets(t)).RestoreReader(context.Background(), strings.NewReader(doc))
	if err == nil {
		t.Fatal("expected an error on schema version mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "schema version") {
		t.Fatalf("expected a schema version error, got: %v", err)
	}
}

// Test_RestoreReader_MalformedJSON surfaces a decode error (rather than
// silently succeeding) when the stream is not valid JSON, and exposes it as a
// *json.SyntaxError so callers can map it to a client error.
func Test_RestoreReader_MalformedJSON(t *testing.T) {
	dst, _ := inmemory.Instantiate()

	err := backup.NewUsecase(dst, plaintextSecrets(t)).RestoreReader(context.Background(), strings.NewReader(`{"Users": [ not json ]}`))
	if err == nil {
		t.Fatal("expected an error on malformed JSON, got nil")
	}

	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected a *json.SyntaxError, got %T: %v", err, err)
	}
}
