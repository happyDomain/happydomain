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

type ScalewayChallenge struct {
	Challenge string
}

func (s *ScalewayChallenge) GetNbResources() int {
	return 1
}

func (s *ScalewayChallenge) GenComment(origin string) string {
	return s.Challenge
}

func (s *ScalewayChallenge) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rc := utils.NewRecordConfig(utils.DomainJoin("_scaleway-challenge", domain), "TXT", ttl, origin)
	rc.SetTargetTXT(s.Challenge)
	rrs = append(rrs, rc)
	return
}

func scalewaychallenge_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_scaleway-challenge"}) {
		domain := strings.TrimPrefix(record.NameFQDN, "_scaleway-challenge.")
		if record.Type == "TXT" {
			a.UseRR(record, domain, &ScalewayChallenge{
				Challenge: record.GetTargetTXTJoined(),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &ScalewayChallenge{}
		},
		scalewaychallenge_analyze,
		svcs.ServiceInfos{
			Name:        "Scaleway Challenge",
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
