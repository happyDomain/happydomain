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
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	_ "git.happydns.org/happyDomain/services/providers/google"
)

func TestSPFContributor(t *testing.T) {
	spfSvc := &svcs.SPF{
		Record: &happydns.TXT{
			Hdr: dns.RR_Header{Rrtype: dns.TypeTXT, Class: dns.ClassINET},
			Txt: "v=spf1 ip4:203.0.113.0/24 include:example.com ~all",
		},
	}

	directives := spfSvc.GetSPFDirectives()
	if len(directives) != 2 {
		t.Fatalf("expected 2 directives, got %d: %v", len(directives), directives)
	}
	if directives[0] != "ip4:203.0.113.0/24" {
		t.Errorf("directives[0] = %q; want %q", directives[0], "ip4:203.0.113.0/24")
	}
	if directives[1] != "include:example.com" {
		t.Errorf("directives[1] = %q; want %q", directives[1], "include:example.com")
	}

	policy := spfSvc.GetSPFAllPolicy()
	if policy != "~all" {
		t.Errorf("GetSPFAllPolicy() = %q; want %q", policy, "~all")
	}
}

func TestSPFContributorHardFail(t *testing.T) {
	spfSvc := &svcs.SPF{
		Record: &happydns.TXT{
			Hdr: dns.RR_Header{Rrtype: dns.TypeTXT, Class: dns.ClassINET},
			Txt: "v=spf1 ip4:10.0.0.0/8 -all",
		},
	}

	policy := spfSvc.GetSPFAllPolicy()
	if policy != "-all" {
		t.Errorf("GetSPFAllPolicy() = %q; want %q", policy, "-all")
	}
}

func TestResolveSPFAllPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []string
		want     string
	}{
		{"empty defaults to ~all", nil, "~all"},
		{"single softfail", []string{"~all"}, "~all"},
		{"single hardfail", []string{"-all"}, "-all"},
		{"strictest wins", []string{"~all", "-all"}, "-all"},
		{"neutral vs softfail", []string{"?all", "~all"}, "~all"},
		{"pass vs hardfail", []string{"+all", "-all"}, "-all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svcs.ResolveSPFAllPolicy(tt.policies)
			if got != tt.want {
				t.Errorf("ResolveSPFAllPolicy(%v) = %q; want %q", tt.policies, got, tt.want)
			}
		})
	}
}

func TestMergeSPFDirectives(t *testing.T) {
	merged := svcs.MergeSPFDirectives(
		[]string{"include:_spf.google.com", "ip4:1.2.3.0/24"},
		[]string{"include:_spf.google.com", "ip4:5.6.7.0/24"},
	)

	if len(merged) != 3 {
		t.Fatalf("expected 3 directives, got %d: %v", len(merged), merged)
	}
}

func TestSPFAnalyze(t *testing.T) {
	rr, err := dns.NewRR("example.com. 3600 IN TXT \"v=spf1 ip4:203.0.113.0/24 ~all\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{txt})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service at apex, got %d", len(s[""]))
	}

	spfSvc, ok := s[""][0].Service.(*svcs.SPF)
	if !ok {
		t.Fatalf("Expected SPF service, got %T", s[""][0].Service)
	}

	if !strings.Contains(spfSvc.Record.Txt, "ip4:203.0.113.0/24") {
		t.Errorf("SPF record should contain ip4 directive, got %q", spfSvc.Record.Txt)
	}
}

func TestSPFAnalyzeWithGSuiteClaimedDirectives(t *testing.T) {
	// Simulate a zone with Google MX records and a combined SPF record
	records := []happydns.Record{}

	// Google MX records
	for i, mx := range []string{
		"aspmx.l.google.com.",
		"alt1.aspmx.l.google.com.",
		"alt2.aspmx.l.google.com.",
		"alt3.aspmx.l.google.com.",
		"alt4.aspmx.l.google.com.",
	} {
		rr, err := dns.NewRR("example.com. 3600 IN MX " + string(rune('1'+i)) + "0 " + mx)
		if err != nil {
			t.Fatalf("dns.NewRR MX failed: %v", err)
		}
		records = append(records, rr.(*dns.MX))
	}

	// Combined SPF record with both Google and custom directives
	spfRR, err := dns.NewRR("example.com. 3600 IN TXT \"v=spf1 include:_spf.google.com ip4:1.2.3.0/24 ~all\"")
	if err != nil {
		t.Fatalf("dns.NewRR TXT failed: %v", err)
	}
	records = append(records, happydns.NewTXT(spfRR.(*dns.TXT)))

	s, _, err := svcs.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	// Should have two services at apex: GSuite and SPF
	if len(s[""]) != 2 {
		t.Fatalf("Expected 2 services at apex, got %d: %v", len(s[""]), s[""])
	}

	// Find the SPF service
	var spfSvc *svcs.SPF
	for _, svc := range s[""] {
		if sp, ok := svc.Service.(*svcs.SPF); ok {
			spfSvc = sp
		}
	}

	if spfSvc == nil {
		t.Fatal("SPF service not found")
	}

	// The SPF service should NOT contain the Google include (claimed by GSuite)
	if strings.Contains(spfSvc.Record.Txt, "include:_spf.google.com") {
		t.Errorf("SPF service should not contain Google include directive, got %q", spfSvc.Record.Txt)
	}

	// But it should contain the custom directive
	if !strings.Contains(spfSvc.Record.Txt, "ip4:1.2.3.0/24") {
		t.Errorf("SPF service should contain ip4 directive, got %q", spfSvc.Record.Txt)
	}
}
