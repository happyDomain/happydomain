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
	"context"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// mockProviderGetter implements ProviderGetter for testing.
type mockProviderGetter struct {
	provider *happydns.Provider
	err      error
}

func (m *mockProviderGetter) GetUserProvider(_ context.Context, _ *happydns.User, _ happydns.Identifier) (*happydns.Provider, error) {
	return m.provider, m.err
}

// mockZoneCorrector implements ZoneCorrector for testing.
type mockZoneCorrector struct {
	corrections []*happydns.Correction
	nbDiff      int
	err         error
}

func (m *mockZoneCorrector) ListZoneCorrections(_ context.Context, _ *happydns.Provider, _ *happydns.Domain, _ []happydns.Record) ([]*happydns.Correction, int, error) {
	return m.corrections, m.nbDiff, m.err
}

// mockZoneRetriever implements ZoneRetriever for testing.
type mockZoneRetriever struct {
	records []happydns.Record
	err     error
}

func (m *mockZoneRetriever) RetrieveZone(_ context.Context, _ *happydns.Provider, _ string) ([]happydns.Record, error) {
	return m.records, m.err
}

func newTestListRecordsUsecase() *zoneUC.ListRecordsUsecase {
	return zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
}

func TestZoneCorrectionLister_List_Success(t *testing.T) {
	provider := &happydns.Provider{}

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: provider},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{},
		&mockZoneRetriever{records: nil},
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

	got, nbDiff, err := uc.List(context.Background(), user, domain, zone)
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

func TestZoneCorrectionLister_List_ProviderError(t *testing.T) {
	providerErr := errors.New("provider not found")

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{err: providerErr},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{},
		&mockZoneRetriever{},
	)

	user := &happydns.User{Id: happydns.Identifier([]byte("test-user"))}
	domain := &happydns.Domain{
		ProviderId: happydns.Identifier([]byte("missing-provider")),
		DomainName: "example.com.",
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600},
	}

	_, _, err := uc.List(context.Background(), user, domain, zone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, providerErr) {
		t.Errorf("expected %v, got %v", providerErr, err)
	}
}

func TestZoneCorrectionLister_List_RetrieveZoneError(t *testing.T) {
	retrieveErr := errors.New("zone retrieval failed")

	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: &happydns.Provider{}},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{},
		&mockZoneRetriever{err: retrieveErr},
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

	_, _, err := uc.List(context.Background(), user, domain, zone)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, retrieveErr) {
		t.Errorf("expected %v, got %v", retrieveErr, err)
	}
}

func TestZoneCorrectionLister_List_NoCorrections(t *testing.T) {
	uc := orchestrator.NewZoneCorrectionListerUsecase(
		&mockProviderGetter{provider: &happydns.Provider{}},
		newTestListRecordsUsecase(),
		&mockZoneCorrector{corrections: nil, nbDiff: 0},
		&mockZoneRetriever{records: nil},
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

	got, nbDiff, err := uc.List(context.Background(), user, domain, zone)
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
