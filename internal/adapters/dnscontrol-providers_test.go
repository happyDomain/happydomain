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

// Package adapter provides tests for DNSControlAdapterNSProvider instrumentation.
// These tests live in the internal package so they can construct the struct
// directly and set the unexported providerName field used in metric labels.
package adapter

import (
	"errors"
	"testing"

	dnscontrolmodels "github.com/StackExchange/dnscontrol/v4/models"
	dnscontrol "github.com/StackExchange/dnscontrol/v4/pkg/providers"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"git.happydns.org/happyDomain/internal/metrics"
)

// --- mock DNSServiceProvider -------------------------------------------------

// mockDNSProvider implements dnscontrol.DNSServiceProvider (i.e. models.DNSProvider).
type mockDNSProvider struct {
	getZoneRecordsErr    error
	getZoneRecordsResult dnscontrolmodels.Records
	correctionsErr       error
	panicOnGetZoneRecords bool
}

func (m *mockDNSProvider) GetNameservers(domain string) ([]*dnscontrolmodels.Nameserver, error) {
	return nil, nil
}

func (m *mockDNSProvider) GetZoneRecords(domain string, meta map[string]string) (dnscontrolmodels.Records, error) {
	if m.panicOnGetZoneRecords {
		panic("simulated provider panic")
	}
	return m.getZoneRecordsResult, m.getZoneRecordsErr
}

func (m *mockDNSProvider) GetZoneRecordsCorrections(dc *dnscontrolmodels.DomainConfig, existing dnscontrolmodels.Records) ([]*dnscontrolmodels.Correction, int, error) {
	return nil, 0, m.correctionsErr
}

// mockZoneLister extends mockDNSProvider with ZoneLister.
type mockZoneLister struct {
	mockDNSProvider
	listErr    error
	listResult []string
}

func (m *mockZoneLister) ListZones() ([]string, error) {
	return m.listResult, m.listErr
}

// mockZoneCreator extends mockDNSProvider with ZoneCreator.
type mockZoneCreator struct {
	mockDNSProvider
	ensureErr error
}

func (m *mockZoneCreator) EnsureZoneExists(domain string, metadata map[string]string) error {
	return m.ensureErr
}

// --- helpers -----------------------------------------------------------------

// noopAuditor is a RecordAuditor that approves all records.
func noopAuditor(rcs []*dnscontrolmodels.RecordConfig) []error { return nil }

// newTestAdapter constructs a DNSControlAdapterNSProvider with the given
// provider mock and a fixed providerName so metric labels are predictable.
func newTestAdapter(provider dnscontrol.DNSServiceProvider) *DNSControlAdapterNSProvider {
	return &DNSControlAdapterNSProvider{
		DNSServiceProvider: provider,
		RecordAuditor:      noopAuditor,
		providerName:       "TEST_PROVIDER",
	}
}

// --- GetZoneRecords ----------------------------------------------------------

func TestObserveProviderCall_GetZoneRecords_Success(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockDNSProvider{})
	_, err := a.GetZoneRecords("example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_records", "success")); got != 1 {
		t.Errorf("expected counter=1 for success, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_records", "error")); got != 0 {
		t.Errorf("expected error counter=0, got %v", got)
	}
}

func TestObserveProviderCall_GetZoneRecords_Error(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockDNSProvider{getZoneRecordsErr: errors.New("upstream timeout")})
	_, err := a.GetZoneRecords("example.com.")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_records", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_records", "success")); got != 0 {
		t.Errorf("expected success counter=0, got %v", got)
	}
}

func TestObserveProviderCall_GetZoneRecords_Panic(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockDNSProvider{panicOnGetZoneRecords: true})
	// The recover() block in GetZoneRecords must catch the panic and return an
	// error, which the observe closure then records as "error".
	_, err := a.GetZoneRecords("example.com.")
	if err == nil {
		t.Fatal("expected panic to be recovered as an error, got nil")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_records", "error")); got != 1 {
		t.Errorf("expected error counter=1 after recovered panic, got %v", got)
	}
}

// --- GetZoneCorrections ------------------------------------------------------

func TestObserveProviderCall_GetZoneCorrections_Success(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockDNSProvider{})
	_, _, err := a.GetZoneCorrections("example.com.", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_corrections", "success")); got != 1 {
		t.Errorf("expected counter=1 for success, got %v", got)
	}
}

func TestObserveProviderCall_GetZoneCorrections_Error(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockDNSProvider{getZoneRecordsErr: errors.New("provider down")})
	_, _, err := a.GetZoneCorrections("example.com.", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "get_zone_corrections", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
}

// --- CreateDomain ------------------------------------------------------------

func TestObserveProviderCall_CreateDomain_NotSupported(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	// mockDNSProvider does not implement ZoneCreator, so CreateDomain must fail.
	a := newTestAdapter(&mockDNSProvider{})
	err := a.CreateDomain("example.com.")
	if err == nil {
		t.Fatal("expected error when provider does not support domain creation")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "create_domain", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
}

func TestObserveProviderCall_CreateDomain_Success(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockZoneCreator{})
	err := a.CreateDomain("example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "create_domain", "success")); got != 1 {
		t.Errorf("expected success counter=1, got %v", got)
	}
}

func TestObserveProviderCall_CreateDomain_Error(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockZoneCreator{ensureErr: errors.New("zone already exists")})
	err := a.CreateDomain("example.com.")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "create_domain", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
}

// --- ListZones ---------------------------------------------------------------

func TestObserveProviderCall_ListZones_NotSupported(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	// mockDNSProvider does not implement ZoneLister, so ListZones must fail.
	a := newTestAdapter(&mockDNSProvider{})
	_, err := a.ListZones()
	if err == nil {
		t.Fatal("expected error when provider does not support zone listing")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "list_zones", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
}

func TestObserveProviderCall_ListZones_Success(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockZoneLister{listResult: []string{"example.com", "example.net"}})
	zones, err := a.ListZones()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(zones))
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "list_zones", "success")); got != 1 {
		t.Errorf("expected success counter=1, got %v", got)
	}
}

func TestObserveProviderCall_ListZones_Error(t *testing.T) {
	metrics.ProviderAPICallsTotal.Reset()

	a := newTestAdapter(&mockZoneLister{listErr: errors.New("API rate limit")})
	_, err := a.ListZones()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if got := testutil.ToFloat64(metrics.ProviderAPICallsTotal.WithLabelValues("TEST_PROVIDER", "list_zones", "error")); got != 1 {
		t.Errorf("expected error counter=1, got %v", got)
	}
}
