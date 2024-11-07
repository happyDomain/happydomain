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

type GitlabPageVerif struct {
	Code string `happydomain:"label=Verification code given by Gitlab"`
}

func (s *GitlabPageVerif) GetNbResources() int {
	return 1
}

func (s *GitlabPageVerif) GenComment(origin string) string {
	return s.Code
}

func (s *GitlabPageVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	if strings.Contains(s.Code, " TXT ") {
		s.Code = s.Code[strings.Index(s.Code, "gitlab-pages-verification-code="):]
	}
	domain = strings.TrimPrefix(domain, "_gitlab-pages-verification-code.")

	rc := utils.NewRecordConfig("_gitlab-pages-verification-code."+domain, "TXT", ttl, origin)
	rc.SetTargetTXT("gitlab-pages-verification-code=" + strings.TrimPrefix(s.Code, "gitlab-pages-verification-code="))

	rrs = append(rrs, rc)
	return
}

func gitlabverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_gitlab-pages-verification-code"}) {
		domain := record.NameFQDN
		if record.Type == "TXT" && strings.HasPrefix(record.GetTargetTXTJoined(), "gitlab-pages-verification-code=") {
			a.UseRR(record, strings.TrimPrefix(domain, "_gitlab-pages-verification-code"), &GitlabPageVerif{
				Code: strings.TrimPrefix(record.GetTargetTXTJoined(), "gitlab-pages-verification-code="),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &GitlabPageVerif{}
		},
		gitlabverification_analyze,
		svcs.ServiceInfos{
			Name:        "Gitlab Pages Verification",
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
