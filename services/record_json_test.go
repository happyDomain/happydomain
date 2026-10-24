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
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

const httpsZoneLine = `example.com. 300 IN HTTPS 1 . alpn="h3,h2" ipv4hint="104.21.51.188,172.67.184.36" ech="AEX+DQBB9QAgACBg80oTU+t7uyBFSxzEuJ0uU61qecetrwp9YvTZi4qzfgAEAAEAAQASY2xvdWRmbGFyZS1lY2guY29tAAA=" ipv6hint="2606:4700:3032::6815:33bc,2606:4700:3033::ac43:b824"`

// An imported HTTPS record ends up in an Orphan, which has to survive being
// stored and read back.
func TestOrphanHTTPSFromZone(t *testing.T) {
	rr, err := dns.NewRR(httpsZoneLine)
	if err != nil {
		t.Fatal(err)
	}

	services, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatal(err)
	}

	var orphan *svcs.Orphan
	for _, list := range services {
		for _, s := range list {
			if o, ok := s.Service.(*svcs.Orphan); ok {
				orphan = o
			}
		}
	}
	if orphan == nil {
		t.Fatalf("no Orphan service in %v", services)
	}

	payload, err := json.Marshal(orphan)
	if err != nil {
		t.Fatal(err)
	}

	var got svcs.Orphan
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", payload, err)
	}

	if _, ok := got.Record.(*happydns.HTTPS); !ok {
		t.Errorf("read back a %T, want *happydns.HTTPS", got.Record)
	}
	if got.Record.String() != orphan.Record.String() {
		t.Errorf("round trip changed the record:\n want %s\n got  %s", orphan.Record, got.Record)
	}
}

// The Orphan stored before happyDomain had its own SVCB type, as reported.
func TestOrphanHTTPSLegacy(t *testing.T) {
	legacy := `{"record":{"Hdr":{"Name":"","Rrtype":65,"Class":1,"Ttl":300,"Rdlength":136},"Priority":1,"Target":".","Value":[{"Alpn":["h3","h2"]},{"Hint":["104.21.51.188","172.67.184.36"]},{"ECH":"AEX+DQBB9QAgACBg80oTU+t7uyBFSxzEuJ0uU61qecetrwp9YvTZi4qzfgAEAAEAAQASY2xvdWRmbGFyZS1lY2guY29tAAA="},{"Hint":["2606:4700:3032::6815:33bc","2606:4700:3033::ac43:b824"]}]}}`

	var got svcs.Orphan
	if err := json.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want, err := dns.NewRR(httpsZoneLine)
	if err != nil {
		t.Fatal(err)
	}
	want.Header().Name = ""

	if got.Record.String() != want.String() {
		t.Errorf("legacy record misread:\n want %s\n got  %s", want, got.Record)
	}
}

