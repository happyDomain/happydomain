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

type GoogleVerif struct {
	SiteVerification string `happydomain:"label=Site Verification"`
}

func (s *GoogleVerif) GetNbResources() int {
	return 1
}

func (s *GoogleVerif) GenComment(origin string) string {
	return s.SiteVerification
}

func (s *GoogleVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rc := utils.NewRecordConfig(domain, "TXT", ttl, origin)
	rc.SetTargetTXT("google-site-verification=" + strings.TrimPrefix(s.SiteVerification, "google-site-verification="))

	rrs = append(rrs, rc)
	return
}

func googleverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		domain := record.NameFQDN
		if record.Type == "TXT" && strings.HasPrefix(record.GetTargetTXTJoined(), "google-site-verification=") {
			a.UseRR(record, domain, &GoogleVerif{
				SiteVerification: strings.TrimPrefix(record.GetTargetTXTJoined(), "google-site-verification="),
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
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
