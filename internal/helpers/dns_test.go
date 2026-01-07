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
