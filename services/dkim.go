// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

	"git.happydns.org/happyDomain/model"
)

type DKIM struct {
	Record *dns.TXT `json:"txt"`
}

func (s *DKIM) GetNbResources() int {
	return 1
}

func (s *DKIM) GenComment(origin string) string {
	p := strings.Index(s.Record.Hdr.Name, "._domainkey")

	if p <= 0 {
		return "Invalid DKIM selector"
	}

	return s.Record.Hdr.Name[:p]
}

func (s *DKIM) GetRecords(domain string, ttl uint32, origin string) (rrs []dns.RR, e error) {
	return []dns.RR{s.Record}, nil
}

func dkim_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		dkidx := strings.Index(record.NameFQDN, "._domainkey.")
		if dkidx <= 0 {
			continue
		}

		err = a.UseRR(record, record.NameFQDN[dkidx+12:], &DKIM{
			Record: record.ToRR().(*dns.TXT),
		})
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &DKIM{}
		},
		dkim_analyze,
		ServiceInfos{
			Name:        "DKIM",
			Description: "DomainKeys Identified Mail, authenticate outgoing emails.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeTXT,
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeTXT,
				},
			},
		},
		1,
	)
}
