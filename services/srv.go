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

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
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

func (s *SRV) GenComment() string {
	return fmt.Sprintf("%s:%d", s.Target, s.Port)
}

func (s *SRV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rr := utils.NewRecord(domain, "SRV", ttl, origin)
	rr.(*dns.SRV).Priority = s.Priority
	rr.(*dns.SRV).Weight = s.Weight
	rr.(*dns.SRV).Port = s.Port
	rr.(*dns.SRV).Target = utils.DomainFQDN(s.Target, origin)
	return []happydns.Record{rr}, nil
}

func ParseSRV(record *dns.SRV) (ret *SRV) {
	if record.Header().Rrtype == dns.TypeSRV {
		ret = &SRV{
			Priority: record.Priority,
			Weight:   record.Weight,
			Port:     record.Port,
			Target:   record.Target,
		}
	}

	return
}

var (
	SRV_DOMAIN = regexp.MustCompile(`^_([^.]+)\._(tcp|udp)(?:\.(.*))?$`)
)

type UnknownSRV struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	SRV   []*SRV `json:"srv"`
}

func (s *UnknownSRV) GetNbResources() int {
	return len(s.SRV)
}

func (s *UnknownSRV) GenComment() string {
	return fmt.Sprintf("%s (%s)", s.Name, s.Proto)
}

func (s *UnknownSRV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	var rrs []happydns.Record
	for _, service := range s.SRV {
		srv, err := service.GetRecords(utils.DomainJoin(fmt.Sprintf("_%s._%s", s.Name, s.Proto), domain), ttl, origin)
		if err != nil {
			return nil, err
		}
		rrs = append(rrs, srv...)
	}
	return rrs, nil
}

func srv_analyze(a *Analyzer) error {
	srvDomains := map[string]map[string]*UnknownSRV{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSRV}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.Header().Name)
		if len(subdomains) < 4 {
			continue
		}

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

		srv, ok := record.(*dns.SRV)
		if !ok {
			continue
		}

		// Make record relative
		srv.Target = utils.DomainRelative(srv.Target, a.GetOrigin())

		srvDomains[domain][svc].SRV = append(srvDomains[domain][svc].SRV, ParseSRV(utils.RRRelative(record, a.GetOrigin()).(*dns.SRV)))

		a.UseRR(
			record,
			subdomains[3],
			srvDomains[domain][svc],
		)
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &UnknownSRV{}
		},
		srv_analyze,
		happydns.ServiceInfos{
			Name:        "Service Record",
			Description: "Indicates to dedicated software the existance of the given service in the domain.",
			Categories: []string{
				"service",
			},
			RecordTypes: []uint16{
				dns.TypeSRV,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		99999,
	)
}
