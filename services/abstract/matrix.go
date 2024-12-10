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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type MatrixIM struct {
	Matrix []*svcs.SRV `json:"matrix"`
}

func (s *MatrixIM) GetNbResources() int {
	return len(s.Matrix)
}

func (s *MatrixIM) GenComment(origin string) string {
	dest := map[string][]uint16{}

destloop:
	for _, srv := range s.Matrix {
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
		dn = strings.TrimSuffix(dn, "."+origin)
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

func (s *MatrixIM) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, matrix := range s.Matrix {
		rrs = append(rrs, matrix.GenRRs(utils.DomainJoin("_matrix._tcp", domain), ttl, origin)...)
	}
	return
}

func matrix_analyze(a *svcs.Analyzer) error {
	matrixDomains := map[string]*MatrixIM{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Prefix: "_matrix._tcp.", Type: dns.TypeSRV}) {
		if srv := svcs.ParseSRV(record); srv != nil {
			domain := strings.TrimPrefix(record.NameFQDN, "_matrix._tcp.")

			if _, ok := matrixDomains[domain]; !ok {
				matrixDomains[domain] = &MatrixIM{}
			}

			matrixDomains[domain].Matrix = append(matrixDomains[domain].Matrix, srv)

			a.UseRR(
				record,
				domain,
				matrixDomains[domain],
			)
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &MatrixIM{}
		},
		matrix_analyze,
		svcs.ServiceInfos{
			Name:        "Matrix IM",
			Description: "Communicate on Matrix using your domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"service",
			},
			Restrictions: svcs.ServiceRestrictions{
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
