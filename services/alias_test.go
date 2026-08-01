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

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
)

func TestAliasCNAME(t *testing.T) {
	// Create a CNAME DNS record
	rr, err := dns.NewRR("www.example.com. 3600 IN CNAME target.example.org.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	s, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone failed: %v", err)
	}

	if len(s) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(s))
	}

	if len(s["www"]) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(s["www"]))
	}

	cnameSvc, ok := s["www"][0].Service.(*svcs.Alias)
	if !ok {
		t.Fatalf("Expected service to be of type *Alias, got %T", s["www"][0].Service)
	}

	// Test GetNbResources always returns 1
	if cnameSvc.GetNbResources() != 1 {
		t.Errorf("GetNbResources() = %d; want 1", cnameSvc.GetNbResources())
	}

	// Test GenComment returns the correct target
	if cnameSvc.GenComment() != "target.example.org." {
		t.Errorf("GenComment() = %q; want %q", cnameSvc.GenComment(), "target.example.org.")
	}

	// Test GetRecords
	records, err := cnameSvc.GetRecords("www.example.com.", 3600, "example.org.")
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	cnameRecord, ok := records[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("Expected *dns.CNAME, got %T", records[0])
	}

	// The target should be fully qualified by helpers.DomainFQDN
	expectedTarget := helpers.DomainFQDN(cnameRecord.Target, "example.org.")
	if cnameRecord.Target != expectedTarget {
		t.Errorf("CNAME target = %q; want %q", cnameRecord.Target, expectedTarget)
	}
}

// TestAliasPseudoTypes covers the alias flavours DNSControl calls pseudo-types:
// they have no DNS wire format, so happyDomain gives them a private use type
// code, and they must survive the analysis like any other record.
func TestAliasPseudoTypes(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		line    string
		comment string
	}{
		{
			name:    "ALIAS",
			line:    "example.com. 3600 IN ALIAS target.example.org.",
			comment: "ALIAS target.example.org.",
		},
		{
			name:    "DNAME",
			line:    "sub.example.com. 3600 IN DNAME target.example.org.",
			comment: "DNAME target.example.org.",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rr, err := dns.NewRR(testCase.line)
			if err != nil {
				t.Fatalf("dns.NewRR failed: %v", err)
			}

			s, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{rr})
			if err != nil {
				t.Fatalf("AnalyzeZone failed: %v", err)
			}

			subdomain := happydns.Subdomain(helpers.DomainRelative(rr.Header().Name, "example.com."))
			if subdomain == "@" {
				subdomain = ""
			}

			if len(s[subdomain]) != 1 {
				t.Fatalf("Expected 1 service for %q, got %d", subdomain, len(s[subdomain]))
			}

			aliasSvc, ok := s[subdomain][0].Service.(*svcs.Alias)
			if !ok {
				t.Fatalf("Expected service to be of type *Alias, got %T", s[subdomain][0].Service)
			}

			if aliasSvc.GenComment() != testCase.comment {
				t.Errorf("GenComment() = %q; want %q", aliasSvc.GenComment(), testCase.comment)
			}

			records, err := aliasSvc.GetRecords(string(subdomain), 3600, "example.com.")
			if err != nil {
				t.Fatalf("GetRecords failed: %v", err)
			}
			if len(records) != 1 {
				t.Fatalf("Expected 1 record, got %d", len(records))
			}
			if got := records[0].Header().Rrtype; got != rr.Header().Rrtype {
				t.Errorf("GetRecords returned a record of type %d; want %d", got, rr.Header().Rrtype)
			}
		})
	}
}

// TestAliasUnmarshalLegacyCNAME checks that the zones stored while this service
// was the CNAME one keep loading.
func TestAliasUnmarshalLegacyCNAME(t *testing.T) {
	var alias svcs.Alias

	if err := json.Unmarshal([]byte(`{"cname":{"Hdr":{"Name":"www","Rrtype":5,"Class":1,"Ttl":3600},"Target":"target.example.org."}}`), &alias); err != nil {
		t.Fatalf("unable to load the former shape: %s", err)
	}

	if alias.GenComment() != "target.example.org." {
		t.Errorf("GenComment() = %q; want %q", alias.GenComment(), "target.example.org.")
	}
}

// TestAliasUnmarshalPseudoType checks the round trip through the storage of a
// pseudo-type record, whose rdata sits behind an interface encoding/json cannot
// fill on its own.
func TestAliasUnmarshalPseudoType(t *testing.T) {
	rr, err := dns.NewRR("example.com. 3600 IN ALIAS target.example.org.")
	if err != nil {
		t.Fatalf("dns.NewRR failed: %v", err)
	}

	stored, err := json.Marshal(svcs.Alias{Record: rr})
	if err != nil {
		t.Fatalf("unable to store the service: %s", err)
	}

	var alias svcs.Alias
	if err := json.Unmarshal(stored, &alias); err != nil {
		t.Fatalf("unable to load the service back: %s", err)
	}

	if alias.Record.String() != rr.String() {
		t.Errorf("the record came back as %q; want %q", alias.Record.String(), rr.String())
	}
}

// TestAliasServiceRegistration covers the service alias mechanism, of which the
// Alias service is the first user: the former "svcs.CNAME" name must still
// resolve, without the service showing up twice or its analyzer running twice.
func TestAliasServiceRegistration(t *testing.T) {
	body, err := svc.FindService("svcs.CNAME")
	if err != nil {
		t.Fatalf("FindService(\"svcs.CNAME\") failed: %s", err)
	}
	if _, ok := body.(*svcs.Alias); !ok {
		t.Errorf("FindService(\"svcs.CNAME\") returned a %T, want a *svcs.Alias", body)
	}

	listed := *svc.ListServices()
	if _, found := listed["svcs.CNAME"]; found {
		t.Error("the former name shows up in ListServices: the service would appear twice in the UI")
	}
	if _, found := listed["svcs.Alias"]; !found {
		t.Error("svcs.Alias is missing from ListServices")
	}

	var occurrences int
	for _, registered := range svc.OrderedServices() {
		if registered.Infos.Type == "svcs.Alias" {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Errorf("the Alias service is registered %d times among the ordered services, want 1: its analyzer would run as many times", occurrences)
	}
}
