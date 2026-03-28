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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/miekg/dns"

	providerReg "git.happydns.org/happyDomain/internal/provider"
	domainlogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	"git.happydns.org/happyDomain/internal/usecase/orchestrator"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"

	// Import AXFRDDNS provider to register its capabilities.
	_ "git.happydns.org/happyDomain/providers"
)

// mockDomainUpdater implements DomainUpdater for testing.
type mockDomainUpdater struct {
	domain *happydns.Domain
	err    error
}

func (m *mockDomainUpdater) Update(_ happydns.Identifier, _ *happydns.User, updateFn func(*happydns.Domain)) error {
	if m.err != nil {
		return m.err
	}
	if m.domain != nil {
		updateFn(m.domain)
	}
	return nil
}

// inMemoryZoneStorage implements ZoneStorage for testing.
type inMemoryZoneStorage struct {
	zones map[string]*happydns.Zone
}

func newInMemoryZoneStorage() *inMemoryZoneStorage {
	return &inMemoryZoneStorage{zones: map[string]*happydns.Zone{}}
}

func (s *inMemoryZoneStorage) ListAllZones() (happydns.Iterator[happydns.ZoneMessage], error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *inMemoryZoneStorage) GetZoneMeta(zoneid happydns.Identifier) (*happydns.ZoneMeta, error) {
	z, ok := s.zones[zoneid.String()]
	if !ok {
		return nil, fmt.Errorf("zone not found: %s", zoneid)
	}
	return &z.ZoneMeta, nil
}

func (s *inMemoryZoneStorage) GetZone(zoneid happydns.Identifier) (*happydns.ZoneMessage, error) {
	z, ok := s.zones[zoneid.String()]
	if !ok {
		return nil, fmt.Errorf("zone not found: %s", zoneid)
	}

	// Convert Zone to ZoneMessage by marshaling services.
	msg := &happydns.ZoneMessage{
		ZoneMeta: z.ZoneMeta,
		Services: map[happydns.Subdomain][]*happydns.ServiceMessage{},
	}

	for subdn, svcs := range z.Services {
		for _, svc := range svcs {
			body, err := json.Marshal(svc.Service)
			if err != nil {
				return nil, err
			}
			msg.Services[subdn] = append(msg.Services[subdn], &happydns.ServiceMessage{
				ServiceMeta: svc.ServiceMeta,
				Service:     body,
			})
		}
	}

	return msg, nil
}

func (s *inMemoryZoneStorage) CreateZone(zone *happydns.Zone) error {
	if zone.Id == nil {
		zone.Id = happydns.Identifier([]byte(fmt.Sprintf("zone-%d", len(s.zones))))
	}
	s.zones[zone.Id.String()] = zone
	return nil
}

func (s *inMemoryZoneStorage) UpdateZone(zone *happydns.Zone) error {
	s.zones[zone.Id.String()] = zone
	return nil
}

func (s *inMemoryZoneStorage) DeleteZone(zoneid happydns.Identifier) error {
	delete(s.zones, zoneid.String())
	return nil
}

func (s *inMemoryZoneStorage) ClearZones() error {
	s.zones = map[string]*happydns.Zone{}
	return nil
}

// mockZoneRetrieverFailOnNth returns records until the Nth call, then fails.
type mockZoneRetrieverFailOnNth struct {
	records   []happydns.Record
	failOnNth int
	failErr   error
	calls     int
}

func (m *mockZoneRetrieverFailOnNth) RetrieveZone(_ context.Context, _ *happydns.Provider, _ string) ([]happydns.Record, error) {
	m.calls++
	if m.calls >= m.failOnNth {
		return nil, m.failErr
	}
	return m.records, nil
}

// testZoneRetriever is an interface matching orchestrator.ZoneRetriever.
type testZoneRetriever interface {
	RetrieveZone(ctx context.Context, provider *happydns.Provider, name string) ([]happydns.Record, error)
}

