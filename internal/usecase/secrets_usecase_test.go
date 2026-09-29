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
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	"git.happydns.org/happyDomain/internal/usecase"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

func seedSecretProvider(t *testing.T, db storage.Storage, secrets *secret.Manager, id byte, apikey string) {
	t.Helper()
	p, err := providerUC.ParseProvider(&happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{Type: "SettingsSecretProvider", Id: happydns.Identifier{id}, Owner: happydns.Identifier{0x01}},
		Provider:     json.RawMessage(`{"apikey":"` + apikey + `"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := secrets.SealObject(context.Background(), providerUC.SecretContext(p), p.Provider); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProvider(p); err != nil {
		t.Fatal(err)
	}
}

// newSecretManagers generates an instance keyset and returns it with a
// manager of each policy, both keeping their safes in db.
func newSecretManagers(t *testing.T, db storage.Storage) (key *secret.InstanceKey, instance, plaintext *secret.Manager) {
	t.Helper()
	h, err := secret.GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	if key, err = secret.NewInstanceKey(h); err != nil {
		t.Fatal(err)
	}
	if instance, err = secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db}); err != nil {
		t.Fatal(err)
	}
	if plaintext, err = secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db}); err != nil {
		t.Fatal(err)
	}
	return key, instance, plaintext
}

// newProviderSecretsUsecase returns a SecretsUsecase holding only the
// providers of db, run under m.
func newProviderSecretsUsecase(db storage.Storage, m *secret.Manager) *usecase.SecretsUsecase {
	return usecase.NewSecretsUsecase(m, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, m),
	}, db)
}

// newKVTestStorage returns the real storage over an in-memory store, and that
// store, to write records the storage itself would not.
func newKVTestStorage(t *testing.T) (*inmemory.InMemoryStorage, storage.Storage) {
	t.Helper()
	raw, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, db
}

func TestSecretsUsecase(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	_, instance, plaintext := newSecretManagers(t, db)

	seedSecretProvider(t, db, plaintext, 1, "legacy-1")
	seedSecretProvider(t, db, plaintext, 2, "legacy-2")
	seedSecretProvider(t, db, instance, 3, "sealed-3")

	uc := newProviderSecretsUsecase(db, instance)

	status, err := uc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Policy != "instance" {
		t.Errorf("Policy = %q", status.Policy)
	}
	pc := status.Objects[providerUC.SecretObjectType]
	if pc.Clear != 2 || pc.Sealed[secret.KindInstance] != 1 || pc.Unreadable != 0 {
		t.Errorf("provider counts = %+v, want 2 clear and 1 sealed", pc)
	}
	if len(status.InstanceKeys) != 1 || status.InstanceKeys[0].Safes != 1 || !status.InstanceKeys[0].Primary {
		t.Errorf("instance keys = %+v, want the primary wrapping one safe", status.InstanceKeys)
	}

	reports, err := uc.Reseal(ctx)
	if err != nil || len(reports) != 1 || reports[0].Changed != 2 {
		t.Fatalf("Reseal = %+v, %v; want 2 providers changed", reports, err)
	}

	status, _ = uc.Status(ctx)
	if pc := status.Objects[providerUC.SecretObjectType]; pc.Clear != 0 || pc.Sealed[secret.KindInstance] != 3 {
		t.Errorf("after reseal = %+v, want everything sealed", pc)
	}

	if report, err := uc.Rewrap(ctx); err != nil || report.Changed != 0 || report.Failed != 0 {
		t.Errorf("Rewrap = %+v, %v; want nothing to do on a single key", report, err)
	}
}

func TestSecretsUsecaseDropSafes(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	key, instance, plaintext := newSecretManagers(t, db)
	if err := key.VerifyCheck(db); err != nil {
		t.Fatal(err)
	}

	seedSecretProvider(t, db, instance, 1, "sealed-1")

	if _, err := newProviderSecretsUsecase(db, instance).DropSafes(ctx); err == nil {
		t.Error("safes dropped under the instance policy")
	}

	uc := newProviderSecretsUsecase(db, plaintext)

	// A value still sealed: reseal first.
	if _, err := uc.DropSafes(ctx); err == nil {
		t.Fatal("safes dropped while a value was still sealed in them")
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Fatalf("a refused drop deleted the safe: %v", err)
	}

	if _, err := uc.Reseal(ctx); err != nil {
		t.Fatal(err)
	}
	report, err := uc.DropSafes(ctx)
	if err != nil || report.Changed != 1 || report.Failed != 0 {
		t.Fatalf("DropSafes = %+v, %v; want 1 dropped", report, err)
	}

	if err := secret.StartupCheck(secret.PolicyPlaintext, nil, db); err != nil {
		t.Errorf("StartupCheck without keyset after the drop: %v", err)
	}
	msg, err := db.GetProvider(happydns.Identifier{1})
	if err != nil || string(msg.Provider) != `{"apikey":"sealed-1"}` {
		t.Errorf("provider after the drop = %s, %v; want it in clear", msg.Provider, err)
	}
}

// sealedHolder reports one secret sealed in an instance safe, and reseals
// nothing.
type sealedHolder struct{}

func (sealedHolder) InspectSecrets(context.Context) (secret.Counts, error) {
	return secret.Counts{Sealed: map[string]int{secret.KindInstance: 1}}, nil
}

func (sealedHolder) ResealSecrets(context.Context) (secret.ResealReport, error) {
	return secret.ResealReport{}, nil
}

// With several types of objects still holding sealed secrets, the refusal
// names the same one every time: the first in order.
func TestSecretsUsecaseDropSafesNamesTheFirstTypeInOrder(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	_, _, plaintext := newSecretManagers(t, db)

	holders := map[string]usecase.SecretHolder{}
	for _, name := range []string{"type-a", "type-b", "type-c", "type-d", "type-e", "type-f", "type-g", "type-h"} {
		holders[name] = sealedHolder{}
	}
	uc := usecase.NewSecretsUsecase(plaintext, holders, db)

	for range 50 {
		_, err := uc.DropSafes(ctx)
		if err == nil || !strings.Contains(err.Error(), "type-a secret") {
			t.Fatalf("DropSafes = %v, want the refusal to name type-a", err)
		}
	}
}

// On the real storage, a damaged safe record keeps neither the others from
// being dropped nor happyDomain from starting with its keyset. But the drop
// fails, and the keyset check record stays, keeping the keyset required at
// startup, while the record is there.
func TestSecretsUsecaseDropSafesGoesPastADamagedRecord(t *testing.T) {
	ctx := context.Background()
	raw, db := newKVTestStorage(t)
	if err := raw.Put("safe-damaged", "not a safe"); err != nil {
		t.Fatal(err)
	}

	key, instance, plaintext := newSecretManagers(t, db)
	if err := secret.StartupCheck(secret.PolicyInstance, key, db); err != nil {
		t.Fatalf("StartupCheck with a damaged safe record: %v", err)
	}
	seedSecretProvider(t, db, instance, 1, "sealed-1")
	// Restarted: a safe that decodes now vouches for the keyset.
	if err := secret.StartupCheck(secret.PolicyInstance, key, db); err != nil {
		t.Fatalf("StartupCheck once a safe decodes: %v", err)
	}

	uc := newProviderSecretsUsecase(db, plaintext)
	if _, err := uc.Reseal(ctx); err != nil {
		t.Fatal(err)
	}

	report, err := uc.DropSafes(ctx)
	if !errors.Is(err, secret.ErrSafesLeft) || report.Changed != 1 || report.Failed != 1 {
		t.Fatalf("DropSafes = %+v, %v; want 1 dropped, the damaged one reported, ErrSafesLeft", report, err)
	}
	if !strings.Contains(err.Error(), "safe-damaged") {
		t.Errorf("DropSafes error = %q, want it to name the damaged record", err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err == nil {
		t.Error("the readable safe is still there")
	}
	if _, err := db.GetSecretCheck(); err != nil {
		t.Errorf("GetSecretCheck = %v, want the check record kept", err)
	}
	if err := secret.StartupCheck(secret.PolicyPlaintext, key, db); err != nil {
		t.Errorf("StartupCheck with the keyset after the drop: %v", err)
	}
	if err := secret.StartupCheck(secret.PolicyPlaintext, nil, db); err == nil {
		t.Error("StartupCheck without keyset accepted while the damaged record is there")
	}
}

// An object that could not be inspected, a record that does not decode or of
// a type this build no longer knows, may still hold values sealed in the
// instance safes: they are not dropped while it is there.
func TestSecretsUsecaseDropSafesRefusesWhileAnObjectIsUndecodable(t *testing.T) {
	ctx := context.Background()
	raw, db := newKVTestStorage(t)

	key, instance, plaintext := newSecretManagers(t, db)
	if err := key.VerifyCheck(db); err != nil {
		t.Fatal(err)
	}
	seedSecretProvider(t, db, instance, 1, "sealed-1")

	damaged, _ := happydns.NewRandomIdentifier()
	if err := raw.Put("provider-"+damaged.String(), "not a provider"); err != nil {
		t.Fatal(err)
	}

	uc := newProviderSecretsUsecase(db, plaintext)
	if _, err := uc.Reseal(ctx); err != nil {
		t.Fatal(err)
	}

	_, err := uc.DropSafes(ctx)
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("DropSafes = %v, want a ValidationError", err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Errorf("a refused drop deleted the safe: %v", err)
	}
	if _, err := db.GetSecretCheck(); err != nil {
		t.Errorf("GetSecretCheck = %v, want the check record kept", err)
	}
}

// flakySafeStorage fails to read a safe by identifier, like a storage briefly
// down.
type flakySafeStorage struct {
	storage.Storage
}

func (flakySafeStorage) GetSafe(happydns.Identifier) (*happydns.Safe, error) {
	return nil, errors.New("storage unavailable")
}

// A value whose safe could not be read for now is not lost: the safes are not
// dropped while that is the case.
func TestSecretsUsecaseDropSafesRefusesWhileASafeCannotBeRead(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	key, instance, _ := newSecretManagers(t, db)
	if err := key.VerifyCheck(db); err != nil {
		t.Fatal(err)
	}
	seedSecretProvider(t, db, instance, 1, "sealed-1")

	plaintext, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: flakySafeStorage{db}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = newProviderSecretsUsecase(db, plaintext).DropSafes(ctx)
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("DropSafes = %v, want a ValidationError", err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Errorf("a refused drop deleted the safe: %v", err)
	}
	if _, err := db.GetSecretCheck(); err != nil {
		t.Errorf("GetSecretCheck = %v, want the check record kept", err)
	}
}
