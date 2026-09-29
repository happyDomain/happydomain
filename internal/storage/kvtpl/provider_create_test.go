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

func TestCreateProviderWithPresetId(t *testing.T) {
	s := newStorage(t)

	owner, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()
	prvd := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: id, Owner: owner},
		Provider:     stubProviderBody{Field: "value"},
	}

	if err := s.CreateProvider(prvd); err != nil {
		t.Fatalf("CreateProvider with a preset id: %v", err)
	}
	if !prvd.Id.Equals(id) {
		t.Errorf("CreateProvider replaced the preset id %s by %s", id.String(), prvd.Id.String())
	}

	got, err := s.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider(preset id): %v", err)
	}
	if !got.Owner.Equals(owner) {
		t.Errorf("stored owner = %s, want %s", got.Owner.String(), owner.String())
	}

	providers, err := s.ListProviders(&happydns.User{Id: owner})
	if err != nil || len(providers) != 1 {
		t.Errorf("ListProviders = %d, %v; want 1 provider", len(providers), err)
	}
}

func TestCreateProviderWithUsedIdFails(t *testing.T) {
	s := newStorage(t)

	owner, _ := happydns.NewRandomIdentifier()
	other, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()

	first := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: id, Owner: owner},
		Provider:     stubProviderBody{Field: "first"},
	}
	if err := s.CreateProvider(first); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	second := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: append(happydns.Identifier(nil), id...), Owner: other},
		Provider:     stubProviderBody{Field: "second"},
	}
	if err := s.CreateProvider(second); err == nil {
		t.Fatal("CreateProvider with an id already used succeeded, want an error")
	}

	got, err := s.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if !got.Owner.Equals(owner) {
		t.Error("the first provider was overwritten")
	}
	if providers, _ := s.ListProviders(&happydns.User{Id: other}); len(providers) != 0 {
		t.Error("a failed creation left an owner index entry")
	}
}

func TestCreateProviderGeneratesId(t *testing.T) {
	s := newStorage(t)

	owner, _ := happydns.NewRandomIdentifier()
	prvd := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Owner: owner},
		Provider:     stubProviderBody{Field: "value"},
	}

	if err := s.CreateProvider(prvd); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if prvd.Id.IsEmpty() {
		t.Fatal("CreateProvider without an id did not generate one")
	}
	if _, err := s.GetProvider(prvd.Id); err != nil {
		t.Errorf("GetProvider(generated id): %v", err)
	}
}
