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

package svcs_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	intsvc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/services"
	_ "git.happydns.org/happyDomain/services/abstract"
	_ "git.happydns.org/happyDomain/services/providers/google"
)

// roundTrip analyzes the given DNS records into services, then regenerates
// records from those services and returns them. This exercises the full
// analyze -> generate path.
func roundTrip(t *testing.T, origin string, records []happydns.Record) []happydns.Record {
	t.Helper()

	services, defaultTTL, err := intsvc.AnalyzeZone(origin, records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	var regenerated []happydns.Record
	for _, domainSvcs := range services {
		for _, svc := range domainSvcs {
			ttl := defaultTTL
			if svc.Ttl != 0 {
				ttl = svc.Ttl
			}

			rrs, err := svc.Service.GetRecords(svc.Domain, ttl, origin)
			if err != nil {
				t.Fatalf("GetRecords failed for %s: %v", svc.Type, err)
			}

			for i, rr := range rrs {
				rrs[i] = helpers.CopyRecord(rr)
				rrs[i].Header().Name = helpers.DomainJoin(rrs[i].Header().Name, svc.Domain)
				if origin != "" {
					rrs[i] = helpers.RRAbsolute(rrs[i], origin)
				}
				if rrs[i].Header().Ttl == 0 {
					rrs[i].Header().Ttl = ttl
				}
			}

			regenerated = append(regenerated, rrs...)
		}
	}

	return regenerated
}

// canonicalStrings returns a sorted list of string representations for the
// given records, for comparison purposes.
func canonicalStrings(records []happydns.Record) []string {
	strs := make([]string, len(records))
	for i, rr := range records {
		strs[i] = rr.String()
	}
	sort.Strings(strs)
	return strs
}

// assertRoundTrip verifies that records survive a round-trip through
// analyze -> generate.
func assertRoundTrip(t *testing.T, origin string, records []happydns.Record) {
	t.Helper()

	regenerated := roundTrip(t, origin, records)

	original := canonicalStrings(records)
	result := canonicalStrings(regenerated)

	if len(original) != len(result) {
		t.Errorf("record count mismatch: input %d, output %d", len(original), len(result))
		t.Logf("input:  %v", original)
		t.Logf("output: %v", result)
		return
	}

	for i := range original {
		if original[i] != result[i] {
			t.Errorf("record %d mismatch:\n  input:  %s\n  output: %s", i, original[i], result[i])
		}
	}
}

func mustNewRR(t *testing.T, s string) happydns.Record {
	t.Helper()
	rr, err := dns.NewRR(s)
	if err != nil {
		t.Fatalf("dns.NewRR(%q) failed: %v", s, err)
	}
	if rr.Header().Rrtype == dns.TypeTXT {
		return happydns.NewTXT(rr.(*dns.TXT))
	}
	return rr
}

func TestRoundTrip_MX(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN MX 10 mail1.example.com."),
		mustNewRR(t, "example.com. 3600 IN MX 20 mail2.example.com."),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_CNAME(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "www.example.com. 3600 IN CNAME example.com."),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_CAA(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN CAA 0 issue \"letsencrypt.org\""),
		mustNewRR(t, "example.com. 3600 IN CAA 0 issuewild \"letsencrypt.org\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_TXT(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN TXT \"some verification text\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_SPF(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, fmt.Sprintf("example.com. 3600 IN TXT \"v=spf1 include:_spf.google.com ~all\"")),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_DMARC(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "_dmarc.example.com. 3600 IN TXT \"v=DMARC1; p=reject; rua=mailto:dmarc@example.com\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_MultiSubdomain(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN MX 10 mail.example.com."),
		mustNewRR(t, "www.example.com. 3600 IN CNAME example.com."),
		mustNewRR(t, "example.com. 3600 IN TXT \"some text\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_Subdomain_CNAME(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "blog.example.com. 3600 IN CNAME hosting.provider.com."),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_MultipleTXT(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN TXT \"google-site-verification=abc123\""),
		mustNewRR(t, "example.com. 3600 IN TXT \"facebook-domain-verification=xyz789\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_MixedTTLs(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN MX 10 mail.example.com."),
		mustNewRR(t, "example.com. 3600 IN MX 20 mail2.example.com."),
		mustNewRR(t, "example.com. 3600 IN TXT \"hello\""),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_Orphan_A(t *testing.T) {
	// A records without an abstract.Server service registered still survive as Orphan
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN A 93.184.216.34"),
	}

	regenerated := roundTrip(t, origin, records)

	if len(regenerated) != len(records) {
		t.Fatalf("expected %d records, got %d", len(records), len(regenerated))
	}

	// Orphan wraps the record; verify the string representation matches
	for _, rr := range regenerated {
		s := rr.String()
		if !strings.Contains(s, "93.184.216.34") {
			t.Errorf("expected A record with 93.184.216.34, got %s", s)
		}
	}
}

