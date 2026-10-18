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
	"maps"
	"net/http"
	"slices"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// sealedHolder holds objects sealed in one safe each, and moves them as the
// Manager would, without any storage.
type sealedHolder struct {
	objects map[string]happydns.Identifier
	moved   []string
}

func (h *sealedHolder) InspectSecrets(context.Context) (secret.Counts, error) {
	return secret.Counts{}, nil
}

func (h *sealedHolder) ResealSecrets(context.Context) (secret.ResealReport, error) {
	return secret.ResealReport{}, nil
}

func (h *sealedHolder) OwnerObjects(_ context.Context, owner happydns.Identifier) ([]secret.ObjectSafes, error) {
	var out []secret.ObjectSafes
	for _, id := range slices.Sorted(maps.Keys(h.objects)) {
		out = append(out, secret.ObjectSafes{ObjectType: "thing", ObjectId: id, Label: "Thing " + id, Safes: []happydns.Identifier{h.objects[id]}})
	}
	return out, nil
}

func (h *sealedHolder) MoveOwnerSecrets(_ context.Context, _ happydns.Identifier, from happydns.Identifier, to string) (secret.ResealReport, error) {
	report := secret.ResealReport{ObjectType: "thing"}
	toId, _ := happydns.NewIdentifierFromString(to)
	for id, safe := range h.objects {
		if safe.Equals(from) {
			report.Processed++
			report.Changed++
			h.objects[id] = toId
			h.moved = append(h.moved, id)
		}
	}
	return report, nil
}

func keystoreUsecase(t *testing.T) (*usecase.KeystoreUsecase, *secret.Manager, *sealedHolder) {
	t.Helper()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	holder := &sealedHolder{objects: map[string]happydns.Identifier{}}
	return usecase.NewKeystoreUsecase(m, map[string]usecase.SecretHolder{"thing": holder}), m, holder
}

func httpStatus(err error) int {
	var e happydns.HTTPError
	if errors.As(err, &e) {
		return e.HTTPStatus()
	}
	return 0
}

func TestKeystoreUsecaseLifecycle(t *testing.T) {
	ctx := context.Background()
	uc, _, holder := keystoreUsecase(t)
	owner := happydns.Identifier{0x01}

	if kinds := uc.Kinds(); len(kinds) != 1 || kinds[0].Kind != secrettest.PersonalKind {
		t.Errorf("Kinds = %+v, want the personal test kind only", kinds)
	}

	a, err := uc.Create(ctx, owner, secrettest.PersonalKind, "A", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	b, err := uc.Create(ctx, owner, secrettest.PersonalKind, "B", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(ctx, owner, secrettest.PersonalKind, "A", secrettest.PersonalParams()); httpStatus(err) != http.StatusBadRequest {
		t.Errorf("Create(name taken) = %v, want a 400", err)
	}

	holder.objects["x"] = a.SafeId
	holder.objects["y"] = a.SafeId

	list, err := uc.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	var usedA *usecase.KeystoreUsage
	for i := range list {
		if list[i].Ref == a.Ref {
			usedA = &list[i]
		}
	}
	if usedA == nil || len(usedA.Objects) != 2 || usedA.Objects[0].Label != "Thing x" {
		t.Fatalf("List = %+v, want A used by x and y", list)
	}

	if err := uc.Delete(ctx, owner, a.Ref, false); httpStatus(err) != http.StatusConflict {
		t.Errorf("Delete(used) = %v, want a 409", err)
	}
	if err := uc.Delete(ctx, owner, secret.KindInstance, false); httpStatus(err) != http.StatusForbidden {
		t.Errorf("Delete(instance) = %v, want a 403", err)
	}
	if err := uc.Delete(ctx, owner, "nope", false); httpStatus(err) != http.StatusNotFound {
		t.Errorf("Delete(unknown) = %v, want a 404", err)
	}

	if _, err := uc.Migrate(ctx, owner, a.Ref, a.Ref); httpStatus(err) != http.StatusBadRequest {
		t.Errorf("Migrate(to itself) = %v, want a 400", err)
	}
	if _, err := uc.Migrate(ctx, owner, a.Ref, "nope"); httpStatus(err) != http.StatusNotFound {
		t.Errorf("Migrate(to unknown) = %v, want a 404", err)
	}
	reports, err := uc.Migrate(ctx, owner, a.Ref, b.Ref)
	if err != nil || len(reports) != 1 || reports[0].Changed != 2 {
		t.Fatalf("Migrate = %+v, %v; want 2 objects moved", reports, err)
	}

	if err := uc.SetDefault(ctx, owner, b.Ref); err != nil {
		t.Fatal(err)
	}
	if err := uc.SetDefault(ctx, owner, "nope"); httpStatus(err) != http.StatusNotFound {
		t.Errorf("SetDefault(unknown) = %v, want a 404", err)
	}

	if err := uc.Delete(ctx, owner, a.Ref, false); err != nil {
		t.Errorf("Delete(emptied) = %v", err)
	}
	list, _ = uc.List(ctx, owner)
	for _, ks := range list {
		if ks.Ref == a.Ref {
			t.Error("the deleted keystore is still listed")
		}
		if ks.Ref == b.Ref && (!ks.Default || len(ks.Objects) != 2) {
			t.Errorf("B = %+v, want default and used by both objects", ks)
		}
	}
}
