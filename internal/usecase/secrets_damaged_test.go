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
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// damagedSafeSetup stores two providers sealed in the instance safe of user
// 0x01, then damages that safe. It returns the safe as it was.
func damagedSafeSetup(t *testing.T) (raw *inmemory.InMemoryStorage, db storage.Storage, key *secret.InstanceKey, instance *secret.Manager, safe *happydns.Safe) {
	t.Helper()
	raw, db = newKVTestStorage(t)
	key, instance, _ = newSecretManagers(t, db)
	if err := key.VerifyCheck(db); err != nil {
		t.Fatal(err)
	}
	seedSecretProvider(t, db, instance, 1, "sealed-1")
	seedSecretProvider(t, db, instance, 2, "sealed-2")

	safe, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Put("safe-"+safe.Id.String(), "not a safe"); err != nil {
		t.Fatal(err)
	}
	return raw, db, key, instance, safe
}

// The status names each damaged safe, its owner when the index tells it,
// and how many sealed values depend on it: what forgetting it would lose.
func TestSecretsUsecaseStatusNamesDamagedSafes(t *testing.T) {
	_, db, _, instance, safe := damagedSafeSetup(t)

	status, err := newProviderSecretsUsecase(db, instance).Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(status.DamagedSafes) != 1 {
		t.Fatalf("DamagedSafes = %+v, want the damaged safe", status.DamagedSafes)
	}
	d := status.DamagedSafes[0]
	if !d.Id.Equals(safe.Id) || !d.Owner.Equals(happydns.Identifier{0x01}) || d.Kind != secret.KindInstance || d.Values != 2 {
		t.Errorf("DamagedSafes[0] = %+v, want safe %s of 0x01, instance, 2 values", d, safe.Id.String())
	}
}

// Forgetting a damaged safe gives up what it sealed: the values it held count
// as unreadable, like any value that no longer opens, and no longer keep the
// objects from being inspected, nor the keyset from being removed.
func TestSecretsUsecaseForgetSafe(t *testing.T) {
	ctx := context.Background()
	_, db, key, instance, safe := damagedSafeSetup(t)
	uc := newProviderSecretsUsecase(db, instance)

	other := secret.SecretContext{Owner: happydns.Identifier{0x02}, ObjectType: "provider", ObjectId: "AQ", Field: "k"}
	if _, err := instance.SealValue(ctx, other, "v"); err != nil {
		t.Fatal(err)
	}
	sound, err := db.GetSafeByOwner(other.Owner, secret.KindInstance)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.ForgetSafe(ctx, sound.Id); !errors.Is(err, happydns.ErrSafeNotDamaged) {
		t.Errorf("ForgetSafe on a sound safe = %v, want ErrSafeNotDamaged", err)
	}
	missing, _ := happydns.NewRandomIdentifier()
	if err := uc.ForgetSafe(ctx, missing); !errors.Is(err, happydns.ErrSafeNotFound) {
		t.Errorf("ForgetSafe on no safe = %v, want ErrSafeNotFound", err)
	}

	if err := uc.ForgetSafe(ctx, safe.Id); err != nil {
		t.Fatalf("ForgetSafe: %v", err)
	}

	status, err := uc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.DamagedSafes) != 0 {
		t.Errorf("DamagedSafes after forgetting = %+v", status.DamagedSafes)
	}
	if c := status.Objects[providerUC.SecretObjectType]; c.Unreadable != 2 || c.Undecodable != 0 {
		t.Errorf("provider counts after forgetting = %+v, want 2 unreadable, none undecodable", c)
	}

	// The owner gets a new safe for what they enter again.
	seedSecretProvider(t, db, instance, 3, "sealed-3")
	if got, err := db.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance); err != nil || got.Id.Equals(safe.Id) {
		t.Errorf("safe of the owner after forgetting = %+v, %v; want a new one", got, err)
	}

	// Nothing keeps going back to clear from completing any more.
	var plaintext *secret.Manager
	plaintext, err = secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})
	if err != nil {
		t.Fatal(err)
	}
	back := newProviderSecretsUsecase(db, plaintext)
	if _, err := back.Reseal(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := back.DropSafes(ctx); err != nil {
		t.Fatalf("DropSafes after forgetting the damaged safe: %v", err)
	}
	if err := secret.StartupCheck(secret.PolicyPlaintext, nil, db); err != nil {
		t.Errorf("StartupCheck without keyset after the drop: %v", err)
	}
}

// A backup holding the damaged safe repairs it: what it sealed opens again.
// The safe of the backup is only put back when it opens with the keyset,
// bound to its identifier and owner, and only in place of a damaged record.
func TestSecretsUsecaseRepairSafe(t *testing.T) {
	ctx := context.Background()
	_, db, _, instance, safe := damagedSafeSetup(t)
	uc := newProviderSecretsUsecase(db, instance)

	damagedStill := func(t *testing.T) {
		t.Helper()
		if damaged, _ := db.ListDamagedSafes(); len(damaged) != 1 {
			t.Errorf("a refused repair changed the record: %+v", damaged)
		}
	}

	if err := uc.RepairSafe(ctx, safe.Id, &happydns.Backup{}); err == nil {
		t.Error("RepairSafe from a backup without the safe succeeded")
	}
	damagedStill(t)

	// Another owner: the wrapped key is bound to the owner, it does not open.
	stolen := *safe
	stolen.Owner = happydns.Identifier{0x02}
	if err := uc.RepairSafe(ctx, safe.Id, &happydns.Backup{Safes: []*happydns.Safe{&stolen}}); err == nil {
		t.Error("RepairSafe with a safe of another owner succeeded")
	}
	damagedStill(t)

	// Wrapped for another safe, under another keyset: it never opens here.
	foreign := &happydns.Safe{}
	{
		fdb, _ := inmemory.Instantiate()
		_, fm, _ := newSecretManagers(t, fdb)
		if _, err := fm.SealValue(ctx, secret.SecretContext{Owner: happydns.Identifier{0x01}, ObjectType: "provider", ObjectId: "AQ", Field: "k"}, "v"); err != nil {
			t.Fatal(err)
		}
		fs, err := fdb.GetSafeByOwner(happydns.Identifier{0x01}, secret.KindInstance)
		if err != nil {
			t.Fatal(err)
		}
		*foreign = *fs
		foreign.Id = safe.Id
	}
	if err := uc.RepairSafe(ctx, safe.Id, &happydns.Backup{Safes: []*happydns.Safe{foreign}}); err == nil {
		t.Error("RepairSafe with a safe wrapped under another keyset succeeded")
	}
	damagedStill(t)

	if err := uc.RepairSafe(ctx, safe.Id, &happydns.Backup{Safes: []*happydns.Safe{safe}}); err != nil {
		t.Fatalf("RepairSafe: %v", err)
	}
	status, err := uc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.DamagedSafes) != 0 {
		t.Errorf("DamagedSafes after the repair = %+v", status.DamagedSafes)
	}
	if c := status.Objects[providerUC.SecretObjectType]; c.Sealed[secret.KindInstance] != 2 || c.Unreadable != 0 || c.Undecodable != 0 {
		t.Errorf("provider counts after the repair = %+v, want both values opening again", c)
	}

	// Only a damaged record is repaired.
	if err := uc.RepairSafe(ctx, safe.Id, &happydns.Backup{Safes: []*happydns.Safe{safe}}); !errors.Is(err, happydns.ErrSafeNotDamaged) {
		t.Errorf("RepairSafe over a sound safe = %v, want ErrSafeNotDamaged", err)
	}
}
