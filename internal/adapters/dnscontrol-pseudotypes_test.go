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
	"fmt"
	"slices"
	"strings"
	"testing"

	dnscontrolmodels "github.com/DNSControl/dnscontrol/v4/models"
	"github.com/DNSControl/dnscontrol/v4/pkg/dnsrr"
	dnscontrol "github.com/DNSControl/dnscontrol/v4/pkg/providers"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// testAliasProviderName names a DNSControl provider registered for the tests,
// declaring it handles ALIAS but not R53_ALIAS.
var testAliasProviderName = func() string {
	const name = "TEST_ALIAS_PROVIDER"

	dnscontrol.RegisterDomainServiceProviderType(
		name,
		dnscontrol.DspFuncs{},
		dnscontrol.DocumentationNotes{
			dnscontrol.CanUseAlias:        dnscontrol.Can(),
			dnscontrol.CanUseRoute53Alias: dnscontrol.Cannot(),
		},
	)

	return name
}()

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
	// The supported pseudo-types whose whole rdata is a target are registered
	// in dns.StringToType, so that DNSControl handles them like any other type.
	for _, rtype := range []string{"A", "AAAA", "CNAME", "MX", "TXT", "SOA", "DS", "SVCB", "HTTPS", "ALIAS", "ANAME", "AKAMAICDN"} {
		if IsPseudoRecordType(rtype) {
			t.Errorf("IsPseudoRecordType(%q) = true, want false: ToRR() knows how to build it", rtype)
		}
	}

	// Pseudo-types of the DNSControl core, and a sample of the ones registered
	// by providers. None of them has a wire format, so ToRR() would abort the
	// process on any of them. R53_ALIAS, AZURE_ALIAS and AKAMAITLC are among
	// them although happyDomain represents them: they carry more than a target,
	// so DNSControl has to keep seeing them as pseudo-types.
	for _, rtype := range []string{
		"IMPORT_TRANSFORM",
		"R53_ALIAS", "AZURE_ALIAS", "AKAMAITLC", "URL", "URL301", "FRAME", "LUA",
		"CF_WORKER_ROUTE", "PORKBUN_URLFWD", "CLOUDNS_WR",
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
		{Type: "URL", Name: "shop"},
		{Type: "UNKNOWN", Name: "weird"},
	}

	got := dropPseudoRecords(records)

	if len(got) != 4 {
		t.Fatalf("dropPseudoRecords returned %d records, want 4", len(got))
	}
	for i, want := range []string{"A", "ALIAS", "MX", "R53_ALIAS"} {
		if got[i].Type != want {
			t.Errorf("dropPseudoRecords kept %q in position %d, want %q", got[i].Type, i, want)
		}
	}

	// The input must not be altered: GetZoneCorrections reuses it.
	if len(records) != 6 {
		t.Errorf("dropPseudoRecords mutated its input: %d records left, want 6", len(records))
	}
}

