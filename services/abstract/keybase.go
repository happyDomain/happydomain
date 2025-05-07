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

type KeybaseVerif struct {
	SiteVerification string `happydomain:"label=Site Verification"`
}

func (s *KeybaseVerif) GetNbResources() int {
	return 1
}

func (s *KeybaseVerif) GenComment() string {
	return s.SiteVerification
}

func (s *KeybaseVerif) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rr := helpers.NewRecord(helpers.DomainJoin("_keybase", domain), "TXT", ttl, origin)
	rr.(*dns.TXT).Txt = []string{"keybase-site-verification=" + strings.TrimPrefix(s.SiteVerification, "keybase-site-verification=")}
	return []happydns.Record{rr}, nil
}

func keybaseverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_keybase"}) {
		domain := strings.TrimPrefix(record.Header().Name, "_keybase.")
		if txt, ok := record.(*dns.TXT); ok {
			a.UseRR(record, domain, &KeybaseVerif{
				SiteVerification: strings.TrimPrefix(strings.Join(txt.Txt, ""), "keybase-site-verification="),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
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
