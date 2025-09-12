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
	"bytes"
	"strconv"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type MatrixIM struct {
	Records []*dns.SRV `json:"records"`
}

func (s *MatrixIM) GetNbResources() int {
	return len(s.Records)
}

func (s *MatrixIM) GenComment() string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range s.Records {
		for _, port := range dest[srv.Target] {
			if port == srv.Port {
				continue destloop
			}
		}
		dest[srv.Target] = append(dest[srv.Target], srv.Port)
	}

	var buffer bytes.Buffer
	first := true
	for dn, ports := range dest {
		if !first {
			buffer.WriteString("; ")
		} else {
			first = !first
		}
		buffer.WriteString(dn)
		buffer.WriteString(" (")
		firstport := true
		for _, port := range ports {
			if !firstport {
				buffer.WriteString(", ")
			} else {
				firstport = !firstport
			}
			buffer.WriteString(strconv.Itoa(int(port)))
		}
		buffer.WriteString(")")
	}

	return buffer.String()
}

func (s *MatrixIM) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	rrs := make([]happydns.Record, len(s.Records))
	for i, srv := range s.Records {
		rrs[i] = srv
	}
	return rrs, nil
}

func matrix_analyze(a *svcs.Analyzer) error {
	matrixDomains := map[string]*MatrixIM{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_matrix._tcp.", Type: dns.TypeSRV}) {
		domain := strings.TrimPrefix(record.Header().Name, "_matrix._tcp.")

		if _, ok := matrixDomains[domain]; !ok {
			matrixDomains[domain] = &MatrixIM{}
		}

		if srv, ok := record.(*dns.SRV); ok {
			matrixDomains[domain].Records = append(matrixDomains[domain].Records, helpers.RRRelative(srv, domain).(*dns.SRV))

			a.UseRR(
				srv,
				domain,
				matrixDomains[domain],
			)
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &MatrixIM{}
		},
		matrix_analyze,
		happydns.ServiceInfos{
			Name:        "Matrix IM",
			Description: "Communicate on Matrix using your domain.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"service",
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		1,
	)
}