// TestGetZoneRecordsKeepsSupportedPseudoTypes is the regression test for the
// crash: a zone containing an ALIAS used to reach RecordConfig.ToRR(), whose
// log.Fatalf terminated happyDomain with exit code 1 (and could not be recovered
// from). It now comes back as a record of its own.
func TestGetZoneRecordsKeepsSupportedPseudoTypes(t *testing.T) {
	alias := &dnscontrolmodels.RecordConfig{Type: "ALIAS", TTL: 300}
	alias.SetLabel("@", "example.com")
	if err := alias.SetTarget("target.example.net."); err != nil {
		t.Fatalf("unable to build the ALIAS: %s", err)
	}

	r53 := &dnscontrolmodels.RecordConfig{
		Type: "R53_ALIAS",
		TTL:  300,
		R53Alias: map[string]string{
			"type":                   "A",
			"zone_id":                "Z1234",
			"evaluate_target_health": "false",
		},
	}
	r53.SetLabel("cdn", "example.com")
	if err := r53.SetTarget("aws.example.net."); err != nil {
		t.Fatalf("unable to build the R53_ALIAS: %s", err)
	}

	a := newTestAdapter(&mockDNSProvider{
		getZoneRecordsResult: dnscontrolmodels.Records{
			mustRecordConfig(t, "example.com", "www.example.com. 300 IN A 192.0.2.1"),
			alias,
			r53,
			// This one happyDomain does not represent: it has to be skipped.
			{Type: "URL", Name: "shop", NameFQDN: "shop.example.com", TTL: 300},
		},
	})

	got, err := a.GetZoneRecords("example.com.")
	if err != nil {
		t.Fatalf("GetZoneRecords returned an error: %s", err)
	}

	if len(got) != 3 {
		t.Fatalf("GetZoneRecords returned %d records, want 3 (only the URL must be skipped)", len(got))
	}

	if want := "example.com.\t300\tIN\tALIAS\ttarget.example.net."; got[1].String() != want {
		t.Errorf("GetZoneRecords returned %q for the ALIAS, want %q", got[1].String(), want)
	}

	rdata, ok := got[2].(*dns.PrivateRR).Data.(*happydns.R53AliasRdata)
	if !ok {
		t.Fatalf("GetZoneRecords returned a %T for the R53_ALIAS", got[2])
	}
	if rdata.ZoneID != "Z1234" || rdata.AType != "A" || rdata.Target != "aws.example.net." {
		t.Errorf("GetZoneRecords lost the R53_ALIAS fields: %#v", rdata)
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
		{Type: "URL", Name: "shop", NameFQDN: "shop.example.com", TTL: 300},
	}

	a := newTestAdapter(provider)

	// A private-use type outside the block happyDomain reserved for the
	// pseudo-types it represents, so that miekg/dns has no name for it.
	unknown, err := dns.NewRR("private.example.com. 300 IN TYPE65400 \\# 4 01020304")
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

// TestPseudoTypeTargetCombined locks the invariant the whole representation
// rests on: models.RecordConfig.GetTargetCombined is what a provider stores on
// its side and what the diff engine compares, and it is built differently
// depending on whether dns.StringToType knows the type.
//
// Registering a bare target pseudo-type moves it from "return rc.target" to the
// zone file representation of our rdata: both must give the same string. The
// ones carrying additional fields are deliberately left unregistered, so that
// DNSControl keeps building their composite target; our rdata reproduces it
// verbatim, so that a record shows the same thing wherever it is read.
func TestPseudoTypeTargetCombined(t *testing.T) {
	alias := &dnscontrolmodels.RecordConfig{Type: "ALIAS", TTL: 300}
	alias.SetLabel("@", "example.com")
	if err := alias.SetTarget("target.example.net."); err != nil {
		t.Fatalf("unable to build the ALIAS: %s", err)
	}

	if got, want := alias.GetTargetCombined(), "target.example.net."; got != want {
		t.Errorf("GetTargetCombined() = %q for an ALIAS, want %q: registering it changed what providers store", got, want)
	}

	r53 := &dnscontrolmodels.RecordConfig{
		Type: "R53_ALIAS",
		TTL:  300,
		R53Alias: map[string]string{
			"type":                   "A",
			"zone_id":                "Z1234",
			"evaluate_target_health": "false",
		},
	}
	r53.SetLabel("cdn", "example.com")
	if err := r53.SetTarget("aws.example.net."); err != nil {
		t.Fatalf("unable to build the R53_ALIAS: %s", err)
	}

	rdata := &happydns.R53AliasRdata{
		Target:               "aws.example.net.",
		AType:                "A",
		ZoneID:               "Z1234",
		EvaluateTargetHealth: "false",
	}

	if got, want := r53.GetTargetCombined(), rdata.String(); got != want {
		t.Errorf("GetTargetCombined() = %q for a R53_ALIAS, but happyDomain renders it %q", got, want)
	}
}

// TestPseudoTypeRoundTrip walks a pseudo-type record through both conversions,
// the way a zone read from a provider and published back does.
func TestPseudoTypeRoundTrip(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		record happydns.Record
	}{
		{
			name:   "ALIAS",
			record: mustRR(t, "example.com. 300 IN ALIAS target.example.net."),
		},
		{
			name: "R53_ALIAS",
			record: func() happydns.Record {
				rr := dns.TypeToRR[happydns.TypeR53ALIAS]().(*dns.PrivateRR)
				rr.Hdr = dns.RR_Header{Name: "cdn.example.com.", Rrtype: happydns.TypeR53ALIAS, Class: dns.ClassINET, Ttl: 300}
				*rr.Data.(*happydns.R53AliasRdata) = happydns.R53AliasRdata{
					Target:               "aws.example.net.",
					AType:                "A",
					ZoneID:               "Z1234",
					EvaluateTargetHealth: "false",
				}
				return rr
			}(),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rcs, err := DNSControlRRtoRC([]happydns.Record{testCase.record}, "example.com.")
			if err != nil {
				t.Fatalf("DNSControlRRtoRC returned an error: %s", err)
			}

			if rcs[0].Type != testCase.name {
				t.Errorf("DNSControlRRtoRC gave the type %q, want %q", rcs[0].Type, testCase.name)
			}

			back, err := recordFromRecordConfig(rcs[0])
			if err != nil {
				t.Fatalf("recordFromRecordConfig returned an error: %s", err)
			}

			if back.String() != testCase.record.String() {
				t.Errorf("the record came back as %q, want %q", back.String(), testCase.record.String())
			}
		})
	}
}

