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
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type GithubOrgVerif struct {
	OrganizationName string `happydomain:"label=Organization Name"`
	Code             string `happydomain:"label=Code given by GitHub"`
}

func (s *GithubOrgVerif) GetNbResources() int {
	return 1
}

func (s *GithubOrgVerif) GenComment(origin string) string {
	return s.OrganizationName
}

func (s *GithubOrgVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rc := utils.NewRecordConfig(fmt.Sprintf("_github-challenge-%s-org.", strings.TrimSuffix(strings.TrimPrefix(s.OrganizationName, "_github-challenge-"), "-org"))+domain, "TXT", ttl, origin)
	rc.SetTargetTXT(s.Code)

	rrs = append(rrs, rc)
	return
}

func githubverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_github-challenge-"}) {
		dnparts := strings.Split(record.NameFQDN, ".")
		if len(dnparts) > 1 {
			domain := strings.Join(dnparts[1:], ".")
			org := strings.TrimSuffix(strings.TrimPrefix(dnparts[0], "_github-challenge-"), "-org")

			if record.Type == "TXT" {
				a.UseRR(record, domain, &GithubOrgVerif{
					OrganizationName: org,
					Code:             record.GetTargetTXTJoined(),
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
				"temporary",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