// Every common record type, as the analyzer or the user hands it, survives
// being stored in a service and read back.
func TestRecordJSONRoundTrip(t *testing.T) {
	for _, line := range []string{
		`a.example.com. 300 IN A 192.0.2.1`,
		`a.example.com. 300 IN AAAA 2001:db8::1`,
		`a.example.com. 300 IN CNAME target.example.net.`,
		`a.example.com. 300 IN DNAME target.example.net.`,
		`example.com. 300 IN ALIAS target.example.net.`,
		`example.com. 300 IN NS ns1.example.net.`,
		`example.com. 300 IN SOA ns1.example.net. hostmaster.example.com. 2026102401 7200 3600 1209600 300`,
		`example.com. 300 IN MX 10 mx.example.net.`,
		`1.2.0.192.in-addr.arpa. 300 IN PTR host.example.com.`,
		`_sip._tcp.example.com. 300 IN SRV 10 60 5060 sip.example.com.`,
		`a.example.com. 300 IN TXT "hello world"`,
		`a.example.com. 300 IN TXT "` + strings.Repeat("x", 300) + `"`,
		`a.example.com. 300 IN SPF "v=spf1 -all"`,
		`example.com. 300 IN CAA 0 issue "letsencrypt.org"`,
		`example.com. 300 IN CAA 128 iodef "mailto:security@example.com"`,
		`_443._tcp.example.com. 300 IN TLSA 3 1 1 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`,
		`a.example.com. 300 IN SSHFP 4 2 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`,
		`example.com. 300 IN DS 12345 13 2 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef`,
		`example.com. 300 IN DNSKEY 257 3 13 mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==`,
		`example.com. 300 IN NAPTR 100 10 "S" "SIP+D2U" "" _sip._udp.example.com.`,
		`a.example.com. 300 IN LOC 48 51 24.000 N 2 21 3.000 E 35.00m 1m 10000m 10m`,
		`a.example.com. 300 IN HINFO "x86_64" "Linux"`,
		`a.example.com. 300 IN RP hostmaster.example.com. info.example.com.`,
		`_http._tcp.example.com. 300 IN URI 10 1 "https://example.com/"`,
		`example.com. 300 IN SVCB 1 svc.example.com. alpn="h2" port="8443"`,
		`example.com. 300 IN HTTPS 1 . alpn="h3,h2" ipv4hint="192.0.2.1" ipv6hint="2001:db8::1"`,
		`example.com. 300 IN HTTPS 0 cdn.example.net.`,
		`example.com. 0 IN TYPE65534 \# 5 0d1c520001`,
	} {
		rr, err := helpers.ParseRecord(line, "example.com.")
		if err != nil {
			t.Fatalf("ParseRecord(%q): %v", line, err)
		}

		payload, err := json.Marshal(&svcs.Orphan{Record: rr})
		if err != nil {
			t.Errorf("marshal %q: %v", line, err)
			continue
		}

		var got svcs.Orphan
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Errorf("unmarshal %s: %v", payload, err)
			continue
		}

		if got.Record.String() != rr.String() {
			t.Errorf("round trip changed the record:\n want %s\n got  %s", rr, got.Record)
		}
		if fmt.Sprintf("%T", got.Record) != fmt.Sprintf("%T", rr) {
			t.Errorf("%s: read back a %T, want %T", line, got.Record, rr)
		}
	}
}

// The miekg/dns form, the text split in segments, is still read.
func TestOrphanTXTSegments(t *testing.T) {
	for _, payload := range []string{
		`{"record":{"Hdr":{"Name":"a","Rrtype":16,"Class":1,"Ttl":300},"Txt":["hel","lo"]}}`,
		`{"record":{"Hdr":{"Name":"a","Rrtype":99,"Class":1,"Ttl":300},"Txt":["hel","lo"]}}`,
	} {
		var got svcs.Orphan
		if err := json.Unmarshal([]byte(payload), &got); err != nil {
			t.Fatalf("unmarshal %s: %v", payload, err)
		}

		switch record := got.Record.(type) {
		case *happydns.TXT:
			if record.Txt != "hello" {
				t.Errorf("Txt = %q, want hello", record.Txt)
			}
		case *happydns.SPF:
			if record.Txt != "hello" {
				t.Errorf("Txt = %q, want hello", record.Txt)
			}
		default:
			t.Errorf("read back a %T, want happyDomain's abstraction", got.Record)
		}
	}
}

// A record of a type miekg/dns does not know, as BIND keeps its signing state
// in TYPE65534, as stored when the zone was imported.
func TestOrphanUnknownType(t *testing.T) {
	payload := `{"record":{"Hdr":{"Name":"","Rrtype":65534,"Class":1,"Ttl":0,"Rdlength":5},"Rdata":"0d1c520001"}}`

	var got svcs.Orphan
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	record, ok := got.Record.(*dns.RFC3597)
	if !ok {
		t.Fatalf("read back a %T, want *dns.RFC3597", got.Record)
	}
	if record.Rdata != "0d1c520001" {
		t.Errorf("Rdata = %q, want 0d1c520001", record.Rdata)
	}
}
