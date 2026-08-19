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
	"slices"
	"strings"
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
	found := slices.Contains(caps, "ListDomains")
	if !found {
		t.Error("expected ListDomains capability")
	}

	// Should include common RR types
	expectedTypes := []string{"rr-1-A", "rr-28-AAAA", "rr-5-CNAME", "rr-15-MX", "rr-16-TXT"}
	for _, expected := range expectedTypes {
		found = slices.Contains(caps, expected)
		if !found {
			t.Errorf("expected capability %s", expected)
		}
	}
}

// TestLibdnsAdapterRefusesPseudoTypes covers the pseudo-types on the libdns
// path: libdns carries a type name and its rdata as plain text, so nothing
// tells whether the provider behind it gives ALIAS any meaning. None of them
// declares the capability, and the frontend hides the kinds accordingly, but
// the API is reachable on its own.
func TestLibdnsAdapterRefusesPseudoTypes(t *testing.T) {
	mock := &mockLibdnsProvider{}

	config := &mockLibdnsConfig{provider: mock}
	adapter, err := NewLibdnsProviderAdapter(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	aliasRR, err := dns.NewRR("example.com. 300 IN ALIAS target.example.net.")
	if err != nil {
		t.Fatalf("unable to build the ALIAS: %s", err)
	}

	_, _, err = adapter.GetZoneCorrections("example.com.", []happydns.Record{aliasRR})
	if err == nil {
		t.Fatal("GetZoneCorrections accepted an ALIAS, it must refuse it")
	}
	if !strings.Contains(err.Error(), "ALIAS") {
		t.Errorf("the error does not name the offending type: %s", err)
	}

	if len(mock.appended) != 0 {
		t.Errorf("the provider was handed %d records, want none", len(mock.appended))
	}
}

// TestLibdnsAdapterSkipsPseudoTypesOnImport checks the other direction: a
// pseudo-type already sitting in the zone must not break the whole import, and
// must stay out of the diff, on both sides.
func TestLibdnsAdapterSkipsPseudoTypesOnImport(t *testing.T) {
	mock := &mockLibdnsProvider{
		records: []libdns.Record{
			libdns.Address{
				Name: "www",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.1"),
			},
			libdns.RR{
				Name: "@",
				TTL:  300 * time.Second,
				Type: "ALIAS",
				Data: "target.example.net.",
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
		t.Fatalf("GetZoneRecords returned an error: %s", err)
	}

	if len(records) != 1 {
		t.Fatalf("GetZoneRecords returned %d records, want 1 (the ALIAS must be skipped)", len(records))
	}

	// The zone read back holds exactly what happyDomain knows about: no
	// correction, and above all no deletion of the ALIAS.
	aRR, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")
	corrections, nb, err := adapter.GetZoneCorrections("example.com.", []happydns.Record{aRR})
	if err != nil {
		t.Fatalf("GetZoneCorrections returned an error: %s", err)
	}
	if nb != 0 || len(corrections) != 0 {
		t.Errorf("got %d corrections, want none: %v", nb, corrections)
	}
}

// providerRecord stands for the concrete type a real libdns provider hands
// back, carrying the identifier it needs to delete the record afterwards.
type providerRecord struct {
	rr libdns.RR
	id string
}

func (p providerRecord) RR() libdns.RR { return p.rr }

// TestLibdnsAdapterDeleteKeepsProviderRecord checks a deletion is asked with
// the very record the provider gave us, and not with the plain libdns.RR the
// conversion rebuilds: providers identify the record to delete by the data they
// attached to it, which only their own type carries.
//
// The records below are all spelled the way a provider may spell them, rather
// than the way miekg/dns writes them back.
func TestLibdnsAdapterDeleteKeepsProviderRecord(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rrType string
		data   string
	}{
		{"absolute target", "MX", "10 mail.example.com."},
		{"relative target", "MX", "10 mail"},
		{"relative CNAME target", "CNAME", "target"},
		{"quoted TXT", "TXT", `"hello world"`},
		{"unquoted TXT", "TXT", "hello world"},
		{"TXT in several strings", "TXT", `"part one " "part two"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := providerRecord{
				rr: libdns.RR{Name: "sub", TTL: 300 * time.Second, Type: tc.rrType, Data: tc.data},
				id: "provider-id-42",
			}

			mock := &mockLibdnsProvider{records: []libdns.Record{original}}
			adapter, err := NewLibdnsProviderAdapter(&mockLibdnsConfig{provider: mock})
			if err != nil {
				t.Fatalf("NewLibdnsProviderAdapter: %v", err)
			}

			// An empty desired zone: the record has to be deleted.
			corrections, nb, err := adapter.GetZoneCorrections("example.com", nil)
			if err != nil {
				t.Fatalf("GetZoneCorrections: %v", err)
			}
			if nb != 1 {
				t.Fatalf("got %d corrections, expected the deletion alone", nb)
			}
			if corrections[0].Kind != happydns.CorrectionKindDeletion {
				t.Fatalf("correction kind is %v, expected a deletion", corrections[0].Kind)
			}

			if err := corrections[0].F(); err != nil {
				t.Fatalf("applying the correction: %v", err)
			}

			if len(mock.deleted) != 1 {
				t.Fatalf("%d records deleted, expected 1", len(mock.deleted))
			}
			deleted, ok := mock.deleted[0].(providerRecord)
			if !ok {
				t.Fatalf("the deletion was asked with a %T rebuilt from the diff (%v), not with the record the provider gave us", mock.deleted[0], mock.deleted[0].RR())
			}
			if deleted.id != original.id {
				t.Errorf("the deleted record carries the id %q, expected %q", deleted.id, original.id)
			}
		})
	}
}
