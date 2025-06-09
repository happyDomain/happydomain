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

func TestTLS_RPT(t *testing.T) {
	// Sample TLS-RPT TXT string with rua entries
	txtStr := "v=TLSRPTv1; rua=mailto:tlsrpt@example.com,mailto:security@example.org"

	// Create a DNS TXT record
	rr, err := dns.NewRR("_smtp._tls.example.com. 3600 IN TXT \"" + txtStr + "\"")
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

	tlsrpt, ok := s[""][0].Service.(*svcs.TLS_RPT)
	if !ok {
		t.Fatalf("Expected service to be of type *TLS_RPT, got %T", s[""][0].Service)
	}

	// Check the number of resources
	if tlsrpt.GetNbResources() != 1 {
		t.Errorf("GetNbResources = %d; want 1", tlsrpt.GetNbResources())
	}

	// Check the generated comment
	comment := tlsrpt.GenComment()
	for _, addr := range []string{"mailto:tlsrpt@example.com", "mailto:security@example.org"} {
		if !strings.Contains(comment, addr) {
			t.Errorf("GenComment() missing %q in %q", addr, comment)
		}
	}

	// Check that GetRecords returns exactly the wrapped TXT record
	records, err := tlsrpt.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if r, ok := records[0].(*happydns.TXT); !ok {
		t.Fatalf("Expected *happydns.TXT, got %T", records[0])
	} else if r.Txt != txtStr {
		t.Errorf("Expected TXT = %q, got %q", txtStr, r.Txt)
	}
}