// buildTestApplier creates a ZoneCorrectionApplierUsecase with the given overrides.
func buildTestApplier(
	providerGetter *mockProviderGetter,
	zoneCorrector *mockZoneCorrector,
	retriever testZoneRetriever,
	domainUpdater *mockDomainUpdater,
	storage *inMemoryZoneStorage,
) *orchestrator.ZoneCorrectionApplierUsecase {
	listRecords := zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase())
	lister := orchestrator.NewZoneCorrectionListerUsecase(
		providerGetter,
		listRecords,
		zoneCorrector,
		retriever,
	)

	zoneGetter := zoneUC.NewGetZoneUsecase(storage)
	zoneCreator := zoneUC.NewCreateZoneUsecase(storage)
	zoneUpdater := zoneUC.NewUpdateZoneUsease(storage, zoneGetter)

	return orchestrator.NewZoneCorrectionApplierUsecase(
		domainlogUC.NoopDomainLogAppender{},
		domainUpdater,
		lister,
		zoneCreator,
		zoneGetter,
		retriever,
		zoneUpdater,
	)
}

func TestApply_NoRefetch_WhenProviderLacksCapability(t *testing.T) {
	// Provider without manages-soa-serial capability.
	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type: "NoSuchProvider",
		},
	}

	storage := newInMemoryZoneStorage()

	wipZoneID := happydns.Identifier([]byte("wip-zone"))
	wipZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			Id:         wipZoneID,
			DefaultTTL: 3600,
		},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}
	storage.zones[wipZoneID.String()] = wipZone

	domain := &happydns.Domain{
		Id:          happydns.Identifier([]byte("test-domain")),
		ProviderId:  happydns.Identifier([]byte("test-provider")),
		DomainName:  "example.com.",
		ZoneHistory: []happydns.Identifier{wipZoneID},
	}

	retriever := &mockZoneRetriever{records: nil}

	uc := buildTestApplier(
		&mockProviderGetter{provider: provider},
		&mockZoneCorrector{corrections: nil, nbDiff: 0},
		retriever,
		&mockDomainUpdater{domain: domain},
		storage,
	)

	snapshot, err := uc.Apply(
		context.Background(),
		&happydns.User{Id: happydns.Identifier([]byte("test-user"))},
		domain,
		wipZone,
		&happydns.ApplyZoneForm{
			WantedCorrections: nil,
			CommitMsg:         "test deploy",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}
}

