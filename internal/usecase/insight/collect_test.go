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

package insight_test

import (
	"maps"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/insight"
	"git.happydns.org/happyDomain/model"
)

func TestCollect_CountsKeystoresByKind(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	for _, s := range []struct{ kind, name string }{
		{"instance", ""},
		{"instance", ""},
		{"passphrase", "mine"},
		{"vault", "work"},
		{"passphrase", "other"},
	} {
		id, _ := happydns.NewRandomIdentifier()
		owner, _ := happydns.NewRandomIdentifier()
		if err := store.CreateSafe(&happydns.Safe{Id: id, Owner: owner, Kind: s.kind, Name: s.name}); err != nil {
			t.Fatalf("CreateSafe() returned error: %v", err)
		}
	}

	data, err := insight.Collect(&happydns.Options{}, store, "test", happydns.VersionResponse{}, nil, "go")
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	want := map[string]int{"instance": 2, "passphrase": 2, "vault": 1}
	if !maps.Equal(data.Database.Keystores, want) {
		t.Errorf("Keystores = %v, want %v", data.Database.Keystores, want)
	}
}

func TestCollect_NoKeystore(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	data, err := insight.Collect(&happydns.Options{}, store, "test", happydns.VersionResponse{}, nil, "go")
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if data.Database.Keystores == nil || len(data.Database.Keystores) != 0 {
		t.Errorf("Keystores = %#v, want an empty map", data.Database.Keystores)
	}
}
