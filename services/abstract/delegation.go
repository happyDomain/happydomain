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
)

type Delegation struct {
	NameServers []*dns.NS `json:"ns"`
	DS          []*dns.DS `json:"ds"`
}

func (s *Delegation) GetNbResources() int {
	return len(s.NameServers) + len(s.DS)
}

func (s *Delegation) GenComment(origin string) string {
	ds := ""
	if s.DS != nil {
		ds = fmt.Sprintf(" + %d DS", len(s.DS))
	}

	return fmt.Sprintf("%d name servers"+ds, len(s.NameServers))
}

func (s *Delegation) GetRecords(domain string, ttl uint32, origin string) (rrs []dns.RR, e error) {
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
		if record.NameFQDN == strings.TrimSuffix(a.GetOrigin(), ".") {
			continue
		}

		if record.Type == "NS" {
			if _, ok := delegations[record.NameFQDN]; !ok {
				delegations[record.NameFQDN] = &Delegation{}
			}

			delegations[record.NameFQDN].NameServers = append(delegations[record.NameFQDN].NameServers, record.ToRR().(*dns.NS))

			a.UseRR(
				record,
				record.NameFQDN,
				delegations[record.NameFQDN],
			)
		}
	}

	for subdomain := range delegations {
		for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeDS, Domain: subdomain}) {
			if record.Type == "DS" {
				delegations[subdomain].DS = append(delegations[subdomain].DS, record.ToRR().(*dns.DS))

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
		func() happydns.Service {
			return &Delegation{}
		},
		delegation_analyze,
		svcs.ServiceInfos{
			Name:        "Delegation",
			Description: "Delegate this subdomain to another name server",
			Family:      svcs.Abstract,
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypeNS,
				dns.TypeDS,
			},
			Restrictions: svcs.ServiceRestrictions{
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
