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

func TestDMARCService(t *testing.T) {
	// Example DMARC record
	txtStr := "v=DMARC1; p=reject; adkim=s; aspf=s; pct=75"
	rr, err := dns.NewRR("_dmarc.example.com. 3600 IN TXT \"" + txtStr + "\"")
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
		t.Fatalf("Expected 1 service, got %d", len(s[""]))
	}

	dmarcSvc, ok := s[""][0].Service.(*svcs.DMARC)
	if !ok {
		t.Fatalf("Expected service to be of type *CNAME, got %T", s[""][0].Service)
	}

	// Check number of resources
	if n := dmarcSvc.GetNbResources(); n != 1 {
		t.Errorf("GetNbResources() = %d; want 1", n)
	}

	// Check comment generation
	comment := dmarcSvc.GenComment()

	if !strings.Contains(comment, "strict") {
		t.Errorf("GenComment() missing 'strict': %q", comment)
	}
	if !strings.Contains(comment, "reject") {
		t.Errorf("GenComment() missing 'reject': %q", comment)
	}
	if !strings.Contains(comment, "75") {
		t.Errorf("GenComment() missing percent value: %q", comment)
	}
	if !strings.Contains(comment, " %") && !strings.Contains(comment, "%") {
		t.Errorf("GenComment() missing percent symbol: %q", comment)
	}

	// Check GetRecords
	records, err := dmarcSvc.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}
	if rec, ok := records[0].(*happydns.TXT); !ok {
		t.Errorf("Expected TXT record, got %T", records[0])
	} else if rec.Txt != txtStr {
		t.Errorf("Expected TXT = %q, got %q", txtStr, rec.Txt)
	}
}
