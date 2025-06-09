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

func TestCAAPolicy(t *testing.T) {
	// Simulate some CAA records
	rr1, err := dns.NewRR("example.com. 3600 IN CAA 0 issue \"letsencrypt.org\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	rr2, err := dns.NewRR("example.com. 3600 IN CAA 0 issuewild \"comodoca.com\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	rr3, err := dns.NewRR("example.com. 3600 IN CAA 0 iodef \"mailto:admin@example.com\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	rr4, err := dns.NewRR("example.com. 3600 IN CAA 0 issuemail \"sectigo.com\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{rr1, rr2, rr3, rr4})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s[""]))
	}

	policy, ok := s[""][0].Service.(*svcs.CAAPolicy)
	if !ok {
		t.Fatalf("Expected service to be of type *CAAPolicy, got %T", s[""][0].Service)
	}

	// Check the number of resources
	if policy.GetNbResources() != 4 {
		t.Errorf("GetNbResources = %d; want 4", policy.GetNbResources())
	}

	// Check the generated comment
	comment := policy.GenComment()

	expectedSubstrings := []string{
		"letsencrypt.org",
		"wildcard: comodoca.com",
		"S/MIME: sectigo.com",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(comment, substr) {
			t.Errorf("GenComment() missing %q in %q", substr, comment)
		}
	}

	// Test GetRecords
	recs, err := policy.GetRecords("example.com.", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(recs) != 4 {
		t.Errorf("Expected 4 records, got %d", len(recs))
	}
	for _, r := range recs {
		if _, ok := r.(*dns.CAA); !ok {
			t.Errorf("Expected *dns.CAA, got %T", r)
		}
	}
}
