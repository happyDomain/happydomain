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
	"git.happydns.org/happyDomain/utils"
)

type GoogleVerif struct {
	Record *dns.TXT `json:"txt"`
}

func (s *GoogleVerif) GetNbResources() int {
	return 1
}

func (s *GoogleVerif) GenComment(origin string) string {
	return strings.TrimPrefix(strings.Join(s.Record.Txt, ""), "google-site-verification=")
}

func (s *GoogleVerif) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	return []dns.RR{s.Record}, nil
}

func googleverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		domain := record.NameFQDN
		if record.Type == "TXT" && strings.HasPrefix(record.GetTargetTXTJoined(), "google-site-verification=") {
			a.UseRR(record, domain, &GoogleVerif{
				Record: utils.RRRelative(record.ToRR(), domain).(*dns.TXT),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &GoogleVerif{}
		},
		googleverification_analyze,
		svcs.ServiceInfos{
			Name:        "Google Verification",
			Description: "Temporary record to prove that you control the domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"verification",
			},
		},
		2,
	)
}
