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

package helpers

import (
	"reflect"
	"testing"

	"github.com/miekg/dns"
)

func TestDomainFQDN(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		origin    string
		expected  string
	}{
		{
			name:      "already FQDN",
			subdomain: "www.example.com.",
			origin:    "example.com.",
			expected:  "www.example.com.",
		},
		{
			name:      "relative subdomain",
			subdomain: "www",
			origin:    "example.com.",
			expected:  "www.example.com.",
		},
		{
			name:      "empty subdomain",
			subdomain: "",
			origin:    "example.com.",
			expected:  "example.com.",
		},
		{
			name:      "@ subdomain",
			subdomain: "@",
			origin:    "example.com.",
			expected:  "example.com.",
		},
		{
			name:      "multi-level subdomain",
			subdomain: "api.v1",
			origin:    "example.com.",
			expected:  "api.v1.example.com.",
		},
		{
			name:      "origin without trailing dot",
			subdomain: "www",
			origin:    "example.com",
			expected:  "www.example.com",
		},
		{
			name:      "empty with origin without trailing dot",
			subdomain: "",
			origin:    "example.com",
			expected:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DomainFQDN(tt.subdomain, tt.origin)
			if result != tt.expected {
				t.Errorf("DomainFQDN(%q, %q) = %q, want %q", tt.subdomain, tt.origin, result, tt.expected)
			}
		})
	}
}

func TestDomainJoin(t *testing.T) {
	tests := []struct {
		name     string
		domains  []string
		expected string
	}{
		{
			name:     "two domains",
			domains:  []string{"www", "example.com"},
			expected: "www.example.com",
		},
		{
			name:     "three domains",
			domains:  []string{"api", "v1", "example.com"},
			expected: "api.v1.example.com",
		},
		{
			name:     "single domain",
			domains:  []string{"example.com"},
			expected: "example.com",
		},
		{
			name:     "empty domain in middle",
			domains:  []string{"www", "", "example.com"},
			expected: "www.example.com",
		},
		{
			name:     "@ symbol stops joining",
			domains:  []string{"www", "@", "example.com"},
			expected: "www",
		},
		{
			name:     "FQDN stops joining",
			domains:  []string{"www", "example.com.", "ignored.com"},
			expected: "www.example.com.",
		},
		{
			name:     "all empty",
			domains:  []string{"", "", ""},
			expected: "",
		},
		{
			name:     "@ at start",
			domains:  []string{"@", "example.com"},
			expected: "",
		},
		{
			name:     "empty slice",
			domains:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DomainJoin(tt.domains...)
			if result != tt.expected {
				t.Errorf("DomainJoin(%v) = %q, want %q", tt.domains, result, tt.expected)
			}
		})
	}
}

func TestDomainRelative(t *testing.T) {
	tests := []struct {
		name      string
		subdomain string
		origin    string
		expected  string
	}{
		{
			name:      "full FQDN relative to origin",
			subdomain: "www.example.com.",
			origin:    "example.com.",
			expected:  "www",
		},
		{
			name:      "full FQDN without trailing dot relative to origin",
			subdomain: "www.example.com",
			origin:    "example.com.",
			expected:  "www.example.com",
		},
		{
			name:      "origin without trailing dot",
			subdomain: "www.example.com.",
			origin:    "example.com",
			expected:  "www",
		},
		{
			name:      "subdomain equals origin",
			subdomain: "example.com.",
			origin:    "example.com.",
			expected:  "@",
		},
		{
			name:      "subdomain equals origin without dots",
			subdomain: "example.com",
			origin:    "example.com",
			expected:  "example.com",
		},
		{
			name:      "not relative to origin",
			subdomain: "www.other.com.",
			origin:    "example.com.",
			expected:  "www.other.com.",
		},
		{
			name:      "multi-level subdomain",
			subdomain: "api.v1.example.com.",
			origin:    "example.com.",
			expected:  "api.v1",
		},
		{
			name:      "already relative",
			subdomain: "www",
			origin:    "example.com.",
			expected:  "www",
		},
		{
			name:      "empty becomes @",
			subdomain: "",
			origin:    "example.com.",
			expected:  "@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DomainRelative(tt.subdomain, tt.origin)
			if result != tt.expected {
				t.Errorf("DomainRelative(%q, %q) = %q, want %q", tt.subdomain, tt.origin, result, tt.expected)
			}
		})
	}
}

