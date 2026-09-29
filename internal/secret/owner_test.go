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

package secret

import (
	"context"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

// knownOwners answers for the users that exist.
type knownOwners map[string]bool

func (k knownOwners) GetUser(id happydns.Identifier) (*happydns.User, error) {
	if k[id.String()] {
		return &happydns.User{Id: id}, nil
	}
	return nil, happydns.ErrUserNotFound
}

type brokenOwners struct{}

func (brokenOwners) GetUser(happydns.Identifier) (*happydns.User, error) {
	return nil, errors.New("storage unavailable")
}

// No safe, nor key, is made for a user that does not exist: a provider left
// by a deleted user and resealed before tidy runs must not get one.
func TestNoSafeCreatedForAnUnknownOwner(t *testing.T) {
	store := secrettest.NewSafes()
	sc := objectContext()
	m, err := NewManager(Config{
		Policy:      PolicyInstance,
		InstanceKey: testInstanceKey(t),
		Safes:       store,
		Owners:      knownOwners{sc.Owner.String(): true},
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = sealOne(t, m, sc, "v")

	gone := sc
	gone.Owner = happydns.Identifier{0x09}
	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := m.SealObject(context.Background(), gone, obj); !errors.Is(err, ErrUnknownOwner) {
		t.Errorf("SealObject for a deleted owner = %v, want ErrUnknownOwner", err)
	}
	if !obj.ApiKey.IsClear() {
		t.Error("a refused seal changed the object")
	}

	// The instance itself is not a user.
	inst := sc
	inst.Owner = InstanceOwner()
	_ = sealOne(t, m, inst, "v")

	if store.Creates() != 2 {
		t.Errorf("%d safes created, want one for the user and one for the instance", store.Creates())
	}
}

func TestNoSafeCreatedWhenOwnersCannotBeChecked(t *testing.T) {
	store := secrettest.NewSafes()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: store, Owners: brokenOwners{}})

	obj := &managedObject{ApiKey: happydns.NewSecret("v")}
	if err := m.SealObject(context.Background(), objectContext(), obj); err == nil {
		t.Error("sealed although the owner could not be checked")
	}
	if store.Creates() != 0 {
		t.Errorf("%d safes created, want none", store.Creates())
	}
}
