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

func TestMXs(t *testing.T) {
	// Create MX records
	mx1, err := dns.NewRR("example.com. 3600 IN MX 10 mail1.google.com.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	mx2, err := dns.NewRR("example.com. 3600 IN MX 20 mail2.google.com.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	mx3, err := dns.NewRR("example.com. 3600 IN MX 30 mx.mail.yahoo.com.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}
	mx4, err := dns.NewRR("example.com. 3600 IN MX 40 google.com.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{mx1, mx2, mx3, mx4})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s[""]))
	}

	policy, ok := s[""][0].Service.(*svcs.MXs)
	if !ok {
		t.Fatalf("Expected service to be of type *MXs, got %T", s[""][0].Service)
	}

	// Test GetNbResources
	if policy.GetNbResources() != 4 {
		t.Errorf("GetNbResources = %d; want 4", policy.GetNbResources())
	}

	// Test GenComment
	comment := policy.GenComment()
	expectedSubstrings := []string{
		"google.com.",
		"yahoo.com.",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(comment, substr) {
			t.Errorf("GenComment() missing %q in %q", substr, comment)
		}
	}

	// If multiple MXs are grouped under same root domain, we should see `×2` etc.
	if !strings.Contains(comment, "google.com.") {
		t.Errorf("Expected 'google.com.' in comment: %q", comment)
	}
	if !strings.Contains(comment, "yahoo.com.") {
		t.Errorf("Expected 'yahoo.com.' in comment: %q", comment)
	}
	if !strings.Contains(comment, "google.com. ×3") {
		t.Logf("Comment was: %s", comment)
		t.Errorf("Expected 'google.com. ×3' in comment (3 MX records under google.com), got: %q", comment)
	}

	// Test GetRecords
	origin := "example.com."
	records, err := policy.GetRecords("example.com.", 3600, origin)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	if len(records) != 4 {
		t.Errorf("Expected 4 records, got %d", len(records))
	}

	for _, rr := range records {
		mx, ok := rr.(*dns.MX)
		if !ok {
			t.Errorf("Expected *dns.MX, got %T", rr)
			continue
		}
		if !strings.HasSuffix(mx.Mx, ".") {
			t.Errorf("Expected MX exchange to be FQDN, got %q", mx.Mx)
		}
	}
}
