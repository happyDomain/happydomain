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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type Delegation struct {
	NameServers []string  `json:"ns" happydomain:"label=Name Servers"`
	DS          []svcs.DS `json:"ds" happydomain:"label=Delegation Signer"`
}

func (s *Delegation) GetNbResources() int {
	return len(s.NameServers)
}

func (s *Delegation) GenComment(origin string) string {
	ds := ""
	if s.DS != nil {
		ds = fmt.Sprintf(" + %d DS", len(s.DS))
	}

	return fmt.Sprintf("%d name servers"+ds, len(s.NameServers))
}

func (s *Delegation) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, ns := range s.NameServers {
		rc := utils.NewRecordConfig(utils.DomainJoin(domain), "NS", ttl, origin)
		rc.SetTarget(utils.DomainFQDN(ns, origin))

		rrs = append(rrs, rc)
	}
	for _, ds := range s.DS {
		rc := utils.NewRecordConfig(utils.DomainJoin(domain), "DS", ttl, origin)
		rc.DsKeyTag = ds.KeyTag
		rc.DsAlgorithm = ds.Algorithm
		rc.DsDigestType = ds.DigestType
		rc.DsDigest = ds.Digest

		rrs = append(rrs, rc)
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

			delegations[record.NameFQDN].NameServers = append(delegations[record.NameFQDN].NameServers, record.GetTargetField())

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
				delegations[subdomain].DS = append(delegations[subdomain].DS, svcs.DS{
					KeyTag:     record.DsKeyTag,
					Algorithm:  record.DsAlgorithm,
					DigestType: record.DsDigestType,
					Digest:     record.DsDigest,
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
		func() happydns.Service {
			return &Delegation{}
		},
		delegation_analyze,
		svcs.ServiceInfos{
			Name:        "Delegation",
			Description: "Delegate this subdomain to another name server",
			Family:      svcs.Abstract,
			Categories: []string{
				"internal",
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
