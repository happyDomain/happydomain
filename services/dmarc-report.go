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

package svcs

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type DMARCReport struct {
	Records []*happydns.TXT `json:"txt"`
}

func (s *DMARCReport) GetNbResources() int {
	return len(s.Records)
}

func (s *DMARCReport) GenComment() string {
	var domains []string

	for _, rr := range s.Records {
		domains = append(domains, strings.TrimSuffix(rr.Header().Name, "._report._dmarc"))
	}

	return strings.Join(domains, ", ")
}

func (t *DMARCReport) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	for _, rr := range t.Records {
		rrs = append(rrs, rr)
	}

	return
}
func dmarc_report_analyze(a *Analyzer) (err error) {
	services := map[string]*DMARCReport{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT, DomainContains: "._report._dmarc"}) {
		txt, ok := record.(*happydns.TXT)
		dmidx := strings.Index(record.Header().Name, "._report._dmarc.")
		if dmidx <= 0 || !ok || !strings.HasPrefix(strings.ToLower(txt.Txt), "v=dmarc1") {
			continue
		}
		domain := record.Header().Name[dmidx+16:]

		if _, ok := services[domain]; !ok {
			services[domain] = &DMARCReport{}
		}

		services[domain].Records = append(
			services[domain].Records,
			helpers.RRRelative(record, domain).(*happydns.TXT),
		)

		err = a.UseRR(record, domain, services[domain])
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &DMARCReport{}
		},
		dmarc_report_analyze,
		happydns.ServiceInfos{
			Name:        "DMARC allow receiving reports",
			Description: "Allow a domain to receive DMARC reports for another domain.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
