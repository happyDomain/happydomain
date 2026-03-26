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
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	svc "git.happydns.org/happyDomain/internal/service"
)

type NSOnlyOrigin struct {
	NameServers []*dns.NS `json:"ns"`
}

func (s *NSOnlyOrigin) GetNbResources() int {
	return len(s.NameServers)
}

func (s *NSOnlyOrigin) GenComment() string {
	return fmt.Sprintf("%d NS", len(s.NameServers))
}

func (s *NSOnlyOrigin) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.NameServers))
	for i, r := range s.NameServers {
		ns := *r
		rrs[i] = &ns
	}
	return rrs, nil
}

type Origin struct {
	SOA         *dns.SOA  `json:"soa"`
	NameServers []*dns.NS `json:"ns"`
}

func (s *Origin) GetNbResources() int {
	if s.SOA == nil {
		return len(s.NameServers)
	} else {
		return len(s.NameServers) + 1
	}
}

func (s *Origin) GenComment() string {
	if s.SOA == nil {
		return fmt.Sprintf("%d NS", len(s.NameServers))
	}

	ns := ""
	if len(s.NameServers) > 0 {
		ns = fmt.Sprintf(" + %d NS", len(s.NameServers))
	}

	return fmt.Sprintf("%s %s %d"+ns, s.SOA.Ns, s.SOA.Mbox, s.SOA.Serial)
}

func (s *Origin) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.NameServers))
	for i, ns := range s.NameServers {
		rrs[i] = ns
	}

	if s.SOA != nil {
		rrs = append(rrs, s.SOA)
	}

	return rrs, nil
}

func origin_analyze(a *svc.Analyzer) error {
	hasSOA := false

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeSOA}) {
		if soa, ok := record.(*dns.SOA); ok {
			hasSOA = true

			domain := record.Header().Name
			origin := &Origin{
				SOA: helpers.RRRelativeSubdomain(soa, a.GetOrigin(), domain).(*dns.SOA),
			}

			if err := a.UseRR(
				record,
				domain,
				origin,
			); err != nil {
				return err
			}

			for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: domain}) {
				if ns, ok := record.(*dns.NS); ok {
					origin.NameServers = append(origin.NameServers, helpers.RRRelativeSubdomain(ns, a.GetOrigin(), domain).(*dns.NS))
					if err := a.UseRR(
						record,
						domain,
						origin,
					); err != nil {
						return err
					}
				}
			}
		}
	}

	if !hasSOA {
		origin := &NSOnlyOrigin{}

		for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: a.GetOrigin()}) {
			if ns, ok := record.(*dns.NS); ok {
				domain := record.Header().Name
				origin.NameServers = append(origin.NameServers, helpers.RRRelativeSubdomain(ns, a.GetOrigin(), domain).(*dns.NS))
				if err := a.UseRR(
					record,
					domain,
					origin,
				); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &Origin{}
		},
		origin_analyze,
		happydns.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeSOA,
				dns.TypeNS,
			},
			Restrictions: happydns.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeSOA,
				},
			},
		},
		0,
	)
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &NSOnlyOrigin{}
		},
		nil,
		happydns.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      happydns.SERVICE_FAMILY_HIDDEN,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeNS,
			},
			Restrictions: happydns.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeNS,
				},
			},
		},
		0,
	)
}
