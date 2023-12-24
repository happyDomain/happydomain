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
	"regexp"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type SRV struct {
	Target   string `json:"target"`
	Port     uint16 `json:"port"`
	Weight   uint16 `json:"weight"`
	Priority uint16 `json:"priority"`
}

func (s *SRV) GetNbResources() int {
	return 1
}

func (s *SRV) GenComment(origin string) string {
	return fmt.Sprintf("%s:%d", strings.TrimSuffix(s.Target, "."+origin), s.Port)
}

func (s *SRV) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "SRV", ttl, origin)
	rr.SrvPriority = s.Priority
	rr.SrvWeight = s.Weight
	rr.SrvPort = s.Port
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))

	rrs = append(rrs, rr)
	return
}

func ParseSRV(record *models.RecordConfig) (ret *SRV) {
	if record.Type == "SRV" {
		ret = &SRV{
			Priority: record.SrvPriority,
			Weight:   record.SrvWeight,
			Port:     record.SrvPort,
			Target:   record.GetTargetField(),
		}
	}

	return
}

var (
	SRV_DOMAIN = regexp.MustCompile(`^_([^.]+)\._(tcp|udp)\.(.+)$`)
)

type UnknownSRV struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	SRV   []*SRV `json:"srv"`
}

func (s *UnknownSRV) GetNbResources() int {
	return len(s.SRV)
}

func (s *UnknownSRV) GenComment(origin string) string {
	return fmt.Sprintf("%s (%s)", s.Name, s.Proto)
}

func (s *UnknownSRV) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, service := range s.SRV {
		rrs = append(rrs, service.GenRRs(utils.DomainJoin(fmt.Sprintf("_%s._%s", s.Name, s.Proto), domain), ttl, origin)...)
	}
	return
}

func srv_analyze(a *Analyzer) error {
	srvDomains := map[string]map[string]*UnknownSRV{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSRV}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.NameFQDN)
		if srv := ParseSRV(record); len(subdomains) == 4 && srv != nil {
			svc := subdomains[1] + "." + subdomains[2]
			domain := subdomains[3]

			if _, ok := srvDomains[domain]; !ok {
				srvDomains[domain] = map[string]*UnknownSRV{}
			}

			if _, ok := srvDomains[domain][svc]; !ok {
				srvDomains[domain][svc] = &UnknownSRV{
					Name:  subdomains[1],
					Proto: subdomains[2],
				}
			}

			srvDomains[domain][svc].SRV = append(srvDomains[domain][svc].SRV, srv)

			a.UseRR(
				record,
				subdomains[3],
				srvDomains[domain][svc],
			)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &UnknownSRV{}
		},
		srv_analyze,
		ServiceInfos{
			Name:        "Service Record",
			Description: "Indicates to dedicated software the existance of the given service in the domain.",
			Categories: []string{
				"service",
			},
			RecordTypes: []uint16{
				dns.TypeSRV,
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		99999,
	)
}
