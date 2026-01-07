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

func TestIsDNSSECType(t *testing.T) {
	tests := []struct {
		name     string
		rrtype   uint16
		expected bool
	}{
		// DNSSEC types - should return true
		{
			name:     "NSEC type",
			rrtype:   dns.TypeNSEC,
			expected: true,
		},
		{
			name:     "NSEC3 type",
			rrtype:   dns.TypeNSEC3,
			expected: true,
		},
		{
			name:     "NSEC3PARAM type",
			rrtype:   dns.TypeNSEC3PARAM,
			expected: true,
		},
		{
			name:     "DNSKEY type",
			rrtype:   dns.TypeDNSKEY,
			expected: true,
		},
		{
			name:     "RRSIG type",
			rrtype:   dns.TypeRRSIG,
			expected: true,
		},
		// Common non-DNSSEC types - should return false
		{
			name:     "A type",
			rrtype:   dns.TypeA,
			expected: false,
		},
		{
			name:     "AAAA type",
			rrtype:   dns.TypeAAAA,
			expected: false,
		},
		{
			name:     "CNAME type",
			rrtype:   dns.TypeCNAME,
			expected: false,
		},
		{
			name:     "MX type",
			rrtype:   dns.TypeMX,
			expected: false,
		},
		{
			name:     "TXT type",
			rrtype:   dns.TypeTXT,
			expected: false,
		},
		{
			name:     "NS type",
			rrtype:   dns.TypeNS,
			expected: false,
		},
		{
			name:     "SOA type",
			rrtype:   dns.TypeSOA,
			expected: false,
		},
		{
			name:     "PTR type",
			rrtype:   dns.TypePTR,
			expected: false,
		},
		{
			name:     "SRV type",
			rrtype:   dns.TypeSRV,
			expected: false,
		},
		{
			name:     "CAA type",
			rrtype:   dns.TypeCAA,
			expected: false,
		},
		// Other DNSSEC-related types that are NOT auto-generated
		{
			name:     "DS type",
			rrtype:   dns.TypeDS,
			expected: false,
		},
		{
			name:     "CDS type",
			rrtype:   dns.TypeCDS,
			expected: false,
		},
		{
			name:     "CDNSKEY type",
			rrtype:   dns.TypeCDNSKEY,
			expected: false,
		},
		{
			name:     "DLV type",
			rrtype:   dns.TypeDLV,
			expected: false,
		},
		// Additional common types
		{
			name:     "HINFO type",
			rrtype:   dns.TypeHINFO,
			expected: false,
		},
		{
			name:     "MINFO type",
			rrtype:   dns.TypeMINFO,
			expected: false,
		},
		{
			name:     "RP type",
			rrtype:   dns.TypeRP,
			expected: false,
		},
		{
			name:     "AFSDB type",
			rrtype:   dns.TypeAFSDB,
			expected: false,
		},
		{
			name:     "X25 type",
			rrtype:   dns.TypeX25,
			expected: false,
		},
		{
			name:     "ISDN type",
			rrtype:   dns.TypeISDN,
			expected: false,
		},
		{
			name:     "RT type",
			rrtype:   dns.TypeRT,
			expected: false,
		},
		{
			name:     "NSAPPTR type",
			rrtype:   dns.TypeNSAPPTR,
			expected: false,
		},
		{
			name:     "SIG type",
			rrtype:   dns.TypeSIG,
			expected: false,
		},
		{
			name:     "KEY type",
			rrtype:   dns.TypeKEY,
			expected: false,
		},
		{
			name:     "PX type",
			rrtype:   dns.TypePX,
			expected: false,
		},
		{
			name:     "GPOS type",
			rrtype:   dns.TypeGPOS,
			expected: false,
		},
		{
			name:     "LOC type",
			rrtype:   dns.TypeLOC,
			expected: false,
		},
		{
			name:     "NXT type",
			rrtype:   dns.TypeNXT,
			expected: false,
		},
		{
			name:     "NAPTR type",
			rrtype:   dns.TypeNAPTR,
			expected: false,
		},
		{
			name:     "KX type",
			rrtype:   dns.TypeKX,
			expected: false,
		},
		{
			name:     "CERT type",
			rrtype:   dns.TypeCERT,
			expected: false,
		},
		{
			name:     "DNAME type",
			rrtype:   dns.TypeDNAME,
			expected: false,
		},
		{
			name:     "OPT type",
			rrtype:   dns.TypeOPT,
			expected: false,
		},
		{
			name:     "APL type",
			rrtype:   dns.TypeAPL,
			expected: false,
		},
		{
			name:     "SSHFP type",
			rrtype:   dns.TypeSSHFP,
			expected: false,
		},
		{
			name:     "IPSECKEY type",
			rrtype:   dns.TypeIPSECKEY,
			expected: false,
		},
		{
			name:     "DHCID type",
			rrtype:   dns.TypeDHCID,
			expected: false,
		},
		{
			name:     "TLSA type",
			rrtype:   dns.TypeTLSA,
			expected: false,
		},
		{
			name:     "SMIMEA type",
			rrtype:   dns.TypeSMIMEA,
			expected: false,
		},
		{
			name:     "HIP type",
			rrtype:   dns.TypeHIP,
			expected: false,
		},
		{
			name:     "NINFO type",
			rrtype:   dns.TypeNINFO,
			expected: false,
		},
		{
			name:     "RKEY type",
			rrtype:   dns.TypeRKEY,
			expected: false,
		},
		{
			name:     "TALINK type",
			rrtype:   dns.TypeTALINK,
			expected: false,
		},
		{
			name:     "SPF type",
			rrtype:   dns.TypeSPF,
			expected: false,
		},
		{
			name:     "UINFO type",
			rrtype:   dns.TypeUINFO,
			expected: false,
		},
		{
			name:     "UID type",
			rrtype:   dns.TypeUID,
			expected: false,
		},
		{
			name:     "GID type",
			rrtype:   dns.TypeGID,
			expected: false,
		},
		{
			name:     "UNSPEC type",
			rrtype:   dns.TypeUNSPEC,
			expected: false,
		},
		{
			name:     "EUI48 type",
			rrtype:   dns.TypeEUI48,
			expected: false,
		},
		{
			name:     "EUI64 type",
			rrtype:   dns.TypeEUI64,
			expected: false,
		},
		{
			name:     "TKEY type",
			rrtype:   dns.TypeTKEY,
			expected: false,
		},
		{
			name:     "TSIG type",
			rrtype:   dns.TypeTSIG,
			expected: false,
		},
		{
			name:     "IXFR type",
			rrtype:   dns.TypeIXFR,
			expected: false,
		},
		{
			name:     "AXFR type",
			rrtype:   dns.TypeAXFR,
			expected: false,
		},
		{
			name:     "URI type",
			rrtype:   dns.TypeURI,
			expected: false,
		},
		{
			name:     "OPENPGPKEY type",
			rrtype:   dns.TypeOPENPGPKEY,
			expected: false,
		},
		{
			name:     "CSYNC type",
			rrtype:   dns.TypeCSYNC,
			expected: false,
		},
		{
			name:     "ZONEMD type",
			rrtype:   dns.TypeZONEMD,
			expected: false,
		},
		{
			name:     "SVCB type",
			rrtype:   dns.TypeSVCB,
			expected: false,
		},
		{
			name:     "HTTPS type",
			rrtype:   dns.TypeHTTPS,
			expected: false,
		},
		// Edge cases
		{
			name:     "zero value",
			rrtype:   0,
			expected: false,
		},
		{
			name:     "max uint16 value",
			rrtype:   65535,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDNSSECType(tt.rrtype)
			if result != tt.expected {
				t.Errorf("IsDNSSECType(%d) = %v, want %v", tt.rrtype, result, tt.expected)
			}
		})
	}
}

