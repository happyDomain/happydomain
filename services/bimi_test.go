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
	"testing"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestBIMIService(t *testing.T) {
	txtStr := "v=BIMI1; l=https://example.com/logo.svg; a=https://example.com/vmc.pem"
	rr, err := dns.NewRR("default._bimi.example.com. 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 || len(s[""]) != 1 {
		t.Fatalf("Expected 1 service at apex, got %v", s)
	}

	bimiSvc, ok := s[""][0].Service.(*svcs.BIMI)
	if !ok {
		t.Fatalf("Expected *svcs.BIMI, got %T", s[""][0].Service)
	}

	if n := bimiSvc.GetNbResources(); n != 1 {
		t.Errorf("GetNbResources() = %d, want 1", n)
	}

	if c := bimiSvc.GenComment(); c != "default" {
		t.Errorf("GenComment() = %q, want %q", c, "default")
	}

	records, err := bimiSvc.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}
	rec, ok := records[0].(*happydns.TXT)
	if !ok {
		t.Fatalf("Expected *happydns.TXT, got %T", records[0])
	}
	if rec.Txt != txtStr {
		t.Errorf("Txt = %q, want %q", rec.Txt, txtStr)
	}
}

func TestBIMIServiceCustomSelector(t *testing.T) {
	rr, err := dns.NewRR("brand._bimi.example.com. 3600 IN TXT \"v=BIMI1; l=https://example.com/logo.svg\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service at apex, got %d", len(s[""]))
	}
	bimiSvc, ok := s[""][0].Service.(*svcs.BIMI)
	if !ok {
		t.Fatalf("Expected *svcs.BIMI, got %T", s[""][0].Service)
	}
	if c := bimiSvc.GenComment(); c != "brand" {
		t.Errorf("GenComment() = %q, want %q", c, "brand")
	}
}

func TestBIMIServiceIgnoresNonBIMI(t *testing.T) {
	rr, err := dns.NewRR("default._bimi.example.com. 3600 IN TXT \"foo=bar\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	for _, services := range s {
		for _, svc := range services {
			if _, ok := svc.Service.(*svcs.BIMI); ok {
				t.Errorf("non-BIMI TXT was unexpectedly registered as a BIMI service")
			}
		}
	}
}

func TestBIMIFieldsAnalyzeAndString(t *testing.T) {
	cases := []string{
		"v=BIMI1;l=https://example.com/logo.svg",
		"v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem",
		"v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem;e=https://example.com/evidence",
	}
	for _, in := range cases {
		var f svcs.BIMIFields
		if err := f.Analyze(in); err != nil {
			t.Fatalf("Analyze(%q) failed: %v", in, err)
		}
		if got := f.String(); got != in {
			t.Errorf("String() = %q, want %q", got, in)
		}
	}
}

func TestBIMIFieldsAnalyzeRejectsBadInput(t *testing.T) {
	cases := []string{
		"l=https://example.com/logo.svg",
		"v=DMARC1;l=https://example.com/logo.svg",
		"v=BIMIx;l=https://example.com/logo.svg",
	}
	for _, in := range cases {
		var f svcs.BIMIFields
		if err := f.Analyze(in); err == nil {
			t.Errorf("Analyze(%q) should have failed", in)
		}
	}
}
