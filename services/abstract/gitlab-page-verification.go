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

type GitlabPageVerif struct {
	Code string `happydomain:"label=Verification code given by Gitlab"`
}

func (s *GitlabPageVerif) GetNbResources() int {
	return 1
}

func (s *GitlabPageVerif) GenComment(origin string) string {
	return s.Code
}

func (s *GitlabPageVerif) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	if strings.Contains(s.Code, " TXT ") {
		s.Code = s.Code[strings.Index(s.Code, "gitlab-pages-verification-code="):]
	}
	domain = strings.TrimPrefix(domain, "_gitlab-pages-verification-code.")

	rr := utils.NewRecord("_gitlab-pages-verification-code."+domain, "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{"gitlab-pages-verification-code=" + strings.TrimPrefix(s.Code, "gitlab-pages-verification-code=")}

	return []happydns.Record{rr}, nil
}

func gitlabverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_gitlab-pages-verification-code"}) {
		domain := strings.TrimPrefix(record.Header().Name, "_gitlab-pages-verification-code.")
		if txt, ok := record.(*dns.TXT); ok && strings.HasPrefix(strings.Join(txt.Txt, ""), "gitlab-pages-verification-code=") {
			a.UseRR(record, domain, &GitlabPageVerif{
				Code: strings.TrimPrefix(strings.Join(txt.Txt, ""), "gitlab-pages-verification-code=")})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &GitlabPageVerif{}
		},
		gitlabverification_analyze,
		happydns.ServiceInfos{
			Name:        "Gitlab Pages Verification",
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
