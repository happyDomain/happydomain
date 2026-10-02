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

func TestSecretsUsecase(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})

	seedSecretProvider(t, db, plaintext, 1, "legacy-1")
	seedSecretProvider(t, db, plaintext, 2, "legacy-2")
	seedSecretProvider(t, db, instance, 3, "sealed-3")

	uc := usecase.NewSecretsUsecase(instance, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, instance),
	}, db)

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

	if report, err := uc.Rewrap(ctx); err != nil || report.Processed != 1 || report.Changed != 0 {
		t.Errorf("Rewrap = %+v, %v; want nothing to do on a single key", report, err)
	}
}

// Going back to clear: once nothing is sealed any more, the safes and the
// check record are deleted, and the keyset is no longer needed at startup.
func TestSecretsUsecaseDropSafes(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})
	if err := secret.StartupCheck(secret.PolicyInstance, key, db); err != nil {
		t.Fatal(err)
	}

	seedSecretProvider(t, db, instance, 1, "sealed-1")
	seedSecretProvider(t, db, instance, 2, "sealed-2")

	newUsecase := func(m *secret.Manager) *usecase.SecretsUsecase {
		return usecase.NewSecretsUsecase(m, map[string]usecase.SecretHolder{
			providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, m),
		}, db)
	}

	// Still under the instance policy.
	if _, err := newUsecase(instance).DropSafes(ctx); err == nil {
		t.Fatal("DropSafes under the instance policy succeeded")
	}

	// Back to plaintext, but the credentials are still sealed.
	uc := newUsecase(plaintext)
	if _, err := uc.DropSafes(ctx); err == nil {
		t.Fatal("DropSafes with sealed credentials left succeeded")
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Fatalf("the safe was deleted by a refused DropSafes: %v", err)
	}

	if _, err := uc.Reseal(ctx); err != nil {
		t.Fatal(err)
	}
	n, err := uc.DropSafes(ctx)
	if err != nil || n != 1 {
		t.Fatalf("DropSafes = %d, %v; want the safe dropped", n, err)
	}

	if err := secret.StartupCheck(secret.PolicyPlaintext, nil, db); err != nil {
		t.Errorf("StartupCheck without the keyset: %v", err)
	}
	// A new keyset is accepted later on.
	other, _ := secret.GenerateInstanceKeyset()
	otherKey, _ := secret.NewInstanceKey(other)
	if err := secret.StartupCheck(secret.PolicyInstance, otherKey, db); err != nil {
		t.Errorf("StartupCheck with a new keyset: %v", err)
	}

	// And the credentials are there, in clear.
	status, err := uc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c := status.Objects[providerUC.SecretObjectType]; c.Clear != 2 {
		t.Errorf("counts after dropping = %+v, want 2 in clear", c)
	}
}

// A provider that cannot be looked at may hold sealed credentials: the safes
// are kept.
func TestSecretsUsecaseDropSafesRefusedWithUndecodable(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})

	seedSecretProvider(t, db, instance, 1, "sealed-1")
	msg, err := db.GetProvider(happydns.Identifier{1})
	if err != nil {
		t.Fatal(err)
	}
	// Its type left this build.
	msg.Type = "RemovedProvider"
	if err := db.UpdateProvider(&happydns.Provider{ProviderMeta: msg.ProviderMeta, Provider: &removedBody{Raw: msg.Provider}}); err != nil {
		t.Fatal(err)
	}

	uc := usecase.NewSecretsUsecase(plaintext, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, plaintext),
	}, db)
	if _, err := uc.DropSafes(ctx); err == nil || !strings.Contains(err.Error(), "could not be looked at") {
		t.Errorf("DropSafes = %v, want it refused for the undecodable provider", err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Errorf("the safe was deleted: %v", err)
	}
}

