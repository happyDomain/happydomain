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
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type NSOnlyOrigin struct {
	NameServers []*dns.NS `json:"ns"`
}

func (s *NSOnlyOrigin) GetNbResources() int {
	return len(s.NameServers)
}

func (s *NSOnlyOrigin) GenComment(origin string) string {
	return fmt.Sprintf("%d NS", len(s.NameServers))
}

func (s *NSOnlyOrigin) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	rrs := make([]dns.RR, len(s.NameServers))
	for i, r := range s.NameServers {
		ns := *r
		ns.Ns = utils.DomainFQDN(ns.Ns, origin)
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

func (s *Origin) GenComment(origin string) string {
	if s.SOA == nil {
		return fmt.Sprintf("%d NS", len(s.NameServers))
	}

	ns := ""
	if s.NameServers != nil {
		ns = fmt.Sprintf(" + %d NS", len(s.NameServers))
	}

	return fmt.Sprintf("%s %s %d"+ns, strings.TrimSuffix(s.SOA.Ns, "."+origin), strings.TrimSuffix(s.SOA.Mbox, "."+origin), s.SOA.Serial)
}

func (s *Origin) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	rrs := make([]dns.RR, len(s.NameServers))
	for i, r := range s.NameServers {
		ns := *r
		ns.Ns = utils.DomainFQDN(ns.Ns, origin)
		rrs[i] = &ns
	}

	if s.SOA != nil {
		soa := *s.SOA
		soa.Ns = utils.DomainFQDN(soa.Ns, origin)
		soa.Mbox = utils.DomainFQDN(soa.Mbox, origin)
		rrs = append(rrs, &soa)
	}

	return rrs, nil
}

func origin_analyze(a *svcs.Analyzer) error {
	hasSOA := false

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeSOA}) {
		if record.Type == "SOA" {
			hasSOA = true

			// Make record relative
			record.SetTarget(utils.DomainRelative(record.GetTargetField(), a.GetOrigin()))
			record.SoaMbox = utils.DomainRelative(record.SoaMbox, a.GetOrigin())

			origin := &Origin{
				SOA: utils.RRRelative(record.ToRR(), record.NameFQDN).(*dns.SOA),
			}

			a.UseRR(
				record,
				record.NameFQDN,
				origin,
			)

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: record.NameFQDN}) {
				if record.Type == "NS" {
					// Make record relative
					record.SetTarget(utils.DomainRelative(record.GetTargetField(), a.GetOrigin()))

					origin.NameServers = append(origin.NameServers, utils.RRRelative(record.ToRR(), record.NameFQDN).(*dns.NS))
					a.UseRR(
						record,
						record.NameFQDN,
						origin,
					)
				}
			}
		}
	}

	if !hasSOA {
		origin := &NSOnlyOrigin{}

		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: a.GetOrigin()}) {
			if record.Type == "NS" {
				origin.NameServers = append(origin.NameServers, utils.RRRelative(record.ToRR(), record.NameFQDN).(*dns.NS))
				a.UseRR(
					record,
					record.NameFQDN,
					origin,
				)
			}
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &Origin{}
		},
		origin_analyze,
		svcs.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeSOA,
				dns.TypeNS,
			},
			Restrictions: svcs.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeSOA,
				},
			},
		},
		0,
	)
	svcs.RegisterService(
		func() happydns.Service {
			return &NSOnlyOrigin{}
		},
		nil,
		svcs.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      svcs.Hidden,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeNS,
			},
			Restrictions: svcs.ServiceRestrictions{
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
