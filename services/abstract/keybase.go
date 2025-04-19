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

type KeybaseVerif struct {
	Record *dns.TXT `json:"txt"`
}

func (s *KeybaseVerif) GetNbResources() int {
	return 1
}

func (s *KeybaseVerif) GenComment(origin string) string {
	return strings.TrimPrefix(strings.Join(s.Record.Txt, ""), "keybase-site-verification=")
}

func (s *KeybaseVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	rc, err := models.RRtoRC(s.Record, origin)
	if err != nil {
		return nil, err
	}
	rrs = append(rrs, &rc)
	return
}

func keybaseverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_keybase"}) {
		domain := strings.TrimPrefix(record.NameFQDN, "_keybase.")
		if record.Type == "TXT" {
			a.UseRR(record, domain, &KeybaseVerif{
				Record: record.ToRR().(*dns.TXT),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &KeybaseVerif{}
		},
		keybaseverification_analyze,
		svcs.ServiceInfos{
			Name:        "Keybase Verification",
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
