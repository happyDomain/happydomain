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

type GoogleVerif struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *GoogleVerif) GetNbResources() int {
	return 1
}

func (s *GoogleVerif) GenComment() string {
	return strings.TrimPrefix(s.Record.Txt, "google-site-verification=")
}

func (s *GoogleVerif) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

func googleverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		domain := record.Header().Name
		if txt, ok := record.(*happydns.TXT); ok && strings.HasPrefix(txt.Txt, "google-site-verification=") {
			a.UseRR(record, domain, &GoogleVerif{
				Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &GoogleVerif{}
		},
		googleverification_analyze,
		happydns.ServiceInfos{
			Name:        "Google Verification",
			Description: "Temporary record to prove that you control the domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"verification",
			},
		},
		2,
	)
}