func TestIsDNSSECTypeAllDNSSECTypes(t *testing.T) {
	dnssecTypes := []uint16{
		dns.TypeNSEC,
		dns.TypeNSEC3,
		dns.TypeNSEC3PARAM,
		dns.TypeDNSKEY,
		dns.TypeRRSIG,
	}

	for _, rrtype := range dnssecTypes {
		if !IsDNSSECType(rrtype) {
			t.Errorf("IsDNSSECType(%d) = false, expected true for DNSSEC type", rrtype)
		}
	}
}

func TestIsDNSSECTypeNonDNSSECTypes(t *testing.T) {
	nonDNSSECTypes := []uint16{
		dns.TypeA,
		dns.TypeAAAA,
		dns.TypeCNAME,
		dns.TypeMX,
		dns.TypeTXT,
		dns.TypeNS,
		dns.TypeSOA,
		dns.TypeDS,
		dns.TypeCDS,
		dns.TypeCDNSKEY,
	}

	for _, rrtype := range nonDNSSECTypes {
		if IsDNSSECType(rrtype) {
			t.Errorf("IsDNSSECType(%d) = true, expected false for non-DNSSEC type", rrtype)
		}
	}
}

func TestIsDNSSECTypeConsistency(t *testing.T) {
	testType := dns.TypeNSEC

	for i := 0; i < 100; i++ {
		result := IsDNSSECType(testType)
		if !result {
			t.Errorf("IsDNSSECType returned inconsistent result on iteration %d", i)
		}
	}
}

func BenchmarkIsDNSSECType(b *testing.B) {
	testTypes := []uint16{
		dns.TypeNSEC,
		dns.TypeRRSIG,
		dns.TypeA,
		dns.TypeAAAA,
		dns.TypeMX,
	}

	for _, rrtype := range testTypes {
		b.Run(dns.TypeToString[rrtype], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				IsDNSSECType(rrtype)
			}
		})
	}
}

func BenchmarkIsDNSSECTypeWorstCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsDNSSECType(dns.TypeRRSIG)
	}
}

func BenchmarkIsDNSSECTypeBestCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsDNSSECType(dns.TypeNSEC)
	}
}
