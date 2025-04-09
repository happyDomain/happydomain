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

func (s *SRV) GenComment(origin string) string {
	return fmt.Sprintf("%s:%d", strings.TrimSuffix(s.Target, "."+origin), s.Port)
}

func (s *SRV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rr := utils.NewRecord(domain, "SRV", ttl, origin)
	rr.(*dns.SRV).Priority = s.Priority
	rr.(*dns.SRV).Weight = s.Weight
	rr.(*dns.SRV).Port = s.Port
	rr.(*dns.SRV).Target = s.Target
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
	Records []*dns.SRV `json:"srv"`
}

func (s *UnknownSRV) GetNbResources() int {
	return len(s.Records)
}

func (s *UnknownSRV) GenComment(origin string) string {
	if len(s.Records) == 0 {
		return ""
	}

	subdomains := SRV_DOMAIN.FindStringSubmatch(s.Records[0].Hdr.Name)
	return fmt.Sprintf("%s (%s)", subdomains[1], subdomains[2])
}

func (s *UnknownSRV) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, r := range s.Records {
		srv := *r
		srv.Target = utils.DomainFQDN(srv.Target, origin)
		rrs[i] = &srv
	}
	return rrs, nil
}

func srv_analyze(a *Analyzer) error {
	srvDomains := map[string]map[string]*UnknownSRV{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSRV}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.Header().Name)
		svc := subdomains[1] + "." + subdomains[2]
		domain := subdomains[3]

		if _, ok := srvDomains[domain]; !ok {
			srvDomains[domain] = map[string]*UnknownSRV{}
		}

		if _, ok := srvDomains[domain][svc]; !ok {
			srvDomains[domain][svc] = &UnknownSRV{}
		}

		srv, ok := record.(*dns.SRV)
		if !ok {
			continue
		}

		// Make record relative
		srv.Target = utils.DomainRelative(srv.Target, a.GetOrigin())

		srvDomains[domain][svc].Records = append(srvDomains[domain][svc].Records, utils.RRRelative(record, a.GetOrigin()).(*dns.SRV))

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
