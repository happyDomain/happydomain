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
	"time"

	whoisparser "github.com/likexian/whois-parser"
)

func TestMapWhoisResult_Full(t *testing.T) {
	exp := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	in := &whoisparser.WhoisInfo{
		Domain: &whoisparser.Domain{
			Domain:               "example.com",
			NameServers:          []string{"ns1.example.com", "ns2.example.com"},
			Status:               []string{"clientTransferProhibited"},
			CreatedDateInTime:    &created,
			ExpirationDateInTime: &exp,
		},
		Registrar: &whoisparser.Contact{
			Name:        "Acme Registrar",
			ReferralURL: "https://acme.example",
		},
		Registrant: &whoisparser.Contact{
			Name:         "Alice",
			Organization: "Example Inc",
			Email:        "alice@example.com",
			Country:      "US",
		},
		Administrative: &whoisparser.Contact{Name: "Admin"},
		Technical:      nil,
	}

	out := mapWhoisResult(in)
	if out.Name != "example.com" {
		t.Errorf("Name = %q", out.Name)
	}
	if len(out.Nameservers) != 2 {
		t.Errorf("Nameservers = %v", out.Nameservers)
	}
	if out.ExpirationDate == nil || !out.ExpirationDate.Equal(exp) {
		t.Errorf("ExpirationDate = %v", out.ExpirationDate)
	}
	if out.CreationDate == nil || !out.CreationDate.Equal(created) {
		t.Errorf("CreationDate = %v", out.CreationDate)
	}
	if out.Registrar != "Acme Registrar" {
		t.Errorf("Registrar = %q", out.Registrar)
	}
	if out.RegistrarURL == nil || *out.RegistrarURL != "https://acme.example" {
		t.Errorf("RegistrarURL = %v", out.RegistrarURL)
	}
	if len(out.Status) != 1 || out.Status[0] != "clientTransferProhibited" {
		t.Errorf("Status = %v", out.Status)
	}

	if out.Contacts == nil {
		t.Fatal("expected contacts map")
	}
	if r := out.Contacts["registrant"]; r == nil || r.Name != "Alice" || r.Email != "alice@example.com" || r.Organization != "Example Inc" || r.Country != "US" {
		t.Errorf("registrant = %+v", r)
	}
	if a := out.Contacts["admin"]; a == nil || a.Name != "Admin" {
		t.Errorf("admin = %+v", a)
	}
	if _, ok := out.Contacts["tech"]; ok {
		t.Error("tech should be absent when nil")
	}
}

func TestMapWhoisResult_NilRegistrarAndContacts(t *testing.T) {
	in := &whoisparser.WhoisInfo{
		Domain: &whoisparser.Domain{Domain: "bare.example"},
	}
	out := mapWhoisResult(in)
	if out.Registrar != "Unknown" {
		t.Errorf("Registrar = %q, want Unknown", out.Registrar)
	}
	if out.RegistrarURL != nil {
		t.Errorf("RegistrarURL = %v, want nil", out.RegistrarURL)
	}
	if out.Contacts != nil {
		t.Errorf("Contacts = %v, want nil", out.Contacts)
	}
}

func TestMapWhoisResult_NilDomain(t *testing.T) {
	// Defensive: parser shouldn't normally produce this, but the mapper
	// must not panic on a nil Domain.
	out := mapWhoisResult(&whoisparser.WhoisInfo{})
	if out.Name != "" || out.Nameservers != nil || out.ExpirationDate != nil {
		t.Errorf("expected zero-valued domain fields, got %+v", out)
	}
}
