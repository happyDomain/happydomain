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

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type Delegation struct {
	NameServers []string  `json:"ns" happydomain:"label=Name Servers"`
	DS          []svcs.DS `json:"ds" happydomain:"label=Delegation Signer"`
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
	for _, r := range s.NameServers {
		ns := utils.NewRecord(utils.DomainJoin(domain), "NS", ttl, origin)
		ns.(*dns.NS).Ns = utils.DomainFQDN(r, origin)
		rrs = append(rrs, ns)
	}
	for _, ds := range s.DS {
		rr := utils.NewRecord(utils.DomainJoin(domain), "DS", ttl, origin)
		rr.(*dns.DS).KeyTag = ds.KeyTag
		rr.(*dns.DS).Algorithm = ds.Algorithm
		rr.(*dns.DS).DigestType = ds.DigestType
		rr.(*dns.DS).Digest = ds.Digest

		rrs = append(rrs, rr)
	}
	return
}

func delegation_analyze(a *svcs.Analyzer) error {
	delegations := map[string]*Delegation{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS}) {
		if record.Header().Name == strings.TrimSuffix(a.GetOrigin(), ".") {
			continue
		}

		if ns, ok := record.(*dns.NS); ok {
			dn := record.Header().Name
			if _, ok := delegations[record.Header().Name]; !ok {
				delegations[dn] = &Delegation{}
			}

			// Make record relative
			ns.Ns = utils.DomainRelative(ns.Ns, a.GetOrigin())

			delegations[dn].NameServers = append(delegations[dn].NameServers, ns.Ns)

			a.UseRR(
				record,
				record.Header().Name,
				delegations[dn],
			)
		}
	}

	for subdomain := range delegations {
		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeDS, Domain: subdomain}) {
			if ds, ok := record.(*dns.DS); ok {
				delegations[subdomain].DS = append(delegations[subdomain].DS, svcs.DS{
					KeyTag:     ds.KeyTag,
					Algorithm:  ds.Algorithm,
					DigestType: ds.DigestType,
					Digest:     ds.Digest,
				})

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
				ExclusiveRR: []string{"abstract.Origin"},
				Single:      true,
				NeedTypes: []uint16{
					dns.TypeNS,
				},
			},
		},
		1,
	)
}