func TestNewRecord(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		rrtype   string
		ttl      uint32
		origin   string
		validate func(t *testing.T, rr dns.RR)
	}{
		{
			name:   "A record with relative domain",
			domain: "www",
			rrtype: "A",
			ttl:    3600,
			origin: "example.com.",
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Name != "www.example.com." {
					t.Errorf("Expected Name to be 'www.example.com.', got %q", rr.Header().Name)
				}
				if rr.Header().Rrtype != dns.TypeA {
					t.Errorf("Expected Rrtype to be TypeA, got %d", rr.Header().Rrtype)
				}
				if rr.Header().Ttl != 3600 {
					t.Errorf("Expected TTL to be 3600, got %d", rr.Header().Ttl)
				}
				if rr.Header().Class != dns.ClassINET {
					t.Errorf("Expected Class to be ClassINET, got %d", rr.Header().Class)
				}
			},
		},
		{
			name:   "AAAA record with absolute domain",
			domain: "www.example.com.",
			rrtype: "AAAA",
			ttl:    7200,
			origin: "example.com.",
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Name != "www.example.com." {
					t.Errorf("Expected Name to be 'www.example.com.', got %q", rr.Header().Name)
				}
				if rr.Header().Rrtype != dns.TypeAAAA {
					t.Errorf("Expected Rrtype to be TypeAAAA, got %d", rr.Header().Rrtype)
				}
				if rr.Header().Ttl != 7200 {
					t.Errorf("Expected TTL to be 7200, got %d", rr.Header().Ttl)
				}
			},
		},
		{
			name:   "MX record",
			domain: "@",
			rrtype: "MX",
			ttl:    1800,
			origin: "example.com.",
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Name != "example.com." {
					t.Errorf("Expected Name to be 'example.com.', got %q", rr.Header().Name)
				}
				if rr.Header().Rrtype != dns.TypeMX {
					t.Errorf("Expected Rrtype to be TypeMX, got %d", rr.Header().Rrtype)
				}
			},
		},
		{
			name:   "TXT record",
			domain: "_dmarc",
			rrtype: "TXT",
			ttl:    300,
			origin: "example.com.",
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Name != "_dmarc.example.com." {
					t.Errorf("Expected Name to be '_dmarc.example.com.', got %q", rr.Header().Name)
				}
				if rr.Header().Rrtype != dns.TypeTXT {
					t.Errorf("Expected Rrtype to be TypeTXT, got %d", rr.Header().Rrtype)
				}
			},
		},
		{
			name:   "CNAME record",
			domain: "www",
			rrtype: "CNAME",
			ttl:    600,
			origin: "example.com.",
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Rrtype != dns.TypeCNAME {
					t.Errorf("Expected Rrtype to be TypeCNAME, got %d", rr.Header().Rrtype)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewRecord(tt.domain, tt.rrtype, tt.ttl, tt.origin)
			if result == nil {
				t.Fatal("NewRecord returned nil")
			}
			if dnsrr, ok := result.(dns.RR); ok {
				tt.validate(t, dnsrr)
			} else {
				t.Error("Result is not a dns.RR")
			}
		})
	}
}

