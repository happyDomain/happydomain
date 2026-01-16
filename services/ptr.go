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

type PTR struct {
	Record *dns.PTR
}

func (s *PTR) GetNbResources() int {
	return 1
}

func (s *PTR) GenComment() string {
	return s.Record.Ptr
}

func (s *PTR) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	return []happydns.Record{s.Record}, nil
}

func pointer_analyze(a *Analyzer) error {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypePTR}) {
		if ptr, ok := record.(*dns.PTR); ok {
			domain := record.Header().Name
			newrr := &PTR{
				Record: helpers.RRRelativeSubdomain(ptr, a.GetOrigin(), domain).(*dns.PTR),
			}

			a.UseRR(record, domain, newrr)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.ServiceBody {
			return &PTR{}
		},
		pointer_analyze,
		happydns.ServiceInfos{
			Name:        "Pointer",
			Description: "A pointer to another domain.",
			Categories: []string{
				"domain name",
			},
			RecordTypes: []uint16{
				dns.TypePTR,
			},
			Restrictions: happydns.ServiceRestrictions{
				Alone:  true,
				Single: true,
				NeedTypes: []uint16{
					dns.TypePTR,
				},
			},
		},
		99999998,
	)
}
