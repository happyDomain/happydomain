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

package adapter

import (
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

func TestLibdnsToHappyDNS_A(t *testing.T) {
	rec := libdns.Address{}
	rec.Name = "www"
	rec.TTL = 300 * time.Second
	rec.IP = mustParseAddr("192.0.2.1")

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header().Name != "www.example.com." {
		t.Errorf("expected name www.example.com., got %s", result.Header().Name)
	}
	if result.Header().Rrtype != dns.TypeA {
		t.Errorf("expected type A, got %s", dns.TypeToString[result.Header().Rrtype])
	}
	if result.Header().Ttl != 300 {
		t.Errorf("expected TTL 300, got %d", result.Header().Ttl)
	}
}

func TestLibdnsToHappyDNS_AAAA(t *testing.T) {
	rec := libdns.Address{}
	rec.Name = "@"
	rec.TTL = 600 * time.Second
	rec.IP = mustParseAddr("2001:db8::1")

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header().Name != "example.com." {
		t.Errorf("expected name example.com., got %s", result.Header().Name)
	}
	if result.Header().Rrtype != dns.TypeAAAA {
		t.Errorf("expected type AAAA, got %s", dns.TypeToString[result.Header().Rrtype])
	}
}

func TestLibdnsToHappyDNS_TXT(t *testing.T) {
	rec := libdns.TXT{
		Name: "@",
		TTL:  300 * time.Second,
		Text: "v=spf1 include:_spf.google.com ~all",
	}

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txt, ok := result.(*happydns.TXT)
	if !ok {
		t.Fatalf("expected *happydns.TXT, got %T", result)
	}

	if txt.Txt != "v=spf1 include:_spf.google.com ~all" {
		t.Errorf("expected TXT value 'v=spf1 include:_spf.google.com ~all', got %q", txt.Txt)
	}
	if txt.Hdr.Name != "example.com." {
		t.Errorf("expected name example.com., got %s", txt.Hdr.Name)
	}
}

func TestLibdnsToHappyDNS_CNAME(t *testing.T) {
	rec := libdns.CNAME{
		Name:   "www",
		TTL:    3600 * time.Second,
		Target: "example.com.",
	}

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header().Rrtype != dns.TypeCNAME {
		t.Errorf("expected type CNAME, got %s", dns.TypeToString[result.Header().Rrtype])
	}
}

func TestLibdnsToHappyDNS_MX(t *testing.T) {
	rec := libdns.MX{
		Name:       "@",
		TTL:        3600 * time.Second,
		Preference: 10,
		Target:     "mail.example.com.",
	}

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Header().Rrtype != dns.TypeMX {
		t.Errorf("expected type MX, got %s", dns.TypeToString[result.Header().Rrtype])
	}
}

func TestHappyDNSToLibdns_A(t *testing.T) {
	rr, _ := dns.NewRR("www.example.com. 300 IN A 192.0.2.1")

	result := happyDNSRecordToLibdnsRR(rr, "example.com.")

	if result.Name != "www" {
		t.Errorf("expected name 'www', got %q", result.Name)
	}
	if result.Type != "A" {
		t.Errorf("expected type A, got %s", result.Type)
	}
	if result.TTL != 300*time.Second {
		t.Errorf("expected TTL 300s, got %v", result.TTL)
	}
	if result.Data != "192.0.2.1" {
		t.Errorf("expected data '192.0.2.1', got %q", result.Data)
	}
}

func TestHappyDNSToLibdns_TXT(t *testing.T) {
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{
			Name:   "example.com.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		Txt: "v=spf1 include:_spf.google.com ~all",
	}

	result := happyDNSRecordToLibdnsRR(txt, "example.com.")

	if result.Name != "@" {
		t.Errorf("expected name '@', got %q", result.Name)
	}
	if result.Type != "TXT" {
		t.Errorf("expected type TXT, got %s", result.Type)
	}
	if result.Data != "v=spf1 include:_spf.google.com ~all" {
		t.Errorf("expected data 'v=spf1 include:_spf.google.com ~all', got %q", result.Data)
	}
}

func TestHappyDNSToLibdns_Apex(t *testing.T) {
	rr, _ := dns.NewRR("example.com. 300 IN A 192.0.2.1")

	result := happyDNSRecordToLibdnsRR(rr, "example.com.")

	if result.Name != "@" {
		t.Errorf("expected name '@', got %q", result.Name)
	}
}

func TestRoundTrip_A(t *testing.T) {
	original := libdns.Address{}
	original.Name = "www"
	original.TTL = 300 * time.Second
	original.IP = mustParseAddr("192.0.2.1")

	zone := "example.com."

	hdRecord, err := libdnsToHappyDNSRecord(original, zone)
	if err != nil {
		t.Fatalf("unexpected error converting to happydns: %v", err)
	}

	roundtripped := happyDNSRecordToLibdnsRR(hdRecord, zone)

	origRR := original.RR()
	if roundtripped.Name != origRR.Name {
		t.Errorf("name mismatch: got %q, want %q", roundtripped.Name, origRR.Name)
	}
	if roundtripped.Type != origRR.Type {
		t.Errorf("type mismatch: got %q, want %q", roundtripped.Type, origRR.Type)
	}
	if roundtripped.TTL != origRR.TTL {
		t.Errorf("TTL mismatch: got %v, want %v", roundtripped.TTL, origRR.TTL)
	}
	if roundtripped.Data != origRR.Data {
		t.Errorf("data mismatch: got %q, want %q", roundtripped.Data, origRR.Data)
	}
}

func TestRoundTrip_TXT(t *testing.T) {
	original := libdns.TXT{
		Name: "test",
		TTL:  600 * time.Second,
		Text: "hello world with spaces and special chars: @#$%",
	}

	zone := "example.com."

	hdRecord, err := libdnsToHappyDNSRecord(original, zone)
	if err != nil {
		t.Fatalf("unexpected error converting to happydns: %v", err)
	}

	txt, ok := hdRecord.(*happydns.TXT)
	if !ok {
		t.Fatalf("expected *happydns.TXT, got %T", hdRecord)
	}
	if txt.Txt != original.Text {
		t.Errorf("TXT text mismatch after first conversion: got %q, want %q", txt.Txt, original.Text)
	}

	roundtripped := happyDNSRecordToLibdnsRR(hdRecord, zone)

	origRR := original.RR()
	if roundtripped.Name != origRR.Name {
		t.Errorf("name mismatch: got %q, want %q", roundtripped.Name, origRR.Name)
	}
	if roundtripped.Type != origRR.Type {
		t.Errorf("type mismatch: got %q, want %q", roundtripped.Type, origRR.Type)
	}
	if roundtripped.Data != origRR.Data {
		t.Errorf("data mismatch: got %q, want %q", roundtripped.Data, origRR.Data)
	}
}

func TestLibdnsToHappyDNS_TXT_QuotedData(t *testing.T) {
	// Some libdns providers (e.g. PowerDNS) return TXT data in RFC1035 presentation format.
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{"simple quoted", `"some-acme-challenge-value"`, "some-acme-challenge-value"},
		{"escaped quote", `"foo\"bar"`, `foo"bar`},
		{"escaped backslash", `"foo\\bar"`, `foo\bar`},
		{"multi-chunk", `"chunk1" "chunk2"`, "chunk1chunk2"},
		{"unquoted passthrough", "v=spf1 ~all", "v=spf1 ~all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := libdns.RR{
				Name: "_acme-challenge",
				TTL:  3600 * time.Second,
				Type: "TXT",
				Data: tt.data,
			}

			result, err := libdnsToHappyDNSRecord(rec, "example.com.")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			txt, ok := result.(*happydns.TXT)
			if !ok {
				t.Fatalf("expected *happydns.TXT, got %T", result)
			}

			if txt.Txt != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, txt.Txt)
			}
		})
	}
}

