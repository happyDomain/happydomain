// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
)

func forsaleRR(t *testing.T, s string) happydns.Record {
	t.Helper()

	rr, err := dns.NewRR(s)
	if err != nil {
		t.Fatalf("dns.NewRR(%q) failed: %v", s, err)
	}

	return happydns.NewTXT(rr.(*dns.TXT))
}

func TestForSale(t *testing.T) {
	records := []happydns.Record{
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "v=FORSALE1;fcod=EXCO-S2lscm95IHdhcyBoZXJl"`),
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "v=FORSALE1;ftxt=Call for info."`),
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "v=FORSALE1;furi=https://example.com/fs"`),
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "v=FORSALE1;fval=USD750"`),
	}

	s, _, err := svc.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}
	if len(s[""]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s[""]))
	}

	forsale, ok := s[""][0].Service.(*svcs.ForSale)
	if !ok {
		t.Fatalf("Expected service to be of type *ForSale, got %T", s[""][0].Service)
	}

	if forsale.GetNbResources() != 4 {
		t.Errorf("GetNbResources = %d; want 4", forsale.GetNbResources())
	}

	if comment := forsale.GenComment(); comment != "USD 750" {
		t.Errorf("GenComment() = %q; want %q", comment, "USD 750")
	}

	recs, err := forsale.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}
	if len(recs) != 4 {
		t.Errorf("Expected 4 records, got %d", len(recs))
	}
	for _, r := range recs {
		if r.Header().Name != "_for-sale" {
			t.Errorf("Expected relative name %q, got %q", "_for-sale", r.Header().Name)
		}
		if _, ok := r.(*happydns.TXT); !ok {
			t.Errorf("Expected *happydns.TXT, got %T", r)
		}
	}
}

// TestForSaleOnSubdomain checks that a _for-sale node is recognized anywhere in
// the zone, not only at the apex.
func TestForSaleOnSubdomain(t *testing.T) {
	records := []happydns.Record{
		forsaleRR(t, `_for-sale.shop.example.com. 3600 IN TXT "v=FORSALE1;ftxt=Make an offer"`),
	}

	s, _, err := svc.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s["shop"]) != 1 {
		t.Fatalf("Expected 1 service under shop, got %d", len(s["shop"]))
	}

	forsale, ok := s["shop"][0].Service.(*svcs.ForSale)
	if !ok {
		t.Fatalf("Expected service to be of type *ForSale, got %T", s["shop"][0].Service)
	}

	if comment := forsale.GenComment(); comment != "Make an offer" {
		t.Errorf("GenComment() = %q; want %q", comment, "Make an offer")
	}
}

// TestForSaleWithoutVersion checks that RFC 10023 section 2.4 is honored: a
// _for-sale record without a valid version tag must not be claimed.
func TestForSaleWithoutVersion(t *testing.T) {
	records := []happydns.Record{
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "ftxt=this domain is for sale"`),
	}

	s, _, err := svc.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	for _, services := range s {
		for _, service := range services {
			if _, ok := service.Service.(*svcs.ForSale); ok {
				t.Fatalf("Expected no *ForSale service, got one for %q", service.Domain)
			}
		}
	}
}

// TestForSaleVersionOnly covers the "for sale, no detail" case.
func TestForSaleVersionOnly(t *testing.T) {
	records := []happydns.Record{
		forsaleRR(t, `_for-sale.example.com. 3600 IN TXT "v=FORSALE1;"`),
	}

	s, _, err := svc.AnalyzeZone("example.com.", records)
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	forsale, ok := s[""][0].Service.(*svcs.ForSale)
	if !ok {
		t.Fatalf("Expected service to be of type *ForSale, got %T", s[""][0].Service)
	}

	if comment := forsale.GenComment(); comment != "Domain for sale" {
		t.Errorf("GenComment() = %q; want %q", comment, "Domain for sale")
	}
}

func TestParseForSalePair(t *testing.T) {
	tests := []struct {
		txt     string
		tag     string
		value   string
		wantErr bool
	}{
		{`v=FORSALE1;fval=USD750`, "fval", "USD750", false},
		{`v=FORSALE1;ftxt=Call for info.`, "ftxt", "Call for info.", false},
		// A single space after the version tag is tolerated (RFC 10023, 3.6).
		{`v=FORSALE1; ftxt=Call for info.`, "ftxt", "Call for info.", false},
		{`v=FORSALE1;`, "", "", false},
		{`v=FORSALE1;furi=https://example.com/fs?a=b`, "furi", "https://example.com/fs?a=b", false},
		{`v=FORSALE2;ftxt=nope`, "", "", true},
		{`ftxt=no version`, "", "", true},
		{`v=FORSALE1;garbage`, "", "", true},
	}

	for _, tt := range tests {
		tag, value, err := svcs.ParseForSalePair(tt.txt)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseForSalePair(%q) error = %v; wantErr %v", tt.txt, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if tag != tt.tag || value != tt.value {
			t.Errorf("ParseForSalePair(%q) = (%q, %q); want (%q, %q)", tt.txt, tag, value, tt.tag, tt.value)
		}
	}
}

func TestParseForSalePrice(t *testing.T) {
	tests := []struct {
		value    string
		currency string
		amount   string
		wantErr  bool
	}{
		{"USD750", "USD", "750", false},
		{"EUR1234.56", "EUR", "1234.56", false},
		{"X1", "X", "1", false},
		{"750", "", "", true},
		{"USD", "", "", true},
		{"USD1.2.3", "", "", true},
		{"USD.5", "", "", true},
		{"USD5.", "", "", true},
		{"usd750", "", "", true},
	}

	for _, tt := range tests {
		price, err := svcs.ParseForSalePrice(tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseForSalePrice(%q) error = %v; wantErr %v", tt.value, err, tt.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if price.Currency != tt.currency || price.Amount != tt.amount {
			t.Errorf("ParseForSalePrice(%q) = %+v; want (%q, %q)", tt.value, price, tt.currency, tt.amount)
		}
	}
}

func TestForSaleFieldsAnalyze(t *testing.T) {
	fields := svcs.ForSaleFields{}

	for _, txt := range []string{
		`v=FORSALE1;fcod=EXCO-1`,
		`v=FORSALE1;fcod=EXCO-2`,
		`v=FORSALE1;ftxt=Call for info.`,
		`v=FORSALE1;furi=mailto:sales@example.com`,
		`v=FORSALE1;fval=USD750`,
		`v=FORSALE1;fxyz=future tag`,
		`v=FORSALE1;`,
	} {
		if err := fields.Analyze(txt); err != nil {
			t.Fatalf("Analyze(%q) failed: %v", txt, err)
		}
	}

	if len(fields.Codes) != 2 {
		t.Errorf("Codes = %v; want 2 entries", fields.Codes)
	}
	if len(fields.Texts) != 1 || fields.Texts[0] != "Call for info." {
		t.Errorf("Texts = %v; want [Call for info.]", fields.Texts)
	}
	if len(fields.URIs) != 1 || fields.URIs[0] != "mailto:sales@example.com" {
		t.Errorf("URIs = %v; want [mailto:sales@example.com]", fields.URIs)
	}
	if len(fields.Prices) != 1 || fields.Prices[0].String() != "USD 750" {
		t.Errorf("Prices = %v; want [USD 750]", fields.Prices)
	}
	if len(fields.Unknown) != 1 || fields.Unknown[0] != "fxyz=future tag" {
		t.Errorf("Unknown = %v; want [fxyz=future tag]", fields.Unknown)
	}
}
