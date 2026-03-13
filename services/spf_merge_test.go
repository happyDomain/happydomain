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

package svcs_test

import (
	"strings"
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	svcs "git.happydns.org/happyDomain/services"
)

// mockSPFContributor implements both ServiceBody and SPFContributor.
type mockSPFContributor struct {
	directives []string
	allPolicy  string
}

func (m *mockSPFContributor) GetNbResources() int { return 0 }
func (m *mockSPFContributor) GenComment() string  { return "" }
func (m *mockSPFContributor) GetRecords(string, uint32, string) ([]happydns.Record, error) {
	return nil, nil
}
func (m *mockSPFContributor) GetSPFDirectives() []string { return m.directives }
func (m *mockSPFContributor) GetSPFAllPolicy() string    { return m.allPolicy }

func TestCollectAndMergeSPF_NoContributors(t *testing.T) {
	zone := &happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     &svcs.TXT{Record: &happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "some text"}},
				},
			},
		},
	}

	input := []happydns.Record{
		&happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "some text"},
	}

	result := svcs.CollectAndMergeSPF("example.com.", zone, input, 3600)

	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
}

func TestCollectAndMergeSPF_SingleContributor(t *testing.T) {
	contributor := &mockSPFContributor{
		directives: []string{"include:_spf.google.com"},
		allPolicy:  "~all",
	}

	zone := &happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service:     contributor,
				},
			},
		},
	}

	// Input has an SPF record that should be filtered out
	input := []happydns.Record{
		&happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "v=spf1 include:_spf.google.com ~all"},
	}

	result := svcs.CollectAndMergeSPF("example.com.", zone, input, 3600)

	if len(result) != 1 {
		t.Fatalf("expected 1 record (merged SPF), got %d", len(result))
	}

	txt, ok := result[0].(*happydns.TXT)
	if !ok {
		t.Fatalf("expected *happydns.TXT, got %T", result[0])
	}

	if !strings.HasPrefix(txt.Txt, "v=spf1") {
		t.Errorf("expected SPF record, got %q", txt.Txt)
	}
	if !strings.Contains(txt.Txt, "include:_spf.google.com") {
		t.Errorf("expected google include in SPF, got %q", txt.Txt)
	}
	if !strings.Contains(txt.Txt, "~all") {
		t.Errorf("expected ~all in SPF, got %q", txt.Txt)
	}
}

func TestCollectAndMergeSPF_MultipleContributors(t *testing.T) {
	zone := &happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service: &mockSPFContributor{
						directives: []string{"include:_spf.google.com"},
						allPolicy:  "~all",
					},
				},
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service: &mockSPFContributor{
						directives: []string{"ip4:203.0.113.0/24"},
						allPolicy:  "-all",
					},
				},
			},
		},
	}

	input := []happydns.Record{
		&happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "v=spf1 include:_spf.google.com ~all"},
	}

	result := svcs.CollectAndMergeSPF("example.com.", zone, input, 3600)

	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}

	txt := result[0].(*happydns.TXT)
	if !strings.Contains(txt.Txt, "include:_spf.google.com") {
		t.Errorf("missing google include: %q", txt.Txt)
	}
	if !strings.Contains(txt.Txt, "ip4:203.0.113.0/24") {
		t.Errorf("missing ip4 directive: %q", txt.Txt)
	}
	// -all is stricter than ~all, should win
	if !strings.Contains(txt.Txt, "-all") {
		t.Errorf("expected -all (strictest), got %q", txt.Txt)
	}
}

func TestCollectAndMergeSPF_PreservesNonSPFRecords(t *testing.T) {
	zone := &happydns.Zone{
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {
				{
					ServiceMeta: happydns.ServiceMeta{Domain: ""},
					Service: &mockSPFContributor{
						directives: []string{"include:_spf.google.com"},
						allPolicy:  "~all",
					},
				},
			},
		},
	}

	input := []happydns.Record{
		&happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "google-site-verification=abc"},
		&happydns.TXT{Hdr: dns.RR_Header{Name: "example.com.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 3600}, Txt: "v=spf1 include:_spf.google.com ~all"},
	}

	result := svcs.CollectAndMergeSPF("example.com.", zone, input, 3600)

	// Should have: 1 non-SPF TXT + 1 merged SPF
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}

	foundVerification := false
	foundSPF := false
	for _, rr := range result {
		txt := rr.(*happydns.TXT)
		if strings.HasPrefix(txt.Txt, "google-site") {
			foundVerification = true
		}
		if strings.HasPrefix(txt.Txt, "v=spf1") {
			foundSPF = true
		}
	}
	if !foundVerification {
		t.Error("non-SPF TXT record was removed")
	}
	if !foundSPF {
		t.Error("merged SPF record not found")
	}
}
