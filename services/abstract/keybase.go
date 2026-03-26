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
	svc "git.happydns.org/happyDomain/internal/service"
)

type KeybaseVerif struct {
	Record *happydns.TXT `json:"txt"`
}

func (s *KeybaseVerif) GetNbResources() int {
	return 1
}

func (s *KeybaseVerif) GenComment() string {
	return strings.TrimPrefix(s.Record.Txt, "keybase-site-verification=")
}

func (s *KeybaseVerif) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{s.Record}, nil
}

func keybaseverification_analyze(a *svc.Analyzer) error {
	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_keybase"}) {
		domain := strings.TrimPrefix(record.Header().Name, "_keybase.")
		if record.Header().Rrtype == dns.TypeTXT {
			if err := a.UseRR(record, domain, &KeybaseVerif{
				Record: helpers.RRRelativeSubdomain(record, a.GetOrigin(), domain).(*happydns.TXT),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &KeybaseVerif{}
		},
		keybaseverification_analyze,
		happydns.ServiceInfos{
			Name:        "Keybase Verification",
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
