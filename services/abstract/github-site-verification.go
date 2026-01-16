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

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type GithubOrgVerif struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *GithubOrgVerif) GetNbResources() int {
	return 1
}

func (s *GithubOrgVerif) GenComment() string {
	dnparts := strings.Split(s.Record.Hdr.Name, ".")
	if len(dnparts) > 0 {
		return strings.TrimSuffix(strings.TrimPrefix(dnparts[0], "_github-challenge-"), "-org")
	}
	return ""
}

func (s *GithubOrgVerif) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

func githubverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_github-challenge-"}) {
		dnparts := strings.Split(record.Header().Name, ".")
		if len(dnparts) > 1 {
			domain := strings.Join(dnparts[1:], ".")

			if record.Header().Rrtype == dns.TypeTXT {
				a.UseRR(record, domain, &GithubOrgVerif{
					Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
				})
			}
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &GithubOrgVerif{}
		},
		githubverification_analyze,
		happydns.ServiceInfos{
			Name:        "GitHub Verification",
			Description: "Temporary record to prove that you control the domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"verification",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
