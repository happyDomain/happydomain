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
	"testing"

	"github.com/miekg/dns"

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