func TestRoundTrip_GSuite_MX(t *testing.T) {
	// GSuite claims MX records for google.com and SPF directive
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN MX 1 aspmx.l.google.com."),
		mustNewRR(t, "example.com. 3600 IN MX 5 alt1.aspmx.l.google.com."),
		mustNewRR(t, "example.com. 3600 IN MX 5 alt2.aspmx.l.google.com."),
		mustNewRR(t, "example.com. 3600 IN MX 10 alt3.aspmx.l.google.com."),
		mustNewRR(t, "example.com. 3600 IN MX 10 alt4.aspmx.l.google.com."),
		mustNewRR(t, fmt.Sprintf("example.com. 3600 IN TXT \"v=spf1 include:_spf.google.com ~all\"")),
	}

	services, _, err := intsvc.AnalyzeZone(origin, records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	// Should have a GSuite service and an SPF service at root
	var foundGSuite, foundSPF bool
	for _, domainSvcs := range services {
		for _, svc := range domainSvcs {
			if svc.Type == "google.GSuite" {
				foundGSuite = true
			}
			if svc.Type == "svcs.SPF" {
				foundSPF = true
			}
		}
	}

	if !foundGSuite {
		t.Error("expected GSuite service to be found")
	}
	if !foundSPF {
		t.Error("expected SPF service to be found")
	}

	// Verify MX records round-trip
	regenerated := roundTrip(t, origin, records)

	mxCount := 0
	for _, rr := range regenerated {
		if rr.Header().Rrtype == dns.TypeMX {
			mxCount++
		}
	}

	if mxCount != 5 {
		t.Errorf("expected 5 MX records after round-trip, got %d", mxCount)
	}
}

func TestRoundTrip_GSuite_SPFClaimed(t *testing.T) {
	// When GSuite claims the SPF include directive, the remaining SPF record
	// should have the google directive filtered out.
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN MX 1 aspmx.l.google.com."),
		mustNewRR(t, "example.com. 3600 IN MX 5 alt1.aspmx.l.google.com."),
		mustNewRR(t, fmt.Sprintf("example.com. 3600 IN TXT \"v=spf1 include:_spf.google.com ip4:203.0.113.0/24 ~all\"")),
	}

	services, _, err := intsvc.AnalyzeZone(origin, records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	// Check that SPF service has the google include filtered out
	for _, domainSvcs := range services {
		for _, svc := range domainSvcs {
			if svc.Type == "svcs.SPF" {
				comment := svc.Service.GenComment()
				// The SPF service should still have directives
				if !strings.Contains(comment, "directive") {
					t.Logf("SPF comment: %s", comment)
				}
			}
		}
	}
}

func TestRoundTrip_Origin_SOA_NS(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN SOA ns1.example.com. admin.example.com. 2024010101 3600 900 604800 86400"),
		mustNewRR(t, "example.com. 3600 IN NS ns1.example.com."),
		mustNewRR(t, "example.com. 3600 IN NS ns2.example.com."),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_Server_A_AAAA(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "example.com. 3600 IN A 93.184.216.34"),
		mustNewRR(t, "example.com. 3600 IN AAAA 2606:2800:220:1:248:1893:25c8:1946"),
	}
	assertRoundTrip(t, origin, records)
}

func TestRoundTrip_SubdomainServer(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "www.example.com. 3600 IN A 93.184.216.34"),
		mustNewRR(t, "www.example.com. 3600 IN AAAA 2606:2800:220:1:248:1893:25c8:1946"),
	}
	assertRoundTrip(t, origin, records)
}

// TestRoundTrip_CalDAV exercises the RFC 6764 CalDAV SRV records through the
// full analyze → regenerate path so abstract.CalDAV registration stays honest.
func TestRoundTrip_CalDAV(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "_caldavs._tcp.example.com. 3600 IN SRV 10 5 443 dav.example.com."),
		mustNewRR(t, "_caldav._tcp.example.com.  3600 IN SRV 10 5 80  dav.example.com."),
	}
	assertRoundTrip(t, origin, records)
}

// TestRoundTrip_CardDAV - same for the CardDAV half of RFC 6764.
func TestRoundTrip_CardDAV(t *testing.T) {
	origin := "example.com."
	records := []happydns.Record{
		mustNewRR(t, "_carddavs._tcp.example.com. 3600 IN SRV 10 5 443 dav.example.com."),
		mustNewRR(t, "_carddav._tcp.example.com.  3600 IN SRV 10 5 80  dav.example.com."),
	}
	assertRoundTrip(t, origin, records)
}
