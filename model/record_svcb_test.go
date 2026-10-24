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
	"encoding/json"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

const httpsZoneLine = `example.com. 300 IN HTTPS 1 . alpn="h3,h2" ipv4hint="104.21.51.188,172.67.184.36" ech="AEX+DQBB9QAgACBg80oTU+t7uyBFSxzEuJ0uU61qecetrwp9YvTZi4qzfgAEAAEAAQASY2xvdWRmbGFyZS1lY2guY29tAAA=" ipv6hint="2606:4700:3032::6815:33bc,2606:4700:3033::ac43:b824"`

const svcbZoneLine = `example.com. 300 IN SVCB 1 svc.example.com. mandatory="alpn,port" alpn="h2" no-default-alpn="" port="8443" dohpath="/dns-query{?dns}" ohttp="" key65000="foo"`

func mustRR(t *testing.T, line string) dns.RR {
	t.Helper()
	rr, err := dns.NewRR(line)
	if err != nil {
		t.Fatalf("NewRR(%q): %v", line, err)
	}
	return rr
}

func TestHTTPSJSONRoundTrip(t *testing.T) {
	rr := mustRR(t, httpsZoneLine)
	record := happydns.NewHTTPS(rr.(*dns.HTTPS))

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}

	var got happydns.HTTPS
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", payload, err)
	}

	if got.String() != rr.String() {
		t.Errorf("round trip changed the record:\n want %s\n got  %s", rr, got.String())
	}
	if _, ok := got.ToRR().(*dns.HTTPS); !ok {
		t.Errorf("ToRR() = %T, want *dns.HTTPS", got.ToRR())
	}
}

func TestSVCBJSONRoundTrip(t *testing.T) {
	for _, line := range []string{svcbZoneLine, `example.com. 300 IN SVCB 0 alias.example.com.`} {
		rr := mustRR(t, line)

		payload, err := json.Marshal(happydns.NewSVCB(rr.(*dns.SVCB)))
		if err != nil {
			t.Fatal(err)
		}

		var got happydns.SVCB
		if err := json.Unmarshal(payload, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", payload, err)
		}

		if got.String() != rr.String() {
			t.Errorf("round trip changed the record:\n want %s\n got  %s", rr, got.String())
		}
		if _, ok := got.ToRR().(*dns.SVCB); !ok {
			t.Errorf("ToRR() = %T, want *dns.SVCB", got.ToRR())
		}
	}
}

// The parameters are written in their zone file form, the only one telling
// apart the keys sharing a Go type, and the one the frontend displays.
func TestSVCBJSONValueIsText(t *testing.T) {
	payload, err := json.Marshal(happydns.NewHTTPS(mustRR(t, httpsZoneLine).(*dns.HTTPS)))
	if err != nil {
		t.Fatal(err)
	}

	var stored struct{ Value []string }
	if err := json.Unmarshal(payload, &stored); err != nil {
		t.Fatalf("Value is not a list of strings: %s", payload)
	}
	if len(stored.Value) != 4 || stored.Value[0] != `alpn="h3,h2"` || stored.Value[3] != `ipv6hint="2606:4700:3032::6815:33bc,2606:4700:3033::ac43:b824"` {
		t.Errorf("unexpected Value: %q", stored.Value)
	}
}

// Records stored before happyDomain had its own SVCB type carry the objects
// encoding/json made of the miekg/dns key types.
func TestHTTPSUnmarshalLegacyObjects(t *testing.T) {
	legacy := `{"Hdr":{"Name":"example.com.","Rrtype":65,"Class":1,"Ttl":300,"Rdlength":136},"Priority":1,"Target":".","Value":[{"Alpn":["h3","h2"]},{"Hint":["104.21.51.188","172.67.184.36"]},{"ECH":"AEX+DQBB9QAgACBg80oTU+t7uyBFSxzEuJ0uU61qecetrwp9YvTZi4qzfgAEAAEAAQASY2xvdWRmbGFyZS1lY2guY29tAAA="},{"Hint":["2606:4700:3032::6815:33bc","2606:4700:3033::ac43:b824"]}]}`

	var got happydns.HTTPS
	if err := json.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if want := mustRR(t, httpsZoneLine).String(); got.String() != want {
		t.Errorf("legacy record misread:\n want %s\n got  %s", want, got.String())
	}
}

func TestSVCBUnmarshalLegacyObjects(t *testing.T) {
	legacy := `{"Hdr":{"Name":"example.com.","Rrtype":64,"Class":1,"Ttl":300},"Priority":1,"Target":"svc.example.com.","Value":[{"Code":[1,3]},{"Alpn":["h2"]},{},{"Port":8443},{"Template":"/dns-query{?dns}"},{"KeyCode":65000,"Data":"Zm9v"}]}`

	var got happydns.SVCB
	if err := json.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	want := mustRR(t, `example.com. 300 IN SVCB 1 svc.example.com. mandatory="alpn,port" alpn="h2" no-default-alpn="" port="8443" dohpath="/dns-query{?dns}" key65000="foo"`).String()
	if got.String() != want {
		t.Errorf("legacy record misread:\n want %s\n got  %s", want, got.String())
	}
}

func TestSVCBUnmarshalBadParameter(t *testing.T) {
	for _, value := range []string{`"port=\\"http\\""`, `"alpn=\\"h2\\" port=\\"443\\""`, `{"Unknown":1}`} {
		var got happydns.SVCB
		if err := json.Unmarshal([]byte(`{"Hdr":{"Rrtype":64},"Priority":1,"Target":".","Value":[`+value+`]}`), &got); err == nil {
			t.Errorf("expected an error on parameter %s, got %s", value, got.String())
		}
	}
}

func TestSVCBCopy(t *testing.T) {
	original := happydns.NewSVCB(mustRR(t, svcbZoneLine).(*dns.SVCB))
	copied := original.Copy().(*happydns.SVCB)

	copied.Value[1].(*dns.SVCBAlpn).Alpn[0] = "h3"
	copied.Hdr.Ttl = 42

	if original.String() != mustRR(t, svcbZoneLine).String() {
		t.Errorf("modifying the copy changed the original: %s", original.String())
	}
}

func TestHTTPSCopy(t *testing.T) {
	original := happydns.NewHTTPS(mustRR(t, httpsZoneLine).(*dns.HTTPS))
	if _, ok := original.Copy().(*happydns.HTTPS); !ok {
		t.Errorf("Copy() = %T, want *happydns.HTTPS", original.Copy())
	}
}
