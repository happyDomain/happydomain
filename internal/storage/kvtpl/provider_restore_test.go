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

package database_test

import (
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

type stubProviderBody struct {
	Field string `json:"field"`
}

func (stubProviderBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

// TestUpdateProviderUpsertsIntoEmptyStore guards the backup restore path: a
// backup is loaded into a fresh database through UpdateProvider, so the primary
// record does not exist yet. A missing old record must not be treated as an
// error (mirrors UpdateDomain/UpdateZone, which are already upserts).
func TestUpdateProviderUpsertsIntoEmptyStore(t *testing.T) {
	s := newStorage(t)

	owner, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()
	prvd := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: id, Owner: owner},
		Provider:     stubProviderBody{Field: "value"},
	}

	if err := s.UpdateProvider(prvd); err != nil {
		t.Fatalf("UpdateProvider into empty store returned error: %v", err)
	}

	got, err := s.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider after upsert: %v", err)
	}
	if !got.Id.Equals(id) {
		t.Errorf("restored provider id = %s, want %s", got.Id.String(), id.String())
	}
	if !got.Owner.Equals(owner) {
		t.Errorf("restored provider owner = %s, want %s", got.Owner.String(), owner.String())
	}

	// The owner index must also be populated so the provider is listable.
	providers, err := s.ListProviders(&happydns.User{Id: owner})
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("ListProviders returned %d providers, want 1", len(providers))
	}
}
