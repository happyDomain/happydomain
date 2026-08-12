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

package adapter

import (
	"sort"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const rrTypesOrigin = "example.com."

// rrSamples holds one zone file line per record type miekg/dns is able to
// build, the rdata apart: the label and the TTL are added by the test.
//
// The adapters run their conversions over this table, see
// dnscontrol-rrtypes_test.go.
var rrSamples = map[string]string{
	"A":          "1.2.3.4",
	"AAAA":       "2001:db8::1",
	"AFSDB":      "1 afs.example.com.",
	"AMTRELAY":   "10 0 1 203.0.113.1",
	"APL":        "1:192.0.2.0/24 !1:192.0.2.128/25",
	"AVC":        `"app-name=example"`,
	"CAA":        `0 issue "ca.example.net"`,
	"CDNSKEY":    "257 3 13 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"CDS":        "12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"CERT":       "1 12345 13 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"CNAME":      "target.example.com.",
	"CSYNC":      "66 3 A NS AAAA",
	"DHCID":      "AAIBY2/AuCccgoJbsaxcQc9TUapptP69lOjxfNuVAA2kjEA=",
	"DLV":        "12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"DNAME":      "target.example.com.",
	"DNSKEY":     "256 3 13 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"DS":         "12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"EID":        "E32C6F78163167F9",
	"EUI48":      "00-00-5e-00-53-2a",
	"EUI64":      "00-00-5e-ef-10-00-00-2a",
	"GID":        "1000",
	"GPOS":       "-32.6882 116.8652 10.0",
	"HINFO":      `"amd64" "linux"`,
	"HIP":        "2 200100107B1A74DF365639CC39F1D578 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ== rvs.example.com.",
	"HTTPS":      `1 . alpn="h2,h3" ipv4hint="192.0.2.1"`,
	"IPSECKEY":   "10 1 2 192.0.2.38 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"ISDN":       `"150862028003217" "004"`,
	"KEY":        "256 3 13 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"KX":         "10 kx.example.com.",
	"L32":        "10 10.1.2.0",
	"L64":        "10 0014:4fff:ff20:ee64",
	"LOC":        "52 22 23.000 N 4 53 32.000 E -2.00m 0.00m 10000.00m 10.00m",
	"LP":         "10 l64-subnet.example.com.",
	"MB":         "mailbox.example.com.",
	"MD":         "mailagent.example.com.",
	"MF":         "mailagent.example.com.",
	"MG":         "member.example.com.",
	"MINFO":      "rmail.example.com. email.example.com.",
	"MR":         "newname.example.com.",
	"MX":         "10 mail.example.com.",
	"NAPTR":      `100 50 "s" "z3950+I2L+I2C" "" _z3950._tcp.example.com.`,
	"NID":        "10 0014:4fff:ff20:ee64",
	"NIMLOC":     "E32C6F78163167F9",
	"NINFO":      `"this is a zone status"`,
	"NS":         "ns1.example.com.",
	"NSAP-PTR":   "nsap.example.com.",
	"NSEC":       "next.example.com. A MX RRSIG NSEC",
	"NSEC3":      "1 1 12 aabbccdd 2vptu5timamqttgl4luu9kg21e0aor3s A RRSIG",
	"NSEC3PARAM": "1 0 12 aabbccdd",
	"NXT":        "next.example.com. A MX",
	"OPENPGPKEY": "mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"PTR":        "host.example.com.",
	"PX":         "10 net2.example.com. prmd-net2.admd-p400.c-it.",
	"RESINFO":    `"qnamemin" "exterr=15,16,17"`,
	"RKEY":       "0 0 0 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"RP":         "admin.example.com. contact.example.com.",
	"RRSIG":      "A 13 2 3600 20260101000000 20251201000000 12345 example.com. mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"RT":         "10 relay.example.com.",
	"SIG":        "A 13 2 3600 20260101000000 20251201000000 12345 example.com. mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==",
	"SMIMEA":     "3 1 1 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"SOA":        "ns1.example.com. hostmaster.example.com. 2026010101 7200 3600 1209600 3600",
	"SPF":        `"v=spf1 -all"`,
	"SRV":        "10 20 8080 target.example.com.",
	"SSHFP":      "1 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"SVCB":       `1 svc.example.com. alpn="h2" port="8080"`,
	"TA":         "12345 13 2 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"TALINK":     "prev.example.com. next.example.com.",
	"TLSA":       "3 1 1 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
	"TXT":        `"a piece of text"`,
	"UID":        "1000",
	"UINFO":      `"some user info"`,
	"URI":        `10 1 "https://example.com/"`,
	"X25":        "311061700956",
	"ZONEMD":     "2026010101 1 1 0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
}

// rrNotZoneTypes are the types miekg/dns builds records of, but which never
// appear in a zone: the meta-types carried in a message alone, and the ones
// with no presentation format.
var rrNotZoneTypes = map[string]string{
	"ANY":    "QTYPE, asked in a query, never held in a zone",
	"NULL":   "RFC 1035 says it cannot appear in a zone file",
	"NXNAME": "pseudo-type of the compact denial of existence, only in a response",
	"OPT":    "EDNS0 meta-record, carried in the additional section",
	"TKEY":   "meta-record of the key exchange, never held in a zone",
	"TSIG":   "meta-record of the message signature, never held in a zone",
}

// sampleNames holds the types of the sample table in a stable order.
var sampleNames = func() []string {
	names := make([]string, 0, len(rrSamples))
	for name := range rrSamples {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}()

// TestRRSamplesCoverEveryType makes sure the sample table above keeps up with
// miekg/dns: a type it learns to build is either given a sample, or explicitly
// left out.
func TestRRSamplesCoverEveryType(t *testing.T) {
	excluded := map[string]bool{}
	for name := range rrNotZoneTypes {
		excluded[name] = true
	}
	// The DNSControl pseudo-types happyDomain registers as private use types
	// in miekg/dns: dnscontrol-pseudotypes_test.go covers them, they have no
	// rdata a zone file could hold in the general case.
	for name := range supportedPseudoTypes {
		excluded[name] = true
	}

	var missing []string
	for rrtype := range dns.TypeToRR {
		name := dns.TypeToString[rrtype]
		if _, ok := rrSamples[name]; !ok && !excluded[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	for _, name := range missing {
		t.Errorf("type %s has no sample in rrSamples: add one, or list it in rrNotZoneTypes", name)
	}

	// The other way around: a sample naming a type miekg/dns cannot build is a
	// typo.
	for name := range rrSamples {
		rrtype, known := dns.StringToType[name]
		if !known || dns.TypeToRR[rrtype] == nil {
			t.Errorf("rrSamples holds %s, which miekg/dns is unable to build", name)
		}
	}
}

// parseSample builds the record of the given type from its sample rdata.
func parseSample(t *testing.T, name string) happydns.Record {
	t.Helper()

	return mustRR(t, "sample."+rrTypesOrigin+" 300 IN "+name+" "+rrSamples[name])
}
