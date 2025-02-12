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
)

type GithubOrgVerif struct {
	Record *dns.TXT `json:"txt"`
}

func (s *GithubOrgVerif) GetNbResources() int {
	return 1
}

func (s *GithubOrgVerif) GenComment(origin string) string {
	dnparts := strings.Split(s.Record.Hdr.Name, ".")
	if len(dnparts) > 0 {
		return strings.TrimSuffix(strings.TrimPrefix(dnparts[0], "_github-challenge-"), "-org")
	}
	return ""
}

func (s *GithubOrgVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	rc, err := models.RRtoRC(s.Record, origin)
	if err != nil {
		return nil, err
	}
	rrs = append(rrs, &rc)
	return
}

func githubverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_github-challenge-"}) {
		dnparts := strings.Split(record.NameFQDN, ".")
		if len(dnparts) > 1 {
			domain := strings.Join(dnparts[1:], ".")

			if record.Type == "TXT" {
				a.UseRR(record, domain, &GithubOrgVerif{
					Record: record.ToRR().(*dns.TXT),
				})
			}
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &GithubOrgVerif{}
		},
		githubverification_analyze,
		svcs.ServiceInfos{
			Name:        "GitHub Verification",
			Description: "Temporary record to prove that you control the domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"verification",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
