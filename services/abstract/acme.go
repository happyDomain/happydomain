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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type ACMEChallenge struct {
	Record *dns.TXT `json:"txt"`
}

func (s *ACMEChallenge) GetNbResources() int {
	return 1
}

func (s *ACMEChallenge) GenComment(origin string) string {
	return strings.Join(s.Record.Txt, "")
}

func (s *ACMEChallenge) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if !strings.HasPrefix(domain, "_acme-challenge") {
		domain = utils.DomainJoin("_acme-challenge", domain)
	}
	rc, err := models.RRtoRC(s.Record, origin)
	if err != nil {
		return nil, err
	}
	rrs = append(rrs, &rc)
	return
}

func acmechallenge_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_acme-challenge"}) {
		domain := strings.TrimPrefix(record.NameFQDN, "_acme-challenge.")
		if record.Type == "TXT" {
			a.UseRR(record, domain, &ACMEChallenge{
				Record: record.ToRR().(*dns.TXT),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &ACMEChallenge{}
		},
		acmechallenge_analyze,
		svcs.ServiceInfos{
			Name:        "ACME Challenge",
			Description: "Temporary record to prove that you control the sub-domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"temporary",
				"verification",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
