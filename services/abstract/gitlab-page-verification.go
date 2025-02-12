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

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type GitlabPageVerif struct {
	Record *dns.TXT `json:"txt"`
}

func (s *GitlabPageVerif) GetNbResources() int {
	return 1
}

func (s *GitlabPageVerif) GenComment(origin string) string {
	return strings.TrimPrefix(strings.Join(s.Record.Txt, ""), "gitlab-pages-verification-code=")
}

func (s *GitlabPageVerif) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	return []dns.RR{s.Record}, nil
}

func gitlabverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_gitlab-pages-verification-code"}) {
		domain := record.NameFQDN
		if record.Type == "TXT" && strings.HasPrefix(record.GetTargetTXTJoined(), "gitlab-pages-verification-code=") {
			a.UseRR(record, strings.TrimPrefix(domain, "_gitlab-pages-verification-code"), &GitlabPageVerif{
				Record: record.ToRR().(*dns.TXT),
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
				"verification",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