// TestGetZoneCorrectionsKeepsSupportedPseudoTypes checks the symmetry the diff
// relies on, on the side of the pseudo-types happyDomain now represents: the
// ALIAS must reach DNSControl from both the desired zone and the current one,
// otherwise the diff engine would take it for a record to create or to delete.
func TestGetZoneCorrectionsKeepsSupportedPseudoTypes(t *testing.T) {
	alias := &dnscontrolmodels.RecordConfig{Type: "ALIAS", TTL: 300}
	alias.SetLabel("@", "example.com")
	if err := alias.SetTarget("target.example.net."); err != nil {
		t.Fatalf("unable to build the ALIAS: %s", err)
	}

	provider := &mockCorrectionsProvider{}
	provider.getZoneRecordsResult = dnscontrolmodels.Records{alias}

	a := newTestAdapter(provider)
	a.providerName = testAliasProviderName

	wanted := []happydns.Record{mustRR(t, "example.com. 300 IN ALIAS target.example.net.")}

	if _, _, err := a.GetZoneCorrections("example.com.", wanted); err != nil {
		t.Fatalf("GetZoneCorrections returned an error: %s", err)
	}

	if len(provider.seenDesired) != 1 || provider.seenDesired[0].Type != "ALIAS" {
		t.Errorf("the desired zone holds %v, want a single ALIAS", provider.seenDesired)
	}
	if len(provider.seenExisting) != 1 || provider.seenExisting[0].Type != "ALIAS" {
		t.Errorf("the current zone holds %v, want a single ALIAS", provider.seenExisting)
	}

	// Both sides describe the same record: the diff must be empty.
	if got, want := provider.seenDesired[0].GetTargetCombined(), provider.seenExisting[0].GetTargetCombined(); got != want {
		t.Errorf("the two sides of the diff disagree: %q vs %q", got, want)
	}
}

// TestPseudoTypeCapabilities checks that a provider declaring it handles a
// pseudo-type gets the matching "rr-<num>-<NAME>" capability, which is what the
// frontend filters the alias kinds on.
func TestPseudoTypeCapabilities(t *testing.T) {
	caps := pseudoTypeCapabilities(testAliasProviderName)

	if want := fmt.Sprintf("rr-%d-ALIAS", happydns.TypeALIAS); !slices.Contains(caps, want) {
		t.Errorf("capabilities are %v, want them to contain %q", caps, want)
	}
	for _, unwanted := range []string{"R53_ALIAS", "ANAME"} {
		for _, capability := range caps {
			if strings.HasSuffix(capability, "-"+unwanted) {
				t.Errorf("capabilities hold %q, but the provider does not declare it", capability)
			}
		}
	}
}

// TestGetZoneCorrectionsRefusesUndeclaredPseudoTypes covers the provider that
// does not declare it handles a pseudo-type. happyDomain does not run
// DNSControl's own capability check, and a RecordAuditor audits the content of
// the records rather than their type, so nothing else would stop it.
func TestGetZoneCorrectionsRefusesUndeclaredPseudoTypes(t *testing.T) {
	provider := &mockCorrectionsProvider{}

	a := newTestAdapter(provider)
	a.providerName = testAliasProviderName

	// The test provider declares CanUseAlias, but explicitly not
	// CanUseRoute53Alias.
	r53 := dns.TypeToRR[happydns.TypeR53ALIAS]().(*dns.PrivateRR)
	r53.Hdr = dns.RR_Header{Name: "cdn.example.com.", Rrtype: happydns.TypeR53ALIAS, Class: dns.ClassINET, Ttl: 300}
	r53.Data.(*happydns.R53AliasRdata).Target = "aws.example.net."

	_, _, err := a.GetZoneCorrections("example.com.", []happydns.Record{r53})
	if err == nil {
		t.Fatal("GetZoneCorrections accepted a R53_ALIAS, but the provider does not declare it")
	}
	if !strings.Contains(err.Error(), "R53_ALIAS") {
		t.Errorf("the error does not name the offending type: %s", err)
	}

	if provider.seenDesired != nil {
		t.Errorf("the provider was handed a zone anyway: %v", provider.seenDesired)
	}
}
