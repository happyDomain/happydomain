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
	"git.happydns.org/happyDomain/services"
)

type Delegation struct {
	NameServers []*dns.NS `json:"ns"`
	DS          []*dns.DS `json:"ds"`
}

func (s *Delegation) GetNbResources() int {
	return len(s.NameServers)
}

func (s *Delegation) GenComment() string {
	ds := ""
	if s.DS != nil {
		ds = fmt.Sprintf(" + %d DS", len(s.DS))
	}

	return fmt.Sprintf("%d name servers"+ds, len(s.NameServers))
}

func (s *Delegation) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	for _, ns := range s.NameServers {
		rrs = append(rrs, ns)
	}
	for _, ds := range s.DS {
		rrs = append(rrs, ds)
	}
	return
}

func delegation_analyze(a *svcs.Analyzer) error {
	delegations := map[string]*Delegation{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS}) {
		// Origin cannot be a delegation
		if record.Header().Name == a.GetOrigin() {
			continue
		}

		if ns, ok := record.(*dns.NS); ok {
			dn := record.Header().Name
			if _, ok := delegations[record.Header().Name]; !ok {
				delegations[dn] = &Delegation{}
			}

			delegations[dn].NameServers = append(delegations[dn].NameServers, helpers.RRRelativeSubdomain(ns, a.GetOrigin(), dn).(*dns.NS))

			a.UseRR(
				record,
				dn,
				delegations[dn],
			)
		}
	}

	for subdomain := range delegations {
		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeDS, Domain: subdomain}) {
			if _, ok := record.(*dns.DS); ok {
				delegations[subdomain].DS = append(delegations[subdomain].DS, helpers.RRRelativeSubdomain(record, a.GetOrigin(), subdomain).(*dns.DS))

				a.UseRR(
					record,
					subdomain,
					delegations[subdomain],
				)
			}
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &Delegation{}
		},
		delegation_analyze,
		happydns.ServiceInfos{
			Name:        "Delegation",
			Description: "Delegate this subdomain to another name server",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeNS,
				dns.TypeDS,
			},
			Restrictions: happydns.ServiceRestrictions{
				Alone:       true,
				Leaf:        true,
				ExclusiveRR: []string{"abstract.Origin", "abstract.NSOnlyOrigin"},
				Single:      true,
				NeedTypes: []uint16{
					dns.TypeNS,
				},
			},
		},
		1,
	)
}