func TestRRRelative(t *testing.T) {
	origin := "example.com."

	tests := []struct {
		name     string
		input    dns.RR
		origin   string
		validate func(t *testing.T, rr dns.RR)
	}{
		{
			name: "A record",
			input: &dns.A{
				Hdr: dns.RR_Header{
					Name:   "www.example.com.",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				A: []byte{192, 0, 2, 1},
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				if rr.Header().Name != "www" {
					t.Errorf("Expected Name to be 'www', got %q", rr.Header().Name)
				}
			},
		},
		{
			name: "NS record",
			input: &dns.NS{
				Hdr: dns.RR_Header{
					Name:   "example.com.",
					Rrtype: dns.TypeNS,
					Class:  dns.ClassINET,
					Ttl:    86400,
				},
				Ns: "ns1.example.com.",
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				ns := rr.(*dns.NS)
				if ns.Header().Name != "" {
					t.Errorf("Expected Name to be '', got %q", ns.Header().Name)
				}
				if ns.Ns != "ns1" {
					t.Errorf("Expected Ns to be 'ns1', got %q", ns.Ns)
				}
			},
		},
		{
			name: "MX record",
			input: &dns.MX{
				Hdr: dns.RR_Header{
					Name:   "example.com.",
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    1800,
				},
				Preference: 10,
				Mx:         "mail.example.com.",
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				mx := rr.(*dns.MX)
				if mx.Mx != "mail" {
					t.Errorf("Expected Mx to be 'mail', got %q", mx.Mx)
				}
			},
		},
		{
			name: "CNAME record",
			input: &dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   "www.example.com.",
					Rrtype: dns.TypeCNAME,
					Class:  dns.ClassINET,
					Ttl:    600,
				},
				Target: "target.example.com.",
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				cname := rr.(*dns.CNAME)
				if cname.Target != "target" {
					t.Errorf("Expected Target to be 'target', got %q", cname.Target)
				}
			},
		},
		{
			name: "SRV record",
			input: &dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "_http._tcp.example.com.",
					Rrtype: dns.TypeSRV,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Priority: 10,
				Weight:   60,
				Port:     80,
				Target:   "server.example.com.",
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				srv := rr.(*dns.SRV)
				if srv.Target != "server" {
					t.Errorf("Expected Target to be 'server', got %q", srv.Target)
				}
				if srv.Header().Name != "_http._tcp" {
					t.Errorf("Expected Name to be '_http._tcp', got %q", srv.Header().Name)
				}
			},
		},
		{
			name: "PTR record",
			input: &dns.PTR{
				Hdr: dns.RR_Header{
					Name:   "1.2.0.192.in-addr.arpa.",
					Rrtype: dns.TypePTR,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Ptr: "www.example.com.",
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				ptr := rr.(*dns.PTR)
				if ptr.Ptr != "www" {
					t.Errorf("Expected Ptr to be 'www', got %q", ptr.Ptr)
				}
			},
		},
		{
			name: "SOA record",
			input: &dns.SOA{
				Hdr: dns.RR_Header{
					Name:   "example.com.",
					Rrtype: dns.TypeSOA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Ns:      "ns1.example.com.",
				Mbox:    "admin.example.com.",
				Serial:  2024010101,
				Refresh: 3600,
				Retry:   600,
				Expire:  604800,
				Minttl:  86400,
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				soa := rr.(*dns.SOA)
				if soa.Ns != "ns1" {
					t.Errorf("Expected Ns to be 'ns1', got %q", soa.Ns)
				}
				if soa.Mbox != "admin" {
					t.Errorf("Expected Mbox to be 'admin', got %q", soa.Mbox)
				}
			},
		},
		{
			name: "SOA record 2",
			input: &dns.SOA{
				Hdr: dns.RR_Header{
					Name:   "example.com.",
					Rrtype: dns.TypeSOA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Ns:      "ns1.",
				Mbox:    "hostmaster.",
				Serial:  2024010101,
				Refresh: 3600,
				Retry:   600,
				Expire:  604800,
				Minttl:  86400,
			},
			origin: origin,
			validate: func(t *testing.T, rr dns.RR) {
				soa := rr.(*dns.SOA)
				if soa.Ns != "ns1." {
					t.Errorf("Expected Ns to be 'ns1.', got %q", soa.Ns)
				}
				if soa.Mbox != "hostmaster." {
					t.Errorf("Expected Mbox to be 'hostmaster.', got %q", soa.Mbox)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RRRelative(tt.input, tt.origin)
			if result == nil {
				t.Fatal("RRRelative returned nil")
			}
			tt.validate(t, result.(dns.RR))
		})
	}
}

func TestCopyRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    dns.RR
		validate func(t *testing.T, original, copy dns.RR)
	}{
		{
			name: "A record",
			input: &dns.A{
				Hdr: dns.RR_Header{
					Name:   "www.example.com.",
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				A: []byte{192, 0, 2, 1},
			},
			validate: func(t *testing.T, original, copy dns.RR) {
				origA := original.(*dns.A)
				copyA := copy.(*dns.A)
				if origA.A.String() != copyA.A.String() {
					t.Errorf("Expected IP to match, original: %s, copy: %s", origA.A.String(), copyA.A.String())
				}
				if origA.Header().Name != copyA.Header().Name {
					t.Errorf("Expected Name to match")
				}
				// Verify it's a deep copy by modifying the copy
				copyA.A = []byte{192, 0, 2, 2}
				if origA.A.String() == copyA.A.String() {
					t.Error("Copy is not deep - modifying copy affected original")
				}
			},
		},
		{
			name: "MX record",
			input: &dns.MX{
				Hdr: dns.RR_Header{
					Name:   "example.com.",
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    1800,
				},
				Preference: 10,
				Mx:         "mail.example.com.",
			},
			validate: func(t *testing.T, original, copy dns.RR) {
				origMX := original.(*dns.MX)
				copyMX := copy.(*dns.MX)
				if origMX.Preference != copyMX.Preference {
					t.Errorf("Expected Preference to match, original: %d, copy: %d", origMX.Preference, copyMX.Preference)
				}
				if origMX.Mx != copyMX.Mx {
					t.Errorf("Expected Mx to match, original: %s, copy: %s", origMX.Mx, copyMX.Mx)
				}
				// Verify it's a deep copy
				copyMX.Mx = "mail2.example.com."
				if origMX.Mx == copyMX.Mx {
					t.Error("Copy is not deep - modifying copy affected original")
				}
			},
		},
		{
			name: "TXT record",
			input: &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   "_dmarc.example.com.",
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				Txt: []string{"v=DMARC1", "p=none"},
			},
			validate: func(t *testing.T, original, copy dns.RR) {
				origTXT := original.(*dns.TXT)
				copyTXT := copy.(*dns.TXT)
				if !reflect.DeepEqual(origTXT.Txt, copyTXT.Txt) {
					t.Errorf("Expected Txt to match, original: %v, copy: %v", origTXT.Txt, copyTXT.Txt)
				}
				// Verify it's a deep copy
				copyTXT.Txt[0] = "v=DMARC2"
				if origTXT.Txt[0] == copyTXT.Txt[0] {
					t.Error("Copy is not deep - modifying copy affected original")
				}
			},
		},
		{
			name: "SRV record",
			input: &dns.SRV{
				Hdr: dns.RR_Header{
					Name:   "_http._tcp.example.com.",
					Rrtype: dns.TypeSRV,
					Class:  dns.ClassINET,
					Ttl:    3600,
				},
				Priority: 10,
				Weight:   60,
				Port:     80,
				Target:   "server.example.com.",
			},
			validate: func(t *testing.T, original, copy dns.RR) {
				origSRV := original.(*dns.SRV)
				copySRV := copy.(*dns.SRV)
				if origSRV.Priority != copySRV.Priority ||
					origSRV.Weight != copySRV.Weight ||
					origSRV.Port != copySRV.Port ||
					origSRV.Target != copySRV.Target {
					t.Error("SRV record fields do not match")
				}
				// Verify it's a deep copy
				copySRV.Target = "server2.example.com."
				if origSRV.Target == copySRV.Target {
					t.Error("Copy is not deep - modifying copy affected original")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CopyRecord(tt.input)
			if result == nil {
				t.Fatal("CopyRecord returned nil")
			}
			if dnsrr, ok := result.(dns.RR); ok {
				tt.validate(t, tt.input, dnsrr)
			} else {
				t.Error("Result is not a dns.RR")
			}
		})
	}
}