func TestLibdnsToHappyDNS_TXT_UnquotedData(t *testing.T) {
	// libdns.TXT returns raw unquoted text — should pass through unchanged.
	rec := libdns.TXT{
		Name: "@",
		TTL:  300 * time.Second,
		Text: "v=spf1 ~all",
	}

	result, err := libdnsToHappyDNSRecord(rec, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	txt, ok := result.(*happydns.TXT)
	if !ok {
		t.Fatalf("expected *happydns.TXT, got %T", result)
	}

	if txt.Txt != "v=spf1 ~all" {
		t.Errorf("expected unquoted TXT value, got %q", txt.Txt)
	}
}

func TestExtractRdata(t *testing.T) {
	tests := []struct {
		input  string
		rrType string
		want   string
	}{
		{"www.example.com.\t300\tIN\tA\t192.0.2.1", "A", "192.0.2.1"},
		{"example.com.\t3600\tIN\tMX\t10 mail.example.com.", "MX", "10 mail.example.com."},
		{"example.com.\t300\tIN\tAAAA\t2001:db8::1", "AAAA", "2001:db8::1"},
	}

	for _, tt := range tests {
		got := extractRdata(tt.input, tt.rrType)
		if got != tt.want {
			t.Errorf("extractRdata(%q, %q) = %q, want %q", tt.input, tt.rrType, got, tt.want)
		}
	}
}

func mustParseAddr(s string) netip.Addr {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		panic(err)
	}
	return addr
}
