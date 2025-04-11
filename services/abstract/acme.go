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

package abstract

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type ACMEChallenge struct {
	Challenge string
}

func (s *ACMEChallenge) GetNbResources() int {
	return 1
}

func (s *ACMEChallenge) GenComment() string {
	return s.Challenge
}

func (s *ACMEChallenge) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rr := utils.NewRecord(utils.DomainJoin("_acme-challenge", domain), "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{s.Challenge}
	return []happydns.Record{rr}, nil
}

func acmechallenge_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_acme-challenge"}) {
		domain := strings.TrimPrefix(record.Header().Name, "_acme-challenge.")
		if record.Header().Rrtype == dns.TypeTXT {
			a.UseRR(record, domain, &ACMEChallenge{
				Challenge: strings.Join(record.(*dns.TXT).Txt, ""),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &ACMEChallenge{}
		},
		acmechallenge_analyze,
		happydns.ServiceInfos{
			Name:        "ACME Challenge",
			Description: "Temporary record to prove that you control the sub-domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"temporary",
				"verification",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
