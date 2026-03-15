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

package orchestrator_test

import (
	"errors"
	"testing"

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	"git.happydns.org/happyDomain/model"
)

// mockProviderGetter implements ProviderGetter for testing.
type mockProviderGetter struct {
	provider *happydns.Provider
	err      error
}

func (m *mockProviderGetter) GetUserProvider(_ *happydns.User, _ happydns.Identifier) (*happydns.Provider, error) {
	return m.provider, m.err
}

// mockZoneCorrector implements ZoneCorrector for testing.
type mockZoneCorrector struct {
	corrections []*happydns.Correction
	nbDiff      int
	err         error
}

func (m *mockZoneCorrector) ListZoneCorrections(_ *happydns.Provider, _ *happydns.Domain, _ []happydns.Record) ([]*happydns.Correction, int, error) {
	return m.corrections, m.nbDiff, m.err
}

func newTestListRecordsUsecase() *zoneUC.ListRecordsUsecase {
	return zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
}

func TestZoneCorrectionLister_List_Success(t *testing.T) {
	provider := &happydns.Provider{}
	corrections := []*happydns.Correction{
		{Msg: "add A record", Kind: happydns.CorrectionKindAddition},
		{Msg: "delete MX record", Kind: happydns.CorrectionKindDeletion},
	}

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: provider},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{corrections: corrections, nbDiff: 2},
	)

	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}
	domain := &happydns.Domain{
		Id:         happydns.Identifier([]byte("test-domain")),
		ProviderId: happydns.Identifier([]byte("test-provider")),
		DomainName: "example.com.",
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}

	got, nbDiff, err := uc.List(user, domain, zone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nbDiff != 2 {
		t.Errorf("expected nbDiff=2, got %d", nbDiff)
	}
	if len(got) != len(corrections) {
		t.Errorf("expected %d corrections, got %d", len(corrections), len(got))
	}
	for i, c := range got {
		if c.Msg != corrections[i].Msg {
			t.Errorf("correction[%d].Msg = %q, want %q", i, c.Msg, corrections[i].Msg)
		}
		if c.Kind != corrections[i].Kind {
			t.Errorf("correction[%d].Kind = %v, want %v", i, c.Kind, corrections[i].Kind)
		}
	}
}

func TestZoneCorrectionLister_List_ProviderError(t *testing.T) {
	providerErr := errors.New("provider not found")

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{err: providerErr},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{},
	)

	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}
	domain := &happydns.Domain{
		ProviderId: happydns.Identifier([]byte("missing-provider")),
		DomainName: "example.com.",
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
	}

	_, _, err := uc.List(user, domain, zone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, providerErr) {
		t.Errorf("expected %v, got %v", providerErr, err)
	}
}

func TestZoneCorrectionLister_List_ZoneCorrectorError(t *testing.T) {
	correctorErr := errors.New("zone correction failed")

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: &happydns.Provider{}},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{err: correctorErr},
	)

	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}
	domain := &happydns.Domain{
		ProviderId: happydns.Identifier([]byte("test-provider")),
		DomainName: "example.com.",
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}

	_, _, err := uc.List(user, domain, zone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, correctorErr) {
		t.Errorf("expected %v, got %v", correctorErr, err)
	}
}

func TestZoneCorrectionLister_List_NoCorrections(t *testing.T) {
	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: &happydns.Provider{}},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{corrections: nil, nbDiff: 0},
	)

	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}
	domain := &happydns.Domain{
		ProviderId: happydns.Identifier([]byte("test-provider")),
		DomainName: "example.com.",
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}

	got, nbDiff, err := uc.List(user, domain, zone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nbDiff != 0 {
		t.Errorf("expected nbDiff=0, got %d", nbDiff)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 corrections, got %d", len(got))
	}
}
