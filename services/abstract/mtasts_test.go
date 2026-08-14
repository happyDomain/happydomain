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

package abstract_test

import (
	"testing"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

func mkTXT(name, value string) *happydns.TXT {
	return &happydns.TXT{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: value,
	}
}

func TestMTASTS_GetRecords(t *testing.T) {
	s := &abstract.MTASTS{
		Mode:        "enforce",
		MaxAge:      604800,
		MX:          []string{"mail.example.com", "*.example.net"},
		Record:      mkTXT("_mta-sts", "v=STSv1; id=20260814T101500Z"),
		PolicyCNAME: mkCNAME("mta-sts", "happydomain.example.com."),
	}

	if n := s.GetNbResources(); n != 2 {
		t.Errorf("GetNbResources() = %d; want 2", n)
	}

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if len(rrs) != 2 {
		t.Fatalf("GetRecords() returned %d records; want 2", len(rrs))
	}
	if got := rrs[0].Header().Name; got != "_mta-sts" {
		t.Errorf("first record name = %q; want %q", got, "_mta-sts")
	}
	if got := rrs[1].Header().Name; got != "mta-sts" {
		t.Errorf("second record name = %q; want %q", got, "mta-sts")
	}
}

// A service whose editor never enabled hosting must not publish the stubs the
// service-spec auto-initializer leaves behind.
func TestMTASTS_GetRecords_Stubs(t *testing.T) {
	s := &abstract.MTASTS{
		Mode:        "testing",
		Record:      &happydns.TXT{},
		PolicyCNAME: &dns.CNAME{},
	}

	if n := s.GetNbResources(); n != 0 {
		t.Errorf("GetNbResources() = %d; want 0", n)
	}

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if len(rrs) != 0 {
		t.Errorf("GetRecords() returned %d records; want 0", len(rrs))
	}
}

func TestMTASTS_PolicyFile(t *testing.T) {
	s := &abstract.MTASTS{
		Mode:   "enforce",
		MaxAge: 604800,
		MX:     []string{"mail.example.com.", " ", "*.example.net"},
	}

	want := "version: STSv1\r\n" +
		"mode: enforce\r\n" +
		"mx: mail.example.com\r\n" +
		"mx: *.example.net\r\n" +
		"max_age: 604800\r\n"

	if got := string(s.PolicyFile()); got != want {
		t.Errorf("PolicyFile() = %q; want %q", got, want)
	}
}

func TestMTASTS_PolicyFile_Defaults(t *testing.T) {
	s := &abstract.MTASTS{MX: []string{"mail.example.com"}}

	want := "version: STSv1\r\n" +
		"mode: testing\r\n" +
		"mx: mail.example.com\r\n" +
		"max_age: 604800\r\n"

	if got := string(s.PolicyFile()); got != want {
		t.Errorf("PolicyFile() = %q; want %q", got, want)
	}
}

func TestMTASTS_PolicyFile_Empty(t *testing.T) {
	if body := (&abstract.MTASTS{}).PolicyFile(); body != nil {
		t.Errorf("PolicyFile() = %q; want nil", body)
	}
}

// The TXT record alone still belongs to the low-level svcs.MTA_STS service:
// nothing tells us happyDomain serves the policy file.
func TestMTASTS_Analyze_TXTOnlyStaysLowLevel(t *testing.T) {
	rr, err := dns.NewRR(`_mta-sts.example.com. 3600 IN TXT "v=STSv1; id=20240601T000000;"`)
	if err != nil {
		t.Fatalf("dns.NewRR: %v", err)
	}

	services, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone: %v", err)
	}

	if len(services[""]) != 1 {
		t.Fatalf("Expected 1 service at the apex, got %d", len(services[""]))
	}
	if _, ok := services[""][0].Service.(*svcs.MTA_STS); !ok {
		t.Fatalf("Expected *svcs.MTA_STS, got %T", services[""][0].Service)
	}
}

// The TXT record plus the mta-sts. CNAME is the unambiguous signal that the
// policy file is hosted for this domain: both records go to one MTASTS.
func TestMTASTS_Analyze_WithPolicyHost(t *testing.T) {
	txt, err := dns.NewRR(`_mta-sts.example.com. 3600 IN TXT "v=STSv1; id=20240601T000000;"`)
	if err != nil {
		t.Fatalf("dns.NewRR: %v", err)
	}
	cname, err := dns.NewRR(`mta-sts.example.com. 3600 IN CNAME happydomain.example.com.`)
	if err != nil {
		t.Fatalf("dns.NewRR: %v", err)
	}

	services, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{txt, cname})
	if err != nil {
		t.Fatalf("AnalyzeZone: %v", err)
	}

	if len(services[""]) != 1 {
		t.Fatalf("Expected 1 service at the apex, got %d", len(services[""]))
	}

	s, ok := services[""][0].Service.(*abstract.MTASTS)
	if !ok {
		t.Fatalf("Expected *abstract.MTASTS, got %T", services[""][0].Service)
	}
	if s.Record == nil || s.Record.Hdr.Name != "_mta-sts" {
		t.Errorf("TXT record name = %q; want %q", s.Record.Hdr.Name, "_mta-sts")
	}
	if s.PolicyCNAME == nil || s.PolicyCNAME.Hdr.Name != "mta-sts" {
		t.Errorf("CNAME record name = %q; want %q", s.PolicyCNAME.Hdr.Name, "mta-sts")
	}
	if n := s.GetNbResources(); n != 2 {
		t.Errorf("GetNbResources() = %d; want 2", n)
	}
}
