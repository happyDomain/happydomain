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

	"git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func TestTXTAnalyze(t *testing.T) {
	//txtStr := "v=spf1 include:_spf.example.com ~all"
	txtStr := "Ab perspiciatis in dignissimos repudiandae quo. Neque qui sunt quo voluptatum deserunt qui commodi rerum. Officia eaque delectus possimus aut sit ipsum ut laboriosam."
	rr, err := dns.NewRR("test.example.com. 3600 IN TXT \"" + txtStr + "\"")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	txt := happydns.NewTXT(rr.(*dns.TXT))

	s, _, err := svcs.AnalyzeZone("example.com.", []happydns.Record{txt})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s["test"]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s["test"]))
	}

	txtSvc, ok := s["test"][0].Service.(*svcs.TXT)
	if !ok {
		t.Fatalf("Expected service to be of type *TXT, got %T", s["test"][0].Service)
	}

	if txtSvc.GenComment() != txtStr {
		t.Errorf("GenComment() = %q; want %q", txtSvc.GenComment(), txtStr)
	}

	// Try to pretty-print
	records, err := service.NewListRecordsUsecase().List(&happydns.Domain{DomainName: "example.com"}, &happydns.Zone{}, s["test"][0])
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 output record, got %d", len(records))
	}

	if txtrr, ok := records[0].(*happydns.TXT); !ok {
		t.Fatalf("Expected TXT record, got %s", dns.TypeToString[records[0].Header().Rrtype])
	} else if txtrr.Txt != txtStr {
		t.Fatalf("Expected %q, got %q", txtStr, txtrr.Txt)
	}
}
