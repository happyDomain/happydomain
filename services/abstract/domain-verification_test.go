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

package abstract_test

import (
	"encoding/json"
	"testing"

	"github.com/miekg/dns"

	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	happydns "git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

func analyzeOne(t *testing.T, rr string) map[happydns.Subdomain][]*happydns.Service {
	t.Helper()

	record, err := dns.NewRR(rr)
	if err != nil {
		t.Fatalf("dns.NewRR(%q): %v", rr, err)
	}

	services, _, err := svc.AnalyzeZone("example.com.", []happydns.Record{record})
	if err != nil {
		t.Fatalf("AnalyzeZone: %v", err)
	}

	return services
}

func TestDomainVerification_Analyze(t *testing.T) {
	for _, tc := range []struct {
		rr        string
		subdomain happydns.Subdomain
		owner     string
		provider  string
	}{
		{`example.com. 300 IN TXT "adobe-idp-site-verification=b92c9e999aef825edc36e0a3d847d2dbad5b2fc0e05c79ddd7a16139b48ecf4b"`, "", "", "Adobe"},
		{`example.com. 300 IN TXT "anthropic-domain-verification-4az7qn=if8YWuRRqwLycGJDooumzHtxm"`, "", "", "Anthropic"},
		{`example.com. 300 IN TXT "apple-domain-verification=RyQhdzTl6Z6x8ZP4"`, "", "", "Apple"},
		{`example.com. 300 IN TXT "atlassian-domain-verification=jjgw98AKv2aeoYFxiL/VFaoyPkn3undEssTRuMg6C/3Fp/iqhkV4HVV7WjYlVeF8"`, "", "", "Atlassian"},
		{`example.com. 300 IN TXT "calendly-site-verification=at0DQARi7IZvJtXQAWhMqpmIzpvoBNF7aam5VKKxP"`, "", "", "Calendly"},
		{`example.com. 300 IN TXT "cursor-domain-verification-gtfwmt=1rfLOtiTngX5QSxD5HvNKTvm3"`, "", "", "Cursor"},
		{`example.com. 300 IN TXT "docusign=087098e3-3d46-47b7-9b4e-8a23028154cd"`, "", "", "DocuSign"},
		{`example.com. 300 IN TXT "facebook-domain-verification=39xu4jzl7roi7x0n93ldkxjiaarx50"`, "", "", "Facebook"},
		{`example.com. 300 IN TXT "google-site-verification=abcdef"`, "", "", "Google"},
		{`example.com. 300 IN TXT "jamf-site-verification=XtaPNIYghF_e_xRDI8CjgQ"`, "", "", "Jamf"},
		{`example.com. 300 IN TXT "krisp-domain-verification=ZlyiK7XLhnaoUQb2hpak1PLY7dFkl1WE"`, "", "", "Krisp"},
		{`example.com. 300 IN TXT "loom-site-verification=f3787154f1154b7880e720a511ea664d"`, "", "", "Loom"},
		{`example.com. 300 IN TXT "miro-verification=d2e174fdb00c71e0bcf58f8e58c3da2dd80dcfa9"`, "", "", "Miro"},
		{`example.com. 300 IN TXT "openai-domain-verification=dv-3nh33eQwdkotMIAzLrIDVZo1"`, "", "", "OpenAI"},
		{`example.com. 300 IN TXT "serval-domain-verification-ydryhj=qbkiEakpwEpTvHh5fIiCqtaue"`, "", "", "Serval"},
		{`example.com. 300 IN TXT "shopify-verification-code=t1YPwcmvnxZyBycaCpk1MPyWoFs72o"`, "", "", "Shopify"},
		{`example.com. 300 IN TXT "stripe-verification=f88ef17321660a01bab1660454192e014defa29ba7b8de9633c69d6b4912217f"`, "", "", "Stripe"},
		{`_keybase.example.com. 300 IN TXT "keybase-site-verification=abcdef"`, "", "_keybase", "Keybase"},
		{`_scaleway-challenge.example.com. 300 IN TXT "any token"`, "", "_scaleway-challenge", "Scaleway"},
		{`_gitlab-pages-verification-code.www.example.com. 300 IN TXT "gitlab-pages-verification-code=abcdef"`, "www", "_gitlab-pages-verification-code", "GitLab Pages"},
		{`blog.example.com. 300 IN TXT "google-site-verification=abcdef"`, "blog", "", "Google"},
		// The former GitLab editor put the record on the domain itself, and the
		// v7 migration did so for Keybase: both providers accept that form too.
		{`example.com. 300 IN TXT "gitlab-pages-verification-code=abcdef"`, "", "", "GitLab Pages"},
		{`example.com. 300 IN TXT "keybase-site-verification=abcdef"`, "", "", "Keybase"},
		{`_amazonses.example.com. 300 IN TXT "KaBKBH9DdmyLpWTDqk788NvFc2xAzbrBl9ZcVcGC8Ns="`, "", "_amazonses", "Amazon SES"},
		{`_amazonses.mail.example.com. 300 IN TXT "KaBKBH9DdmyLpWTDqk788NvFc2xAzbrBl9ZcVcGC8Ns="`, "mail", "_amazonses", "Amazon SES"},
		{`_twilio.example.com. 300 IN TXT "twilio-domain-verification=7d1f40d6bb531301fdbc8cc7dc405c1f"`, "", "_twilio", "Twilio"},
		{`example.com. 300 IN TXT "twilio-domain-verification=7d1f40d6bb531301fdbc8cc7dc405c1f"`, "", "", "Twilio"},
		{`_atproto.example.com. 300 IN TXT "did=did:plc:z72i7hdynmk6r22z27h6tvur"`, "", "_atproto", "Bluesky"},
		{`_dnsauth.www.example.com. 300 IN TXT "k78wfs12nhm9k0pfnrrztlyhm2skc1dy"`, "www", "_dnsauth", "DigiCert"},
		{`_oktaverification.example.com. 300 IN TXT "f40a23ceb26549b4b2b53a1c85013912"`, "", "_oktaverification", "Okta"},
		{`_vercel.example.com. 300 IN TXT "vc-domain-verify=partners.example.com,379c035b9d5424adf961"`, "", "_vercel", "Vercel"},
		{`example.com. 300 IN TXT "MS=ms12345678"`, "", "", "Microsoft"},
		{`example.com. 300 IN TXT "yandex-verification: 1234abcd"`, "", "", "Yandex"},
		{`example.com. 300 IN TXT "ZOOM_verify_vCQSjij2TCG8TawaqXfiMQ"`, "", "", "Zoom"},
		{`example.com. 300 IN TXT "_globalsign-domain-verification=abcdef"`, "", "", "GlobalSign"},
	} {
		t.Run(tc.provider+"/"+tc.owner, func(t *testing.T) {
			services := analyzeOne(t, tc.rr)

			if len(services) != 1 || len(services[tc.subdomain]) != 1 {
				t.Fatalf("expected 1 service on %q, got %v", tc.subdomain, services)
			}

			s, ok := services[tc.subdomain][0].Service.(*abstract.DomainVerification)
			if !ok {
				t.Fatalf("expected *abstract.DomainVerification, got %T", services[tc.subdomain][0].Service)
			}

			if got := s.Record.Hdr.Name; got != tc.owner {
				t.Errorf("record owner = %q, want %q", got, tc.owner)
			}
			if got := s.GenComment(); got != tc.provider {
				t.Errorf("GenComment() = %q, want %q", got, tc.provider)
			}

			records, err := s.GetRecords(string(tc.subdomain), 300, "example.com.")
			if err != nil {
				t.Fatalf("GetRecords: %v", err)
			}
			if len(records) != 1 {
				t.Errorf("GetRecords returned %d records, want 1", len(records))
			}
		})
	}
}

// A TXT record no provider claims, or a provider value outside the label it
// expects, is left to the generic services.
func TestDomainVerification_AnalyzeIgnored(t *testing.T) {
	for _, rr := range []string{
		`example.com. 300 IN TXT "v=spf1 -all"`,
		`example.com. 300 IN TXT "hello world"`,
		`example.com. 300 IN TXT "any token"`,
	} {
		for _, svcs := range analyzeOne(t, rr) {
			for _, s := range svcs {
				if _, ok := s.Service.(*abstract.DomainVerification); ok {
					t.Errorf("%s: claimed as a domain verification", rr)
				}
			}
		}
	}
}

// The services this one replaces stored the same body: zones saved before
// still load.
func TestDomainVerification_FormerTypes(t *testing.T) {
	for _, name := range []string{
		"abstract.GoogleVerif",
		"abstract.KeybaseVerif",
		"abstract.GitlabPageVerif",
		"abstract.ScalewayChallenge",
	} {
		body, err := svc.FindService(name)
		if err != nil {
			t.Fatalf("FindService(%q): %v", name, err)
		}

		if err := json.Unmarshal([]byte(`{"txt":{"Hdr":{"Name":"","Rrtype":16,"Class":1,"Ttl":0},"Txt":"google-site-verification=abcdef"}}`), body); err != nil {
			t.Fatalf("%s: json.Unmarshal: %v", name, err)
		}

		s, ok := body.(*abstract.DomainVerification)
		if !ok {
			t.Fatalf("FindService(%q) = %T, want *abstract.DomainVerification", name, body)
		}
		if got := s.GenComment(); got != "Google" {
			t.Errorf("%s: GenComment() = %q, want %q", name, got, "Google")
		}

		if got := svc.CanonicalServiceType(name); got != "abstract.DomainVerification" {
			t.Errorf("CanonicalServiceType(%q) = %q", name, got)
		}
	}
}

func TestDomainVerification_GetRecordsWithoutRecord(t *testing.T) {
	if _, err := (&abstract.DomainVerification{}).GetRecords("", 0, "example.com."); err == nil {
		t.Error("GetRecords with a nil record: expected an error, got none")
	}
}

// An apex entry without a prefix would claim every TXT record of the zone,
// and two entries sharing a label and a prefix would make the second one
// unreachable.
func TestVerificationProviders(t *testing.T) {
	seen := map[[2]string]string{}
	for _, p := range abstract.VerificationProviders {
		if p.Name == "" {
			t.Errorf("provider %+v has no name", p)
		}
		if p.Label == "" && p.Prefix == "" {
			t.Errorf("%s: an apex provider needs a prefix", p.Name)
		}

		key := [2]string{p.Label, p.Prefix}
		if other, ok := seen[key]; ok {
			t.Errorf("%s and %s share the same label and prefix", other, p.Name)
		}
		seen[key] = p.Name
	}
}

// Every entry of the table has to be reachable: a record of its own form is
// told as its provider, not as one listed before it, and lands on the domain
// the label hangs from.
func TestVerificationProviders_Reachable(t *testing.T) {
	for _, p := range abstract.VerificationProviders {
		owner := "example.com."
		if p.Label != "" {
			owner = p.Label + "." + owner
		}

		services := analyzeOne(t, owner+` 300 IN TXT "`+p.Prefix+`abc123"`)
		if len(services[""]) != 1 {
			t.Errorf("%s (%q, %q): expected 1 service at apex, got %v", p.Name, p.Label, p.Prefix, services)
			continue
		}

		s, ok := services[""][0].Service.(*abstract.DomainVerification)
		if !ok {
			t.Errorf("%s (%q, %q): got %T", p.Name, p.Label, p.Prefix, services[""][0].Service)
			continue
		}
		if got := s.GenComment(); got != p.Name {
			t.Errorf("%s (%q, %q): told as %q", p.Name, p.Label, p.Prefix, got)
		}
	}
}

// The service is found by searching for any provider it knows.
func TestDomainVerification_Keywords(t *testing.T) {
	svcs := *svc.ListServices()
	dv, ok := svcs["abstract.DomainVerification"]
	if !ok {
		t.Fatal("abstract.DomainVerification is not registered")
	}

	seen := map[string]bool{}
	for _, k := range dv.Infos.Keywords {
		if seen[k] {
			t.Errorf("keyword %q listed twice", k)
		}
		seen[k] = true
	}

	for _, p := range abstract.VerificationProviders {
		if !seen[p.Name] {
			t.Errorf("provider %q is not a keyword", p.Name)
		}
	}
}
