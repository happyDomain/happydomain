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

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type MX struct {
	Target     string `json:"target"`
	Preference uint16 `json:"preference,omitempty"`
}

type MXs struct {
	MX []MX `json:"mx" happydomain:"label=EMail Servers,required"`
}

func (s *MXs) GetNbResources() int {
	return len(s.MX)
}

func (s *MXs) GenComment() string {
	poolMX := map[string]int{}

	for _, mx := range s.MX {
		labels := dns.SplitDomainName(mx.Target)
		nbLabel := len(labels)

		var dn string
		if nbLabel <= 2 {
			dn = mx.Target
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
		buffer.WriteString(dn)
		if nb > 1 {
			buffer.WriteString(fmt.Sprintf(" ×%d", nb))
		}
	}

	return buffer.String()
}

func (s *MXs) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	for _, mx := range s.MX {
		rr := utils.NewRecord(domain, "MX", ttl, origin)
		rr.(*dns.MX).Preference = mx.Preference
		rr.(*dns.MX).Mx = utils.DomainFQDN(mx.Target, origin)

		rrs = append(rrs, rr)
	}

	return
}

func mx_analyze(a *Analyzer) (err error) {
	services := map[string]*MXs{}

	// Handle only MX records
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeMX}) {
		dn := record.Header().Name

		if _, ok := services[dn]; !ok {
			services[dn] = &MXs{}
		}

		services[dn].MX = append(
			services[dn].MX,
			MX{
				Target:     record.(*dns.MX).Mx,
				Preference: record.(*dns.MX).Preference,
			},
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
		func() happydns.ServiceBody {
			return &MXs{}
		},
		mx_analyze,
		happydns.ServiceInfos{
			Name:        "E-Mail servers",
			Description: "Receives e-mail with this domain.",
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeMX,
			},
			Restrictions: happydns.ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeMX,
				},
			},
		},
		1,
	)
}
