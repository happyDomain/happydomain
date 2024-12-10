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

package svcs

import (
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type CNAME struct {
	Target string
}

func (s *CNAME) GetNbResources() int {
	return 1
}

func (s *CNAME) GenComment(origin string) string {
	return strings.TrimSuffix(s.Target, "."+origin)
}

func (s *CNAME) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "CNAME", ttl, origin)
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))
	rrs = append(rrs, rr)
	return
}

type SpecialCNAME struct {
	SubDomain string
	Target    string
}

func (s *SpecialCNAME) GetNbResources() int {
	return 1
}

func (s *SpecialCNAME) GenComment(origin string) string {
	return "(" + s.SubDomain + ") -> " + strings.TrimSuffix(s.Target, "."+origin)
}

func (s *SpecialCNAME) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(utils.DomainJoin(s.SubDomain, domain), "CNAME", ttl, origin)
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))
	rrs = append(rrs, rr)
	return
}

func specialalias_analyze(a *Analyzer) error {
	// Try handle specials domains using CNAME
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "_"}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.NameFQDN)
		if record.Type == "CNAME" && len(subdomains) == 4 {
			a.UseRR(record, subdomains[3], &SpecialCNAME{
				SubDomain: fmt.Sprintf("_%s._%s", subdomains[1], subdomains[2]),
				Target:    record.String(),
			})
		}
	}
	return nil
}

func alias_analyze(a *Analyzer) error {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME}) {
		if record.Type == "CNAME" {
			newrr := &CNAME{
				Target: strings.TrimSuffix(record.String(), "."+a.origin),
			}

			a.UseRR(record, record.NameFQDN, newrr)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &SpecialCNAME{}
		},
		specialalias_analyze,
		ServiceInfos{
			Name:        "SubAlias",
			Description: "A service alias to another domain/service.",
			Categories: []string{
				"alias",
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
			},
		},
		99999997,
	)
	RegisterService(
		func() happydns.Service {
			return &CNAME{}
		},
		alias_analyze,
		ServiceInfos{
			Name:        "Alias",
			Description: "Maps an alias to another (canonical) domain.",
			Categories: []string{
				"alias",
			},
			RecordTypes: []uint16{
				dns.TypeCNAME,
			},
			Restrictions: ServiceRestrictions{
				Alone:  true,
				Single: true,
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
			},
		},
		99999998,
	)
}
