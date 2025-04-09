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

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type CNAME struct {
	Target string
}

func (s *CNAME) GetNbResources() int {
	return 1
}

func (s *CNAME) GenComment() string {
	return s.Target
}

func (s *CNAME) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	cname := utils.NewRecord(domain, "CNAME", ttl, origin)
	cname.(*dns.CNAME).Target = utils.DomainFQDN(s.Target, origin)
	return []happydns.Record{cname}, nil
}

type SpecialCNAME struct {
	SubDomain string
	Target    string
}

func (s *SpecialCNAME) GetNbResources() int {
	return 1
}

func (s *SpecialCNAME) GenComment() string {
	return "(" + s.SubDomain + ") -> " + s.Target
}

func (s *SpecialCNAME) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	cname := utils.NewRecord(utils.DomainJoin(s.SubDomain, domain), "CNAME", ttl, origin)
	cname.(*dns.CNAME).Target = utils.DomainFQDN(s.Target, origin)
	return []happydns.Record{cname}, nil
}

func specialalias_analyze(a *Analyzer) error {
	// Try handle specials domains using CNAME
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "_"}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.Header().Name)
		if cname, ok := record.(*dns.CNAME); ok && len(subdomains) == 4 {
			// Make record relative
			cname.Target = utils.DomainRelative(cname.Target, a.GetOrigin())

			a.UseRR(record, subdomains[3], &SpecialCNAME{
				SubDomain: fmt.Sprintf("_%s._%s", subdomains[1], subdomains[2]),
				Target:    cname.Target,
			})
		}
	}
	return nil
}

func alias_analyze(a *Analyzer) error {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME}) {
		if cname, ok := record.(*dns.CNAME); ok {
			// Make record relative
			cname.Target = utils.DomainRelative(cname.Target, a.GetOrigin())

			a.UseRR(record, record.Header().Name, &CNAME{
				Target: cname.Target,
			})
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &SpecialCNAME{}
		},
		specialalias_analyze,
		happydns.ServiceInfos{
			Name:        "SubAlias",
			Description: "A service alias to another domain/service.",
			Categories: []string{
				"alias",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
			},
		},
		99999997,
	)
	RegisterService(
		func() happydns.ServiceBody {
			return &CNAME{}
		},
		alias_analyze,
		happydns.ServiceInfos{
			Name:        "Alias",
			Description: "Maps an alias to another (canonical) domain.",
			Categories: []string{
				"alias",
			},
			RecordTypes: []uint16{
				dns.TypeCNAME,
			},
			Restrictions: happydns.ServiceRestrictions{
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
