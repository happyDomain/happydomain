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
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
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
