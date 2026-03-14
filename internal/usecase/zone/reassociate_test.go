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

package zone

import (
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

// testEnricherService is a mock service body that implements MetadataEnricher.
type testEnricherService struct {
	ExtraField string
	Record     *dns.A
}

func (s *testEnricherService) GetNbResources() int { return 1 }
func (s *testEnricherService) GenComment() string  { return s.ExtraField }
func (s *testEnricherService) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}
func (s *testEnricherService) EnrichFromPrevious(old happydns.ServiceBody) {
	if prev, ok := old.(*testEnricherService); ok {
		s.ExtraField = prev.ExtraField
	}
}

// testSimpleService is a mock service body without MetadataEnricher.
type testSimpleService struct {
	Record *dns.A
}

func (s *testSimpleService) GetNbResources() int { return 1 }
func (s *testSimpleService) GenComment() string  { return "" }
func (s *testSimpleService) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

func makeIdentifier(b byte) happydns.Identifier {
	return happydns.Identifier{b}
}

func TestReassociateMetadata_UnambiguousMatch(t *testing.T) {
	oldId := makeIdentifier(1)
	oldSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:          oldId,
			Type:        "svcs.testSimpleService",
			Domain:      "www",
			UserComment: "my web server",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{1, 2, 3, 4}}},
	}

	newSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:     makeIdentifier(99),
			Type:   "svcs.testSimpleService",
			Domain: "www",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{1, 2, 3, 4}}},
	}

	dn := happydns.Subdomain("www")
	oldServices := map[happydns.Subdomain][]*happydns.Service{dn: {oldSvc}}
	newServices := map[happydns.Subdomain][]*happydns.Service{dn: {newSvc}}

	ReassociateMetadata(oldServices, newServices, "example.com.", 300)

	if !newSvc.Id.Equals(oldId) {
		t.Errorf("expected Id to be transferred, got %v", newSvc.Id)
	}
	if newSvc.UserComment != "my web server" {
		t.Errorf("expected UserComment to be transferred, got %q", newSvc.UserComment)
	}
}

func TestReassociateMetadata_EnrichFromPrevious(t *testing.T) {
	oldId := makeIdentifier(2)
	oldSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   oldId,
			Type: "svcs.testEnricherService",
		},
		Service: &testEnricherService{
			ExtraField: "preserved-value",
			Record:     &dns.A{A: []byte{10, 0, 0, 1}},
		},
	}

	newSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(88),
			Type: "svcs.testEnricherService",
		},
		Service: &testEnricherService{
			ExtraField: "",
			Record:     &dns.A{A: []byte{10, 0, 0, 1}},
		},
	}

	dn := happydns.Subdomain("")
	oldServices := map[happydns.Subdomain][]*happydns.Service{dn: {oldSvc}}
	newServices := map[happydns.Subdomain][]*happydns.Service{dn: {newSvc}}

	ReassociateMetadata(oldServices, newServices, "example.com.", 300)

	if !newSvc.Id.Equals(oldId) {
		t.Errorf("expected Id to be transferred, got %v", newSvc.Id)
	}

	enriched := newSvc.Service.(*testEnricherService)
	if enriched.ExtraField != "preserved-value" {
		t.Errorf("expected ExtraField to be enriched, got %q", enriched.ExtraField)
	}
}

func TestReassociateMetadata_AmbiguousMatch(t *testing.T) {
	// Two services of the same type on the same subdomain
	oldSvc1 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:          makeIdentifier(1),
			Type:        "svcs.testSimpleService",
			UserComment: "server-A",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{1, 1, 1, 1}}},
	}
	oldSvc2 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:          makeIdentifier(2),
			Type:        "svcs.testSimpleService",
			UserComment: "server-B",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{2, 2, 2, 2}}},
	}

	newSvc1 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(77),
			Type: "svcs.testSimpleService",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{2, 2, 2, 2}}},
	}
	newSvc2 := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(78),
			Type: "svcs.testSimpleService",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{1, 1, 1, 1}}},
	}

	dn := happydns.Subdomain("lb")
	oldServices := map[happydns.Subdomain][]*happydns.Service{dn: {oldSvc1, oldSvc2}}
	newServices := map[happydns.Subdomain][]*happydns.Service{dn: {newSvc1, newSvc2}}

	ReassociateMetadata(oldServices, newServices, "example.com.", 300)

	// newSvc1 has IP 2.2.2.2 -> should match oldSvc2
	if newSvc1.UserComment != "server-B" {
		t.Errorf("expected newSvc1 to get server-B comment, got %q", newSvc1.UserComment)
	}
	if !newSvc1.Id.Equals(makeIdentifier(2)) {
		t.Errorf("expected newSvc1 to get oldSvc2's Id")
	}

	// newSvc2 has IP 1.1.1.1 -> should match oldSvc1
	if newSvc2.UserComment != "server-A" {
		t.Errorf("expected newSvc2 to get server-A comment, got %q", newSvc2.UserComment)
	}
	if !newSvc2.Id.Equals(makeIdentifier(1)) {
		t.Errorf("expected newSvc2 to get oldSvc1's Id")
	}
}

func TestReassociateMetadata_NoMatch(t *testing.T) {
	oldSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:          makeIdentifier(1),
			Type:        "svcs.testSimpleService",
			UserComment: "should not transfer",
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{1, 2, 3, 4}}},
	}

	// New service has a different type
	newSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(50),
			Type: "svcs.testEnricherService",
		},
		Service: &testEnricherService{Record: &dns.A{A: []byte{1, 2, 3, 4}}},
	}

	dn := happydns.Subdomain("www")
	oldServices := map[happydns.Subdomain][]*happydns.Service{dn: {oldSvc}}
	newServices := map[happydns.Subdomain][]*happydns.Service{dn: {newSvc}}

	ReassociateMetadata(oldServices, newServices, "example.com.", 300)

	// Should NOT have transferred
	if newSvc.Id.Equals(makeIdentifier(1)) {
		t.Errorf("Id should not have been transferred for mismatched types")
	}
	if newSvc.UserComment != "" {
		t.Errorf("UserComment should not have been transferred, got %q", newSvc.UserComment)
	}
}

func TestReassociateMetadata_TtlTransfer(t *testing.T) {
	defaultTTL := uint32(300)
	serviceTTL := uint32(600)

	record := &dns.A{
		Hdr: dns.RR_Header{Ttl: 0}, // means "use zone default"
		A:   []byte{10, 0, 0, 1},
	}

	oldSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(1),
			Type: "svcs.testSimpleService",
			Ttl:  serviceTTL,
		},
		Service: &testSimpleService{Record: &dns.A{A: []byte{10, 0, 0, 1}}},
	}

	newSvc := &happydns.Service{
		ServiceMeta: happydns.ServiceMeta{
			Id:   makeIdentifier(50),
			Type: "svcs.testSimpleService",
		},
		Service: &testSimpleService{Record: record},
	}

	dn := happydns.Subdomain("www")
	oldServices := map[happydns.Subdomain][]*happydns.Service{dn: {oldSvc}}
	newServices := map[happydns.Subdomain][]*happydns.Service{dn: {newSvc}}

	ReassociateMetadata(oldServices, newServices, "example.com.", defaultTTL)

	if newSvc.Ttl != serviceTTL {
		t.Errorf("expected Ttl %d, got %d", serviceTTL, newSvc.Ttl)
	}

	// The record had TTL 0 (zone default), after transfer it should be set to defaultTTL
	if record.Hdr.Ttl != defaultTTL {
		t.Errorf("expected record TTL to be set to defaultTTL %d, got %d", defaultTTL, record.Hdr.Ttl)
	}
}
