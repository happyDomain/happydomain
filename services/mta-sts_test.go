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

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestMTA_STS(t *testing.T) {
	txtValue := "v=STSv1; id=20240601T000000;"
	rr, err := dns.NewRR("_mta-sts.example.com. 3600 IN TXT \"" + txtValue + "\"")
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

	mta, ok := s[""][0].Service.(*svcs.MTA_STS)
	if !ok {
		t.Fatalf("Expected service to be of type *MTA_STS, got %T", s[""][0].Service)
	}

	// Check number of resources
	if mta.GetNbResources() != 1 {
		t.Errorf("GetNbResources() = %d; want 1", mta.GetNbResources())
	}

	// Check GenComment returns the correct id
	expectedID := "20240601T000000"
	if comment := mta.GenComment(); comment != expectedID {
		t.Errorf("GenComment() = %q; want %q", comment, expectedID)
	}

	// Check GetRecords returns the correct TXT record
	records, err := mta.GetRecords("example.com", 3600, "example.com")
	if err != nil {
		t.Fatalf("GetRecords() failed: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if txtrr, ok := records[0].(*happydns.TXT); !ok {
		t.Fatalf("Expected *happydns.TXT, got %T", records[0])
	} else if txtrr.Txt != txtValue {
		t.Errorf("Expected TXT = %q, got %q", txtValue, txtrr.Txt)
	}
}
