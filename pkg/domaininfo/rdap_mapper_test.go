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

package domaininfo

import (
	"testing"

	"github.com/openrdap/rdap"
)

// mustVCard parses a jCard JSON literal and fails the test on error.
func mustVCard(t *testing.T, jcard string) *rdap.VCard {
	t.Helper()
	v, err := rdap.NewVCard([]byte(jcard))
	if err != nil {
		t.Fatalf("NewVCard: %v", err)
	}
	return v
}

const registrarVCard = `["vcard",[
	["version",{},"text","4.0"],
	["fn",{},"text","Acme Registrar"],
	["url",{},"uri","https://acme.example"]
]]`

const registrantVCard = `["vcard",[
	["version",{},"text","4.0"],
	["fn",{},"text","Alice Example"],
	["org",{},"text","Example Inc"],
	["email",{},"text","alice@example.com"],
	["adr",{},"text",["","","123 Main St","Springfield","IL","62701","US"]],
	["tel",{"type":["voice"]},"uri","tel:+1-555-0100"]
]]`

func TestMapRDAPDomain_Full(t *testing.T) {
	in := &rdap.Domain{
		LDHName:     "example.com",
		UnicodeName: "",
		Status:      []string{"clientTransferProhibited"},
		Nameservers: []rdap.Nameserver{
			{LDHName: "ns1.example.com"},
			{UnicodeName: "ns2.exämple.com", LDHName: "ns2.xn--exmple-cua.com"},
		},
		Events: []rdap.Event{
			{Action: "registration", Date: "2000-01-01T00:00:00Z"},
			{Action: "expiration", Date: "2027-06-01T00:00:00Z"},
			{Action: "last changed", Date: "2024-01-01T00:00:00Z"},
		},
		Entities: []rdap.Entity{
			{
				Roles: []string{"registrar"},
				VCard: mustVCard(t, registrarVCard),
			},
			{
				Roles: []string{"registrant"},
				VCard: mustVCard(t, registrantVCard),
			},
		},
	}

	out, err := mapRDAPDomain(in)
	if err != nil {
		t.Fatal(err)
	}

	if out.Name != "example.com" {
		t.Errorf("Name = %q", out.Name)
	}
	if out.Registrar != "Acme Registrar" {
		t.Errorf("Registrar = %q", out.Registrar)
	}
	if out.RegistrarURL == nil || *out.RegistrarURL != "https://acme.example" {
		t.Errorf("RegistrarURL = %v", out.RegistrarURL)
	}

	if out.ExpirationDate == nil || out.ExpirationDate.Year() != 2027 {
		t.Errorf("ExpirationDate = %v", out.ExpirationDate)
	}
	if out.CreationDate == nil || out.CreationDate.Year() != 2000 {
		t.Errorf("CreationDate = %v", out.CreationDate)
	}

	if len(out.Nameservers) != 2 || out.Nameservers[0] != "ns1.example.com" || out.Nameservers[1] != "ns2.exämple.com" {
		t.Errorf("Nameservers = %v (expected unicode preference)", out.Nameservers)
	}
	if len(out.Status) != 1 || out.Status[0] != "clientTransferProhibited" {
		t.Errorf("Status = %v", out.Status)
	}

	if out.Contacts == nil {
		t.Fatal("Contacts is nil")
	}
	r := out.Contacts["registrant"]
	if r == nil {
		t.Fatal("registrant missing")
	}
	if r.Name != "Alice Example" {
		t.Errorf("registrant Name = %q", r.Name)
	}
	if r.Organization != "Example Inc" {
		t.Errorf("registrant Organization = %q", r.Organization)
	}
	if r.Email != "alice@example.com" {
		t.Errorf("registrant Email = %q", r.Email)
	}
	if r.Country != "US" {
		t.Errorf("registrant Country = %q", r.Country)
	}
	if r.Phone == "" {
		t.Errorf("registrant Phone empty")
	}
}

func TestMapRDAPDomain_RoleMapping(t *testing.T) {
	mk := func(role string) rdap.Entity {
		return rdap.Entity{
			Roles: []string{role},
			VCard: mustVCard(t, `["vcard",[["version",{},"text","4.0"],["fn",{},"text","`+role+`"]]]`),
		}
	}
	in := &rdap.Domain{
		LDHName: "example.com",
		Entities: []rdap.Entity{
			mk("registrant"),
			mk("administrative"),
			mk("technical"),
			mk("abuse"), // unmapped: should be skipped
		},
	}
	out, err := mapRDAPDomain(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"registrant", "admin", "tech"} {
		if _, ok := out.Contacts[want]; !ok {
			t.Errorf("missing contact role %q", want)
		}
	}
	if _, ok := out.Contacts["abuse"]; ok {
		t.Error("abuse role should not be present")
	}
	if len(out.Contacts) != 3 {
		t.Errorf("got %d contacts, want 3", len(out.Contacts))
	}
}

func TestMapRDAPDomain_NoEntities(t *testing.T) {
	in := &rdap.Domain{LDHName: "bare.example"}
	out, err := mapRDAPDomain(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Registrar != "Unknown" {
		t.Errorf("Registrar = %q, want Unknown", out.Registrar)
	}
	if out.RegistrarURL != nil {
		t.Errorf("RegistrarURL = %v, want nil", out.RegistrarURL)
	}
	if out.Contacts != nil {
		t.Errorf("Contacts = %v, want nil", out.Contacts)
	}
	if out.ExpirationDate != nil || out.CreationDate != nil {
		t.Error("dates should be nil with no events")
	}
}

func TestMapRDAPDomain_UnicodeNamePreference(t *testing.T) {
	in := &rdap.Domain{
		LDHName:     "xn--bcher-kva.example",
		UnicodeName: "bücher.example",
	}
	out, err := mapRDAPDomain(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "bücher.example" {
		t.Errorf("Name = %q, want unicode form", out.Name)
	}
}

func TestMapRDAPDomain_BadEventDate(t *testing.T) {
	in := &rdap.Domain{
		LDHName: "example.com",
		Events: []rdap.Event{
			{Action: "expiration", Date: "not a date"},
		},
	}
	if _, err := mapRDAPDomain(in); err == nil {
		t.Error("expected error on malformed event date")
	}
}
