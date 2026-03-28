// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package adapter

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// mockLibdnsProvider implements RecordGetter, RecordAppender, RecordDeleter for testing.
type mockLibdnsProvider struct {
	records     []libdns.Record
	appended    []libdns.Record
	deleted     []libdns.Record
	zones       []libdns.Zone
	appendErr   error
	deleteErr   error
	getErr      error
	listZoneErr error
}

func (m *mockLibdnsProvider) GetRecords(_ context.Context, _ string) ([]libdns.Record, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.records, nil
}

func (m *mockLibdnsProvider) AppendRecords(_ context.Context, _ string, recs []libdns.Record) ([]libdns.Record, error) {
	if m.appendErr != nil {
		return nil, m.appendErr
	}
	m.appended = append(m.appended, recs...)
	return recs, nil
}

func (m *mockLibdnsProvider) DeleteRecords(_ context.Context, _ string, recs []libdns.Record) ([]libdns.Record, error) {
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	m.deleted = append(m.deleted, recs...)
	return recs, nil
}

func (m *mockLibdnsProvider) ListZones(_ context.Context) ([]libdns.Zone, error) {
	if m.listZoneErr != nil {
		return nil, m.listZoneErr
	}
	return m.zones, nil
}

// mockLibdnsConfig implements LibdnsConfigAdapter.
type mockLibdnsConfig struct {
	provider any
}

func (m *mockLibdnsConfig) LibdnsProvider() any {
	return m.provider
}

func (m *mockLibdnsConfig) InstantiateProvider() (happydns.ProviderActuator, error) {
	return NewLibdnsProviderAdapter(m)
}

func TestNewLibdnsProviderAdapter(t *testing.T) {
	mock := &mockLibdnsProvider{}
	config := &mockLibdnsConfig{provider: mock}

	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !adapter.CanListZones() {
		t.Error("expected CanListZones to be true")
	}
	if adapter.CanCreateDomain() {
		t.Error("expected CanCreateDomain to be false")
	}
}

func TestLibdnsAdapter_GetZoneRecords(t *testing.T) {
	mock := &mockLibdnsProvider{
		records: []libdns.Record{
			libdns.Address{
				Name: "www",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.1"),
			},
			libdns.TXT{
				Name: "@",
				TTL:  300 * time.Second,
				Text: "v=spf1 ~all",
			},
		},
	}

	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records, err := adapter.GetZoneRecords("example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Check A record
	if records[0].Header().Rrtype != dns.TypeA {
		t.Errorf("expected first record to be A, got %s", dns.TypeToString[records[0].Header().Rrtype])
	}

	// Check TXT record
	txt, ok := records[1].(*happydns.TXT)
	if !ok {
		t.Fatalf("expected second record to be *happydns.TXT, got %T", records[1])
	}
	if txt.Txt != "v=spf1 ~all" {
		t.Errorf("expected TXT 'v=spf1 ~all', got %q", txt.Txt)
	}
}

func TestLibdnsAdapter_ListZones(t *testing.T) {
	mock := &mockLibdnsProvider{
		zones: []libdns.Zone{
			{Name: "example.com."},
			{Name: "example.org."},
		},
	}

	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zones, err := adapter.ListZones()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	if zones[0] != "example.com." {
		t.Errorf("expected first zone 'example.com.', got %q", zones[0])
	}
}

func TestLibdnsAdapter_GetZoneCorrections_NoChanges(t *testing.T) {
	records := []libdns.Record{
		libdns.Address{
			Name: "www",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
	}

	mock := &mockLibdnsProvider{records: records}
	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pass the same records as wanted
	aRR, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
	corrections, _, err := adapter.GetZoneCorrections("example.com.", []happydns.Record{aRR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(corrections) != 0 {
		t.Errorf("expected 0 corrections, got %d", len(corrections))
	}
}

func TestLibdnsAdapter_GetZoneCorrections_Addition(t *testing.T) {
	// Provider has one A record, we want to add a CNAME.
	mock := &mockLibdnsProvider{
		records: []libdns.Record{
			libdns.Address{
				Name: "www",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.1"),
			},
		},
	}

	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aRR, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
	cnameRR, _ := dns.NewRR("blog.example.com. 300 IN CNAME www.example.com.")
	corrections, _, err := adapter.GetZoneCorrections("example.com.", []happydns.Record{aRR, cnameRR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(corrections) == 0 {
		t.Fatal("expected at least 1 correction")
	}

	// Execute the correction
	for _, c := range corrections {
		if c.Kind == happydns.CorrectionKindAddition {
			if err := c.F(); err != nil {
				t.Fatalf("unexpected error executing correction: %v", err)
			}
		}
	}

	if len(mock.appended) == 0 {
		t.Error("expected records to be appended")
	}
}

func TestLibdnsAdapter_GetZoneCorrections_Deletion(t *testing.T) {
	// Provider has two records, we want only one.
	mock := &mockLibdnsProvider{
		records: []libdns.Record{
			libdns.Address{
				Name: "www",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.1"),
			},
			libdns.Address{
				Name: "old",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.2"),
			},
		},
	}

	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aRR, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
	corrections, _, err := adapter.GetZoneCorrections("example.com.", []happydns.Record{aRR})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(corrections) == 0 {
		t.Fatal("expected at least 1 correction")
	}

	// Execute the deletion correction
	for _, c := range corrections {
		if c.Kind == happydns.CorrectionKindDeletion {
			if err := c.F(); err != nil {
				t.Fatalf("unexpected error executing correction: %v", err)
			}
		}
	}

	if len(mock.deleted) == 0 {
		t.Error("expected records to be deleted")
	}
}

func TestGetLibdnsProviderCapabilities(t *testing.T) {
	mock := &mockLibdnsProvider{}
	config := &mockLibdnsConfig{provider: mock}

	caps := GetLibdnsProviderCapabilities(config)

	// Should include ListDomains since mock implements ZoneLister
	found := false
	for _, c := range caps {
		if c == "ListDomains" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ListDomains capability")
	}

	// Should include common RR types
	expectedTypes := []string{"rr-1-A", "rr-28-AAAA", "rr-5-CNAME", "rr-15-MX", "rr-16-TXT"}
	for _, expected := range expectedTypes {
		found = false
		for _, c := range caps {
			if c == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected capability %s", expected)
		}
	}
}