func TestApply_Refetch_WhenProviderManagesSOASerial(t *testing.T) {
	// Use the DDNSServer type which has manages-soa-serial capability.
	// First verify it's registered.
	creators := providerReg.GetProviders()
	_, hasDDNS := creators["DDNSServer"]
	if !hasDDNS {
		t.Skip("DDNSServer provider not registered")
	}

	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type: "DDNSServer",
		},
	}

	storage := newInMemoryZoneStorage()

	// Create WIP zone with an Origin service containing old SOA serial.
	wipZoneID := happydns.Identifier([]byte("wip-zone"))
	oldSerial := uint32(2024010100)
	wipZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			Id:         wipZoneID,
			DefaultTTL: 3600,
		},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{
						Id:   happydns.Identifier([]byte("origin-svc")),
						Type: "abstract.Origin",
					},
					Service: &abstract.Origin{
						SOA: &dns.SOA{
							Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 3600},
							Ns:     "ns1.example.com.",
							Mbox:   "admin.example.com.",
							Serial: oldSerial,
						},
						NameServers: []*dns.NS{
							{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 3600}, Ns: "ns1.example.com."},
						},
					},
				},
			},
		},
	}
	storage.zones[wipZoneID.String()] = wipZone

	domain := &happydns.Domain{
		Id:          happydns.Identifier([]byte("test-domain")),
		ProviderId:  happydns.Identifier([]byte("test-provider")),
		DomainName:  "example.com.",
		ZoneHistory: []happydns.Identifier{wipZoneID},
	}

	// The re-fetched records contain a new SOA serial.
	newSerial := uint32(2024010101)
	refetchedRecords := []happydns.Record{
		&dns.SOA{
			Hdr:    dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 3600},
			Ns:     "ns1.example.com.",
			Mbox:   "admin.example.com.",
			Serial: newSerial,
		},
		&dns.NS{
			Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeNS, Class: dns.ClassINET, Ttl: 3600},
			Ns:  "ns1.example.com.",
		},
	}

	retriever := &mockZoneRetriever{records: refetchedRecords}

	uc := buildTestApplier(
		&mockProviderGetter{provider: provider},
		&mockZoneCorrector{corrections: nil, nbDiff: 0},
		retriever,
		&mockDomainUpdater{domain: domain},
		storage,
	)

	snapshot, err := uc.Apply(
		context.Background(),
		&happydns.User{Id: happydns.Identifier([]byte("test-user"))},
		domain,
		wipZone,
		&happydns.ApplyZoneForm{
			WantedCorrections: nil,
			CommitMsg:         "test deploy with SOA",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify snapshot has the new serial.
	snapshotSerial := getOriginSOASerial(t, snapshot)
	if snapshotSerial != newSerial {
		t.Errorf("snapshot SOA serial: got %d, want %d", snapshotSerial, newSerial)
	}

	// Verify WIP zone was patched with new serial.
	updatedWIP := storage.zones[wipZoneID.String()]
	wipSerial := getOriginSOASerial(t, updatedWIP)
	if wipSerial != newSerial {
		t.Errorf("WIP zone SOA serial: got %d, want %d", wipSerial, newSerial)
	}
}

func TestApply_RefetchFails_FallsBackToTargetRecords(t *testing.T) {
	creators := providerReg.GetProviders()
	_, hasDDNS := creators["DDNSServer"]
	if !hasDDNS {
		t.Skip("DDNSServer provider not registered")
	}

	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type: "DDNSServer",
		},
	}

	storage := newInMemoryZoneStorage()

	wipZoneID := happydns.Identifier([]byte("wip-zone"))
	wipZone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{
			Id:         wipZoneID,
			DefaultTTL: 3600,
		},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}
	storage.zones[wipZoneID.String()] = wipZone

	domain := &happydns.Domain{
		Id:          happydns.Identifier([]byte("test-domain")),
		ProviderId:  happydns.Identifier([]byte("test-provider")),
		DomainName:  "example.com.",
		ZoneHistory: []happydns.Identifier{wipZoneID},
	}

	// Retriever succeeds on first call (lister diff), fails on second (re-fetch).
	retriever := &mockZoneRetrieverFailOnNth{
		records:   nil,
		failOnNth: 2,
		failErr:   fmt.Errorf("connection refused"),
	}

	uc := buildTestApplier(
		&mockProviderGetter{provider: provider},
		&mockZoneCorrector{corrections: nil, nbDiff: 0},
		retriever,
		&mockDomainUpdater{domain: domain},
		storage,
	)

	snapshot, err := uc.Apply(
		context.Background(),
		&happydns.User{Id: happydns.Identifier([]byte("test-user"))},
		domain,
		wipZone,
		&happydns.ApplyZoneForm{
			WantedCorrections: nil,
			CommitMsg:         "test deploy fallback",
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot, got nil")
	}
}

// getOriginSOASerial extracts the SOA serial from the Origin service in a zone.
func getOriginSOASerial(t *testing.T, zone *happydns.Zone) uint32 {
	t.Helper()
	if services, ok := zone.Services[""]; ok {
		for _, svc := range services {
			if svc.Type == "abstract.Origin" {
				if origin, ok := svc.Service.(*abstract.Origin); ok && origin.SOA != nil {
					return origin.SOA.Serial
				}
			}
		}
	}
	t.Fatal("no Origin service with SOA found in zone")
	return 0
}
