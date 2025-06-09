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

func TestPTRService(t *testing.T) {
	// Create a basic PTR record
	ptrTarget := "ptr.target.com."
	rr, err := dns.NewRR("4.3.2.1.in-addr.arpa. 3600 IN PTR " + ptrTarget)
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svcs.AnalyzeZone("in-addr.arpa.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s["4.3.2.1"]) != 1 {
		t.Fatalf("Expected 1 service, got %d: %v", len(s["4.3.2.1"]), s)
	}

	ptr, ok := s["4.3.2.1"][0].Service.(*svcs.PTR)
	if !ok {
		t.Fatalf("Expected service to be of type *PTR, got %T", s["4.3.2.1"][0].Service)
	}

	// Check number of resources
	if ptr.GetNbResources() != 1 {
		t.Errorf("GetNbResources() = %d; want 1", ptr.GetNbResources())
	}

	// Check GenComment
	if ptr.GenComment() != ptrTarget {
		t.Errorf("GenComment() = %q; want %q", ptr.GenComment(), ptrTarget)
	}

	// Check GetRecords output
	origin := "example.com."
	expectedFQDN := helpers.DomainFQDN(ptrTarget, origin)

	records, err := ptr.GetRecords("4.3.2.1.in-addr.arpa.", 3600, origin)
	if err != nil {
		t.Fatalf("GetRecords() failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	// Assert it is a PTR and was correctly transformed
	gotPTR, ok := records[0].(*dns.PTR)
	if !ok {
		t.Fatalf("Expected *dns.PTR, got %T", records[0])
	}

	if gotPTR.Ptr != expectedFQDN {
		t.Errorf("PTR target = %q; want %q", gotPTR.Ptr, expectedFQDN)
	}
}
