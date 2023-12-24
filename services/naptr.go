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
	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type NAPTR struct {
	Order       uint16 `json:"order" happydomain:"label=Order,description=The order in which the records must be processed"`
	Preference  uint16 `json:"preference" happydomain:"label=Preference,description=The order in which the records with same order should be processed"`
	Flags       string `json:"flags" happydomain:"label=Flags,choices=S;A;U;P"`
	Service     string `json:"service" happydomain:"label=Service"`
	Regexp      string `json:"regexp" happydomain:"label=Regexp"`
	Replacement string `json:"replacement" happydomain:"label=Replacement"`
}

func (ss *NAPTR) GetNbResources() int {
	return 1
}

func (ss *NAPTR) GenComment(origin string) string {
	return ss.Service
}

func (ss *NAPTR) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "NAPTR", ttl, origin)
	rr.SetTargetNAPTR(ss.Order, ss.Preference, ss.Flags, ss.Service, ss.Regexp, ss.Replacement)
	rrs = append(rrs, rr)
	return
}

func naptr_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeNAPTR}) {
		if record.Type == "NAPTR" {
			err = a.UseRR(
				record,
				record.NameFQDN,
				&NAPTR{
					Order:       record.NaptrOrder,
					Preference:  record.NaptrPreference,
					Flags:       record.NaptrFlags,
					Service:     record.NaptrService,
					Regexp:      record.NaptrRegexp,
					Replacement: record.GetTargetField(),
				},
			)
			if err != nil {
				return
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &NAPTR{}
		},
		naptr_analyze,
		ServiceInfos{
			Name: "Naming Authority Pointer",
			RecordTypes: []uint16{
				dns.TypeNAPTR,
			},
			Restrictions: ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeNAPTR,
				},
			},
		},
		100,
	)
}
