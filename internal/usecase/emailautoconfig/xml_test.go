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

package emailautoconfig

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/services/abstract"
)

func sampleService() *abstract.EmailAutoConfig {
	return &abstract.EmailAutoConfig{
		DisplayName:      "Example Mail",
		DisplayShortName: "Example",
		IncomingAuth:     "password-cleartext",
		OutgoingAuth:     "password-cleartext",
		UsernameFormat:   "%EMAILADDRESS%",
		IncomingSRV: &dns.SRV{
			Hdr:    dns.RR_Header{Name: "_imaps._tcp", Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 3600},
			Weight: 1,
			Port:   993,
			Target: "imap.example.com.",
		},
		OutgoingSRV: &dns.SRV{
			Hdr:    dns.RR_Header{Name: "_submission._tcp", Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 3600},
			Weight: 1,
			Port:   587,
			Target: "smtp.example.com.",
		},
	}
}

func TestRenderMozillaXML(t *testing.T) {
	body, err := RenderMozillaXML(sampleService(), "example.com", "user@example.com")
	if err != nil {
		t.Fatalf("RenderMozillaXML: %v", err)
	}
	out := string(body)

	for _, want := range []string{
		`<?xml version="1.0"`,
		`<clientConfig version="1.1">`,
		`<emailProvider id="example.com">`,
		`<domain>example.com</domain>`,
		`<displayName>Example Mail</displayName>`,
		`<incomingServer type="imap">`,
		`<hostname>imap.example.com</hostname>`,
		`<port>993</port>`,
		`<socketType>SSL</socketType>`,
		`<outgoingServer type="smtp">`,
		`<socketType>STARTTLS</socketType>`,
		`<port>587</port>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Mozilla XML missing %q\nOutput:\n%s", want, out)
		}
	}
}

func TestRenderAutodiscoverXML(t *testing.T) {
	body, err := RenderAutodiscoverXML(sampleService(), "example.com", "user@example.com")
	if err != nil {
		t.Fatalf("RenderAutodiscoverXML: %v", err)
	}
	out := string(body)

	for _, want := range []string{
		`<Autodiscover`,
		`<AccountType>email</AccountType>`,
		`<Action>settings</Action>`,
		`<Type>IMAP</Type>`,
		`<Server>imap.example.com</Server>`,
		`<Port>993</Port>`,
		`<SSL>on</SSL>`,
		`<Type>SMTP</Type>`,
		`<Port>587</Port>`,
		`<Encryption>TLS</Encryption>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Autodiscover XML missing %q\nOutput:\n%s", want, out)
		}
	}
}

func TestStripDiscoveryPrefix(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"autoconfig.example.com", "example.com."},
		{"autodiscover.example.com.", "example.com."},
		{"example.com", "example.com."},
		{"www.example.com", "www.example.com."},
	} {
		if got := stripDiscoveryPrefix(tc.in); got != tc.want {
			t.Errorf("stripDiscoveryPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
