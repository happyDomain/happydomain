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

func TestNewSPF(t *testing.T) {
	rr := &dns.SPF{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: []string{"v=spf1 ", "include:_spf.google.com ", "~all"},
	}

	spf := happydns.NewSPF(rr)
	expected := "v=spf1 include:_spf.google.com ~all"

	if spf.Txt != expected {
		t.Errorf("NewSPF() Txt = %q; want %q", spf.Txt, expected)
	}

	if spf.Hdr.Name != "example.com." {
		t.Errorf("NewSPF() Hdr.Name = %q; want %q", spf.Hdr.Name, "example.com.")
	}
}

func TestSPFToRR_Short(t *testing.T) {
	spf := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: "v=spf1 ~all",
	}

	rr := spf.ToRR().(*dns.SPF)
	if len(rr.Txt) != 1 || rr.Txt[0] != "v=spf1 ~all" {
		t.Errorf("ToRR() = %v; want [\"v=spf1 ~all\"]", rr.Txt)
	}
}

func TestSPFToRR_Long(t *testing.T) {
	longText := strings.Repeat("a", 700)
	spf := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "long.example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: longText,
	}

	rr := spf.ToRR().(*dns.SPF)
	expectedChunks := (len(longText) + happydns.TXT_SEGMENT_LEN - 1) / happydns.TXT_SEGMENT_LEN

	if len(rr.Txt) != expectedChunks {
		t.Fatalf("ToRR() produced %d chunks; want %d", len(rr.Txt), expectedChunks)
	}

	reassembled := strings.Join(rr.Txt, "")
	if reassembled != longText {
		t.Errorf("Reassembled text doesn't match original.\nGot: %q\nWant: %q", reassembled, longText)
	}
}

func TestSPFHeader(t *testing.T) {
	hdr := dns.RR_Header{Name: "spf.example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 7200}
	spf := &happydns.SPF{Hdr: hdr, Txt: "v=spf1"}

	got := spf.Header()
	if got.Name != hdr.Name || got.Rrtype != hdr.Rrtype || got.Ttl != hdr.Ttl {
		t.Errorf("Header() = %+v; want %+v", got, hdr)
	}
}

func TestSPFString(t *testing.T) {
	spf := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "spf.example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: "v=spf1 mx ~all",
	}

	rr := spf.ToRR()
	expected := rr.String()

	if spf.String() != expected {
		t.Errorf("String() = %q; want %q", spf.String(), expected)
	}
}

func TestSPFCopy(t *testing.T) {
	original := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: "v=spf1 include:example.com ~all",
	}

	copied := original.Copy().(*happydns.SPF)

	if copied.Txt != original.Txt {
		t.Errorf("Copy() Txt = %q; want %q", copied.Txt, original.Txt)
	}

	if copied.Hdr.Name != original.Hdr.Name {
		t.Errorf("Copy() Hdr.Name = %q; want %q", copied.Hdr.Name, original.Hdr.Name)
	}

	copied.Txt = "modified"
	if original.Txt == "modified" {
		t.Error("Copy() should create independent copy, but modifying copy affected original")
	}
}

func TestTXTCopy(t *testing.T) {
	original := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: "test text",
	}

	copied := original.Copy().(*happydns.TXT)

	if copied.Txt != original.Txt {
		t.Errorf("Copy() Txt = %q; want %q", copied.Txt, original.Txt)
	}

	if copied.Hdr.Name != original.Hdr.Name {
		t.Errorf("Copy() Hdr.Name = %q; want %q", copied.Hdr.Name, original.Hdr.Name)
	}

	copied.Txt = "modified"
	if original.Txt == "modified" {
		t.Error("Copy() should create independent copy, but modifying copy affected original")
	}
}

func TestSPFToRR_SegmentBoundary(t *testing.T) {
	exactlyOneSegment := strings.Repeat("a", happydns.TXT_SEGMENT_LEN)
	spf := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "boundary.example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: exactlyOneSegment,
	}

	rr := spf.ToRR().(*dns.SPF)
	if len(rr.Txt) != 1 {
		t.Errorf("ToRR() with exactly 255 chars produced %d segments; want 1", len(rr.Txt))
	}

	justOverOneSegment := strings.Repeat("a", happydns.TXT_SEGMENT_LEN+1)
	spf2 := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "boundary.example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: justOverOneSegment,
	}

	rr2 := spf2.ToRR().(*dns.SPF)
	if len(rr2.Txt) != 2 {
		t.Errorf("ToRR() with 256 chars produced %d segments; want 2", len(rr2.Txt))
	}
}

func TestTXTToRR_SegmentBoundary(t *testing.T) {
	exactlyOneSegment := strings.Repeat("a", happydns.TXT_SEGMENT_LEN)
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "boundary.example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: exactlyOneSegment,
	}

	rr := txt.ToRR().(*dns.TXT)
	if len(rr.Txt) != 1 {
		t.Errorf("ToRR() with exactly 255 chars produced %d segments; want 1", len(rr.Txt))
	}

	justOverOneSegment := strings.Repeat("a", happydns.TXT_SEGMENT_LEN+1)
	txt2 := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "boundary.example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: justOverOneSegment,
	}

	rr2 := txt2.ToRR().(*dns.TXT)
	if len(rr2.Txt) != 2 {
		t.Errorf("ToRR() with 256 chars produced %d segments; want 2", len(rr2.Txt))
	}
}

func TestSPFEmpty(t *testing.T) {
	spf := &happydns.SPF{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeSPF, Class: dns.ClassINET, Ttl: 3600},
		Txt: "",
	}

	rr := spf.ToRR().(*dns.SPF)
	if len(rr.Txt) != 0 {
		t.Errorf("ToRR() with empty text length = %d; want 0", len(rr.Txt))
	}
}

func TestTXTEmpty(t *testing.T) {
	txt := &happydns.TXT{
		Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600},
		Txt: "",
	}

	rr := txt.ToRR().(*dns.TXT)
	if len(rr.Txt) != 0 {
		t.Errorf("ToRR() with empty text length = %d; want 0", len(rr.Txt))
	}
}
