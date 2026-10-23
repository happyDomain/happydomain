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

package serviceanalyzer

import (
	"testing"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// testService is a service body standing for any service, or an orphan.
type testService struct {
	kind string
}

func (s *testService) GetNbResources() int { return 1 }
func (s *testService) GenComment() string  { return s.kind }
func (s *testService) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return nil, nil
}

// withServices runs the analysis with only the given analyzers registered,
// in their order, and a stub orphan creator.
func withServices(t *testing.T, analyzers ...ServiceAnalyzer) {
	t.Helper()

	savedServices, savedOrdered, savedOrphan := services, ordered_services, OrphanCreator
	t.Cleanup(func() {
		services, ordered_services, OrphanCreator = savedServices, savedOrdered, savedOrphan
	})

	services = map[string]*Svc{}
	ordered_services = nil
	for i, analyzer := range analyzers {
		services[string(rune('a'+i))] = &Svc{Analyzer: analyzer, Weight: uint32(i)}
	}
	OrphanCreator = func(record happydns.Record) happydns.ServiceBody {
		return &testService{kind: "orphan"}
	}
}

func TestAnalyzeZoneKeepsUnclaimedRecordsIntact(t *testing.T) {
	// Both make the record relative, as the analyzers do before telling
	// whether it is theirs, then leave it.
	relativize := func(a *Analyzer) error {
		for _, rr := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
			helpers.RRRelativeSubdomain(rr, a.GetOrigin(), "example.com.")
		}
		return nil
	}
	withServices(t, relativize, relativize)

	var orphans []string
	OrphanCreator = func(record happydns.Record) happydns.ServiceBody {
		orphans = append(orphans, record.Header().Name)
		return &testService{kind: "orphan"}
	}

	rr, err := dns.NewRR("_adsp._domainkey.example.com. 300 IN TXT \"dkim=all;\"")
	if err != nil {
		t.Fatalf("dns.NewRR: %v", err)
	}

	svcs, _, err := AnalyzeZone("example.com.", []happydns.Record{rr})
	if err != nil {
		t.Fatalf("AnalyzeZone: %v", err)
	}

	if len(orphans) != 1 || orphans[0] != "" {
		t.Errorf("orphan names = %q; want the record relative to its subdomain once", orphans)
	}
	if _, ok := svcs["_adsp._domainkey"]; !ok || len(svcs) != 1 {
		t.Errorf("subdomains = %v; want _adsp._domainkey", svcs)
	}
	if rr.Header().Name != "_adsp._domainkey.example.com." {
		t.Errorf("the record handed was changed: %q", rr.Header().Name)
	}
}
