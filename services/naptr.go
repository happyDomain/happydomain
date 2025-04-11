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
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
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

func (ss *NAPTR) GenComment() string {
	return ss.Service
}

func (ss *NAPTR) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	rr := utils.NewRecord(domain, "NAPTR", ttl, origin)
	rr.(*dns.NAPTR).Order = ss.Order
	rr.(*dns.NAPTR).Preference = ss.Preference
	rr.(*dns.NAPTR).Flags = ss.Flags
	rr.(*dns.NAPTR).Service = ss.Service
	rr.(*dns.NAPTR).Regexp = ss.Regexp
	rr.(*dns.NAPTR).Replacement = ss.Replacement
	rrs = append(rrs, rr)
	return
}

func naptr_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeNAPTR}) {
		if naptr, ok := record.(*dns.NAPTR); ok {
			err = a.UseRR(
				record,
				record.Header().Name,
				&NAPTR{
					Order:       naptr.Order,
					Preference:  naptr.Preference,
					Flags:       naptr.Flags,
					Service:     naptr.Service,
					Regexp:      naptr.Regexp,
					Replacement: naptr.Replacement,
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
		func() happydns.ServiceBody {
			return &NAPTR{}
		},
		naptr_analyze,
		happydns.ServiceInfos{
			Name: "Naming Authority Pointer",
			Categories: []string{
				"telephony",
			},
			RecordTypes: []uint16{
				dns.TypeNAPTR,
			},
			Restrictions: happydns.ServiceRestrictions{
				NeedTypes: []uint16{
					dns.TypeNAPTR,
				},
			},
		},
		100,
	)
}
