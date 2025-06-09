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

package svcs_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestDKIMRecord(t *testing.T) {
	txtStr := "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDs"
	name := "selector1._domainkey.example.com."

	rr, err := dns.NewRR(name + " 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service, got %d: %v", len(s[""]), s)
	}

	dkimRec, ok := s[""][0].Service.(*svcs.DKIMRecord)
	if !ok {
		t.Fatalf("Expected service to be of type *DKIMRecord, got %T", s[""][0].Service)
	}

	// Test Analyze
	parsed, err := dkimRec.Analyze()
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if parsed == nil {
		t.Fatal("Analyze returned nil DKIM struct")
	}

	// Test GenComment
	expectedComment := "selector1"
	if dkimRec.GenComment() != expectedComment {
		t.Errorf("GenComment() = %q; want %q", dkimRec.GenComment(), expectedComment)
	}

	// Test GetNbResources
	if dkimRec.GetNbResources() != 1 {
		t.Errorf("GetNbResources = %d; want 1", dkimRec.GetNbResources())
	}

	// Test GetRecords
	recs, err := dkimRec.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("Expected 1 record, got %d", len(recs))
	}
	if r, ok := recs[0].(*happydns.TXT); !ok {
		t.Errorf("Expected *happydns.TXT, got %T", recs[0])
	} else if r.Txt != txtStr {
		t.Errorf("Expected TXT value %q, got %q", txtStr, r.Txt)
	}
}

func TestDKIMRedirection(t *testing.T) {
	name := "selector1._domainkey.example.com."
	target := "selector1.dkim.amazonses.com."
	rr, err := dns.NewRR(name + " 3600 IN CNAME " + target)
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service, got %d: %v", len(s[""]), s)
	}

	dkimRedirect, ok := s[""][0].Service.(*svcs.DKIMRedirection)
	if !ok {
		t.Fatalf("Expected service to be of type *DKIMRedirection, got %T", s[""][0].Service)
	}

	// Test GenComment
	expectedComment := "selector1"
	if dkimRedirect.GenComment() != expectedComment {
		t.Errorf("GenComment() = %q; want %q", dkimRedirect.GenComment(), expectedComment)
	}

	// Test GetNbResources
	if dkimRedirect.GetNbResources() != 1 {
		t.Errorf("GetNbResources = %d; want 1", dkimRedirect.GetNbResources())
	}

	// Test GetRecords
	recs, err := dkimRedirect.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("Expected 1 record, got %d", len(recs))
	}
	if r, ok := recs[0].(*dns.CNAME); !ok {
		t.Errorf("Expected *dns.CNAME, got %T", recs[0])
	} else if !strings.HasSuffix(r.Target, ".") {
		t.Errorf("Expected FQDN target, got %q", r.Target)
	}
}