// A safe record that does not decode may hold the key of sealed secrets: the
// readable safes go, but the check record stays, and so does the keyset.
func TestSecretsUsecaseDropSafesKeepsCheckWithDamagedSafe(t *testing.T) {
	ctx := context.Background()
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})
	if err := secret.StartupCheck(secret.PolicyInstance, key, db); err != nil {
		t.Fatal(err)
	}
	seedSecretProvider(t, db, instance, 1, "sealed-1")
	if err := kvdb.Put("safe-damaged", "not a safe"); err != nil {
		t.Fatal(err)
	}

	uc := usecase.NewSecretsUsecase(plaintext, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, plaintext),
	}, db)
	if _, err := uc.Reseal(ctx); err != nil {
		t.Fatal(err)
	}

	n, err := uc.DropSafes(ctx)
	if err == nil || n != 1 {
		t.Fatalf("DropSafes = %d, %v; want the readable safe dropped and an error for the damaged one", n, err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err == nil {
		t.Error("the readable safe was kept")
	}
	if _, err := db.GetSecretCheck(); err != nil {
		t.Errorf("the check record was deleted with a damaged safe left: %v", err)
	}
}

// Secrets sealed under a keyset that is gone do not open: dropping the safes
// would lose them for good, so the safes are kept.
func TestSecretsUsecaseDropSafesRefusedWithUnreadable(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	seedSecretProvider(t, db, instance, 1, "sealed-1")

	other, _ := secret.GenerateInstanceKeyset()
	otherKey, _ := secret.NewInstanceKey(other)
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: otherKey, Safes: db})
	uc := usecase.NewSecretsUsecase(plaintext, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, plaintext),
	}, db)
	_, _ = uc.Reseal(ctx)

	if _, err := uc.DropSafes(ctx); err == nil || !strings.Contains(err.Error(), "still sealed") {
		t.Errorf("DropSafes = %v, want it refused for the unreadable credential", err)
	}
	if _, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil {
		t.Errorf("the safe was deleted: %v", err)
	}
}

// holderStub counts one clear secret and reseals nothing, or fails with err.
type holderStub struct{ err error }

func (h holderStub) InspectSecrets(context.Context) (secret.Counts, error) {
	return secret.Counts{Clear: 1}, h.err
}

func (h holderStub) ResealSecrets(context.Context) (secret.ResealReport, error) {
	return secret.ResealReport{ObjectType: "a"}, h.err
}

// A type of object failing stops the walk, names that type, and keeps the
// reports of the types already resealed.
func TestSecretsUsecaseHolderFailure(t *testing.T) {
	ctx := context.Background()
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})

	boom := errors.New("storage unavailable")
	uc := usecase.NewSecretsUsecase(m, map[string]usecase.SecretHolder{
		"a": holderStub{},
		"b": holderStub{err: boom},
	}, db)

	if status, err := uc.Status(ctx); !errors.Is(err, boom) || status != nil || !strings.HasPrefix(err.Error(), "b: ") {
		t.Errorf("Status = %+v, %v; want the failure of b", status, err)
	}
	reports, err := uc.Reseal(ctx)
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "b: ") {
		t.Errorf("Reseal error = %v, want the failure of b", err)
	}
	if len(reports) != 1 || reports[0].ObjectType != "a" {
		t.Errorf("Reseal reports = %+v, want the one of a, done before b failed", reports)
	}
}

// removedBody stores a body as it was, under a type this build does not know.
type removedBody struct {
	Raw json.RawMessage
}

func (b *removedBody) MarshalJSON() ([]byte, error) { return b.Raw, nil }

func (*removedBody) InstantiateProvider() (happydns.ProviderActuator, error) { return nil, nil }

// A safe record that does not decode keeps neither the key report nor the
// rewrap of the other safes from going through.
func TestSecretsUsecaseRewrapGoesPastADamagedSafe(t *testing.T) {
	ctx := context.Background()
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	seedSecretProvider(t, db, instance, 1, "sealed-1")
	if err := kvdb.Put("safe-damaged", "not a safe"); err != nil {
		t.Fatal(err)
	}

	rotated, _ := secret.RotateInstanceKeyset(h)
	newKey, _ := secret.NewInstanceKey(rotated)
	m, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: newKey, Safes: db})
	uc := usecase.NewSecretsUsecase(m, map[string]usecase.SecretHolder{
		providerUC.SecretObjectType: providerUC.NewService(db, acceptAll{}, nil, m),
	}, db)

	status, err := uc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	unreadable := 0
	for _, k := range status.InstanceKeys {
		if k.Status == "UNREADABLE" {
			unreadable = k.Safes
		}
	}
	if unreadable != 1 {
		t.Errorf("instance keys = %+v, want the damaged safe counted UNREADABLE", status.InstanceKeys)
	}

	report, err := uc.Rewrap(ctx)
	if err != nil || report.Changed != 1 || report.Failed != 1 {
		t.Errorf("Rewrap = %+v, %v; want the readable safe rewrapped, the damaged one failed", report, err)
	}
}
