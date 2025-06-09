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
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestNAPTR(t *testing.T) {
	// Create a NAPTR record manually
	rr, err := dns.NewRR(`example.com. 3600 IN NAPTR 10 100 "U" "E2U+sip" "!^.*$!sip:info@example.com!" _sip._udp.example.com.`)
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

	naptr, ok := s[""][0].Service.(*svcs.NAPTR)
	if !ok {
		t.Fatalf("Expected service to be of type *NAPTR, got %T", s[""][0].Service)
	}

	// Check resource count
	if got := naptr.GetNbResources(); got != 1 {
		t.Errorf("GetNbResources() = %d; want 1", got)
	}

	// Check GenComment returns the service string
	if got := naptr.GenComment(); got != "E2U+sip" {
		t.Errorf("GenComment() = %q; want %q", got, "E2U+sip")
	}

	// Check GetRecords returns the record with correct Replacement FQDN
	records, err := naptr.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	r, ok := records[0].(*dns.NAPTR)
	if !ok {
		t.Fatalf("Expected *dns.NAPTR, got %T", records[0])
	}

	expectedReplacement := helpers.DomainFQDN("_sip._udp.example.com.", "example.com.")
	if r.Replacement != expectedReplacement {
		t.Errorf("Replacement = %q; want %q", r.Replacement, expectedReplacement)
	}
}
