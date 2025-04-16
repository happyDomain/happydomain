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

package happydns_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

func TestNewTXT(t *testing.T) {
	rr := &dns.TXT{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: []string{"hello", "world"},
	}

	txt := happydns.NewTXT(rr)
	expected := "helloworld"

	if txt.Txt != expected {
		t.Errorf("NewTXT() Txt = %q; want %q", txt.Txt, expected)
	}

	if txt.Hdr.Name != "example.com." {
		t.Errorf("NewTXT() Hdr.Name = %q; want %q", txt.Hdr.Name, "example.com.")
	}
}

func TestToRR_Short(t *testing.T) {
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: "short text",
	}

	rr := txt.ToRR().(*dns.TXT)
	if len(rr.Txt) != 1 || rr.Txt[0] != "short text" {
		t.Errorf("ToRR() = %v; want [\"short text\"]", rr.Txt)
	}
}

func TestToRR_Long(t *testing.T) {
	longText := strings.Repeat("a", 700)
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "long.example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: longText,
	}

	rr := txt.ToRR().(*dns.TXT)
	expectedChunks := (len(longText) + happydns.TXT_SEGMENT_LEN - 1) / happydns.TXT_SEGMENT_LEN

	if len(rr.Txt) != expectedChunks {
		t.Fatalf("ToRR() produced %d chunks; want %d", len(rr.Txt), expectedChunks)
	}

	reassembled := strings.Join(rr.Txt, "")
	if reassembled != longText {
		t.Errorf("Reassembled text doesn't match original.\nGot: %q\nWant: %q", reassembled, longText)
	}
}

func TestHeader(t *testing.T) {
	hdr := dns.RR_Header{Name: "hdr.example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 123}
	txt := &happydns.TXT{Hdr: hdr, Txt: "testing"}

	got := txt.Header()
	if got.Name != hdr.Name || got.Rrtype != hdr.Rrtype || got.Ttl != hdr.Ttl {
		t.Errorf("Header() = %+v; want %+v", got, hdr)
	}
}

func TestString(t *testing.T) {
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "str.example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: "stringtest",
	}

	rr := txt.ToRR()
	expected := rr.String()

	if txt.String() != expected {
		t.Errorf("String() = %q; want %q", txt.String(), expected)
	}
}
