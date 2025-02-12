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
	"bytes"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
)

type MXs struct {
	Records []*dns.MX `json:"mx"`
}

func (s *MXs) GetNbResources() int {
	return len(s.Records)
}

func (s *MXs) GenComment(origin string) string {
	poolMX := map[string]int{}

	for _, mx := range s.Records {
		labels := dns.SplitDomainName(mx.Mx)
		nbLabel := len(labels)

		var dn string
		if nbLabel <= 2 {
			dn = mx.Mx
		} else if len(labels[nbLabel-2]) < 4 {
			dn = strings.Join(labels[nbLabel-3:], ".") + "."
		} else {
			dn = strings.Join(labels[nbLabel-2:], ".") + "."
		}

		poolMX[dn] += 1
	}

	var buffer bytes.Buffer
	first := true

	for dn, nb := range poolMX {
		if !first {
			buffer.WriteString("; ")
		} else {
			first = !first
		}
		buffer.WriteString(strings.TrimSuffix(dn, "."+origin))
		if nb > 1 {
			buffer.WriteString(fmt.Sprintf(" ×%d", nb))
		}
	}

	return buffer.String()
}

func (s *MXs) GetRecords(domain string, ttl uint32, origin string) ([]dns.RR, error) {
	rrs := make([]dns.RR, len(s.Records))
	for i, r := range s.Records {
		rrs[i] = r
	}
	return rrs, nil
}

func mx_analyze(a *Analyzer) (err error) {
	services := map[string]*MXs{}

	// Handle only MX records
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeMX}) {
		dn := record.NameFQDN

		if _, ok := services[dn]; !ok {
			services[dn] = &MXs{}
		}

		services[dn].Records = append(
			services[dn].Records,
			record.ToRR().(*dns.MX),
		)

		err = a.UseRR(
			record,
			dn,
			services[dn],
		)
		if err != nil {
			return
		}
	}

	return
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &MXs{}
		},
		mx_analyze,
		ServiceInfos{
			Name:        "E-Mail servers",
			Description: "Receives e-mail with this domain.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeMX,
			},
			Restrictions: ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeMX,
				},
			},
		},
		1,
	)
}
