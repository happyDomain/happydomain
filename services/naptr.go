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

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

type NAPTR struct {
	Record *dns.NAPTR `json:"naptr"`
}

func (ss *NAPTR) GetNbResources() int {
	return 1
}

func (ss *NAPTR) GenComment() string {
	return ss.Record.Service
}

func (ss *NAPTR) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	return []happydns.Record{ss.Record}, nil
}

func naptr_analyze(a *Analyzer) (err error) {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeNAPTR}) {
		if naptr, ok := record.(*dns.NAPTR); ok {
			domain := record.Header().Name
			err = a.UseRR(
				record,
				domain,
				&NAPTR{
					Record: helpers.RRRelativeSubdomain(naptr, a.GetOrigin(), domain).(*dns.NAPTR),
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
