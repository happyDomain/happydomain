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
	"encoding/json"
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func storedStub(t *testing.T, msg *happydns.ProviderMessage) string {
	t.Helper()
	var body stubProviderBody
	if err := json.Unmarshal(msg.Provider, &body); err != nil {
		t.Fatal(err)
	}
	return body.Field
}

func TestReplaceProvider(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()
	if err := s.UpdateProvider(&happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "stub", Id: id, Owner: owner},
		Provider:     stubProviderBody{Field: "before"},
	}); err != nil {
		t.Fatal(err)
	}

	err := s.ReplaceProvider(id, func(msg *happydns.ProviderMessage) (*happydns.Provider, error) {
		if storedStub(t, msg) != "before" {
			t.Errorf("update got %s, want the stored provider", msg.Provider)
		}
		return &happydns.Provider{ProviderMeta: msg.ProviderMeta, Provider: stubProviderBody{Field: "after"}}, nil
	})
	if err != nil {
		t.Fatalf("ReplaceProvider: %v", err)
	}
	got, _ := s.GetProvider(id)
	if storedStub(t, got) != "after" {
		t.Errorf("stored = %s, want the replacement", got.Provider)
	}
	if list, _ := s.ListProviders(&happydns.User{Id: owner}); len(list) != 1 {
		t.Errorf("owner lists %d providers, want 1", len(list))
	}

	// Neither the identifier nor the owner can change: the owner index
	// would no longer match.
	other, _ := happydns.NewRandomIdentifier()
	for _, change := range []func(*happydns.ProviderMeta){
		func(m *happydns.ProviderMeta) { m.Owner = other },
		func(m *happydns.ProviderMeta) { m.Id = other },
	} {
		err := s.ReplaceProvider(id, func(msg *happydns.ProviderMessage) (*happydns.Provider, error) {
			meta := msg.ProviderMeta
			change(&meta)
			return &happydns.Provider{ProviderMeta: meta, Provider: stubProviderBody{Field: "moved"}}, nil
		})
		if err == nil {
			t.Error("ReplaceProvider moved a provider")
		}
	}

	// Deleted meanwhile: not brought back.
	err = s.ReplaceProvider(id, func(msg *happydns.ProviderMessage) (*happydns.Provider, error) {
		if err := s.DeleteProvider(id); err != nil {
			t.Fatal(err)
		}
		return &happydns.Provider{ProviderMeta: msg.ProviderMeta, Provider: stubProviderBody{Field: "back"}}, nil
	})
	if !errors.Is(err, happydns.ErrProviderNotFound) {
		t.Errorf("ReplaceProvider over a deletion = %v, want ErrProviderNotFound", err)
	}
	if _, err := s.GetProvider(id); !errors.Is(err, happydns.ErrProviderNotFound) {
		t.Errorf("a deleted provider was brought back: %v", err)
	}
}
