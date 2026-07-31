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

package adapter

import (
	"testing"

	dnscontrolmodels "github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// mustRecordConfig builds the RecordConfig a provider would return for the
// given zone file line.
func mustRecordConfig(t *testing.T, origin, line string) *dnscontrolmodels.RecordConfig {
	t.Helper()

	rr, err := dns.NewRR(line)
	if err != nil {
		t.Fatalf("unable to parse %q: %s", line, err)
	}

	rc, err := dnsrr.RRtoRC(rr, origin)
	if err != nil {
		t.Fatalf("unable to convert %q: %s", line, err)
	}

	return &rc
}

func TestIsPseudoRecordType(t *testing.T) {
	for _, rtype := range []string{"A", "AAAA", "CNAME", "MX", "TXT", "SOA", "DS", "SVCB", "HTTPS"} {
		if IsPseudoRecordType(rtype) {
			t.Errorf("IsPseudoRecordType(%q) = true, want false: this is a real DNS type", rtype)
		}
	}

	// Pseudo-types of the DNSControl core, and a sample of the ones registered
	// by providers. None of them has a wire format, so ToRR() would abort the
	// process on any of them.
	for _, rtype := range []string{
		"ALIAS", "IMPORT_TRANSFORM",
		"R53_ALIAS", "AZURE_ALIAS", "URL", "URL301", "FRAME", "LUA",
		"CF_WORKER_ROUTE", "PORKBUN_URLFWD", "CLOUDNS_WR", "AKAMAICDN",
		"MIKROTIK_NXDOMAIN", "BUNNY_DNS_RDR", "NETLIFYv6",
		"ADGUARDHOME_A_PASSTHROUGH",
		"UNKNOWN",
		// The type DNSControl gives to a dns.RFC3597, ie. to a private-use
		// type miekg/dns has no name for.
		"",
		// Known names with no record implementation: ToRR() would call a nil
		// constructor and panic.
		"AXFR", "IXFR", "MAILA", "MAILB", "ATMA", "UNSPEC", "None", "Reserved",
	} {
		if !IsPseudoRecordType(rtype) {
			t.Errorf("IsPseudoRecordType(%q) = false, want true: this is a pseudo-type", rtype)
		}
	}
}

func TestDropPseudoRecords(t *testing.T) {
	records := dnscontrolmodels.Records{
		{Type: "A", Name: "www"},
		{Type: "ALIAS", Name: "@"},
		{Type: "MX", Name: "@"},
		{Type: "R53_ALIAS", Name: "cdn"},
	}

	got := dropPseudoRecords(records)

	if len(got) != 2 {
		t.Fatalf("dropPseudoRecords returned %d records, want 2", len(got))
	}
	if got[0].Type != "A" || got[1].Type != "MX" {
		t.Errorf("dropPseudoRecords kept %q and %q, want \"A\" and \"MX\"", got[0].Type, got[1].Type)
	}

	// The input must not be altered: GetZoneCorrections reuses it.
	if len(records) != 4 {
		t.Errorf("dropPseudoRecords mutated its input: %d records left, want 4", len(records))
	}
}

// TestGetZoneRecordsSkipsPseudoTypes is the regression test for the crash: a
// zone containing an ALIAS used to reach RecordConfig.ToRR(), whose log.Fatalf
// terminated happyDomain with exit code 1 (and could not be recovered from).
func TestGetZoneRecordsSkipsPseudoTypes(t *testing.T) {
	a := newTestAdapter(&mockDNSProvider{
		getZoneRecordsResult: dnscontrolmodels.Records{
			mustRecordConfig(t, "example.com", "www.example.com. 300 IN A 192.0.2.1"),
			{Type: "ALIAS", Name: "@", NameFQDN: "example.com", TTL: 300},
		},
	})

	got, err := a.GetZoneRecords("example.com.")
	if err != nil {
		t.Fatalf("GetZoneRecords returned an error: %s", err)
	}

	if len(got) != 1 {
		t.Fatalf("GetZoneRecords returned %d records, want 1 (the ALIAS must be skipped)", len(got))
	}
}

// mockCorrectionsProvider records the desired zone it was handed, so the test
// can check what GetZoneCorrections asks DNSControl to diff.
type mockCorrectionsProvider struct {
	mockDNSProvider

	seenDesired  dnscontrolmodels.Records
	seenExisting dnscontrolmodels.Records
}

func (m *mockCorrectionsProvider) GetZoneRecordsCorrections(dc *dnscontrolmodels.DomainConfig, existing dnscontrolmodels.Records) ([]*dnscontrolmodels.Correction, int, error) {
	m.seenDesired = dc.Records
	m.seenExisting = existing

	return nil, 0, nil
}

// TestGetZoneCorrectionsFiltersBothSides checks the symmetry the diff relies on:
// a record happyDomain cannot represent must be absent from the desired zone as
// well as from the current one. A private-use type (stored as a dns.RFC3597 when
// a zone file is uploaded) becomes a RecordConfig with an empty Type, which
// ToRR() cannot handle either; left in the desired zone alone, DNSControl would
// take it for a record to create and abort the process on it.
func TestGetZoneCorrectionsFiltersBothSides(t *testing.T) {
	provider := &mockCorrectionsProvider{}
	provider.getZoneRecordsResult = dnscontrolmodels.Records{
		mustRecordConfig(t, "example.com", "www.example.com. 300 IN A 192.0.2.1"),
		{Type: "ALIAS", Name: "@", NameFQDN: "example.com", TTL: 300},
	}

	a := newTestAdapter(provider)

	unknown, err := dns.NewRR("private.example.com. 300 IN TYPE65280 \\# 4 01020304")
	if err != nil {
		t.Fatalf("unable to build the private-use record: %s", err)
	}

	wanted := []happydns.Record{
		mustRR(t, "www.example.com. 300 IN A 192.0.2.1"),
		unknown,
	}

	if _, _, err := a.GetZoneCorrections("example.com.", wanted); err != nil {
		t.Fatalf("GetZoneCorrections returned an error: %s", err)
	}

	for _, rc := range provider.seenDesired {
		if IsPseudoRecordType(rc.Type) {
			t.Errorf("the desired zone still holds a %q record, it must have been dropped", rc.Type)
		}
	}
	if len(provider.seenDesired) != 1 {
		t.Errorf("the desired zone holds %d records, want 1", len(provider.seenDesired))
	}

	for _, rc := range provider.seenExisting {
		if IsPseudoRecordType(rc.Type) {
			t.Errorf("the current zone still holds a %q record, it must have been dropped", rc.Type)
		}
	}
	if len(provider.seenExisting) != 1 {
		t.Errorf("the current zone holds %d records, want 1", len(provider.seenExisting))
	}
}

// TestGetZoneCorrectionsWithoutAuditor covers the providers registered without a
// RecordAuditor: the field is left nil by NewDNSControlAdapterNSProvider, and
// calling it would panic.
func TestGetZoneCorrectionsWithoutAuditor(t *testing.T) {
	a := newTestAdapter(&mockCorrectionsProvider{})
	a.RecordAuditor = nil

	_, _, err := a.GetZoneCorrections("example.com.", []happydns.Record{
		mustRR(t, "www.example.com. 300 IN A 192.0.2.1"),
	})
	if err != nil {
		t.Fatalf("GetZoneCorrections returned an error: %s", err)
	}
}

// mustRR parses a zone file line into the record type happyDomain stores.
func mustRR(t *testing.T, line string) happydns.Record {
	t.Helper()

	rr, err := dns.NewRR(line)
	if err != nil {
		t.Fatalf("unable to parse %q: %s", line, err)
	}

	return rr
}
