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

func TestCNAME(t *testing.T) {
	// Create a CNAME DNS record
	rr, err := dns.NewRR("www.example.com. 3600 IN CNAME target.example.org.")
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

	if len(s["www"]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s["www"]))
	}

	cnameSvc, ok := s["www"][0].Service.(*svcs.CNAME)
	if !ok {
		t.Fatalf("Expected service to be of type *CNAME, got %T", s["www"][0].Service)
	}

	// Test GetNbResources always returns 1
	if cnameSvc.GetNbResources() != 1 {
		t.Errorf("GetNbResources() = %d; want 1", cnameSvc.GetNbResources())
	}

	// Test GenComment returns the correct target
	if cnameSvc.GenComment() != "target.example.org." {
		t.Errorf("GenComment() = %q; want %q", cnameSvc.GenComment(), "target.example.org.")
	}

	// Test GetRecords
	records, err := cnameSvc.GetRecords("www.example.com.", 3600, "example.org.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	cnameRecord, ok := records[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("Expected *dns.CNAME, got %T", records[0])
	}

	// The target should be fully qualified by helpers.DomainFQDN
	expectedTarget := helpers.DomainFQDN(cnameRecord.Target, "example.org.")
	if cnameRecord.Target != expectedTarget {
		t.Errorf("CNAME target = %q; want %q", cnameRecord.Target, expectedTarget)
	}
}
