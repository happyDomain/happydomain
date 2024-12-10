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
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type EMail struct {
	MX      []svcs.MX             `json:"mx,omitempty" happydomain:"label=EMail Servers,required"`
	SPF     *svcs.SPF             `json:"spf,omitempty" happydomain:"label=Sender Policy Framework"`
	DKIM    map[string]*svcs.DKIM `json:"dkim,omitempty" happydomain:"label=Domain Keys,required"`
	DMARC   *svcs.DMARC           `json:"dmarc,omitempty" happydomain:"label=DMARC"`
	MTA_STS *svcs.MTA_STS         `json:"mta_sts,omitempty" happydomain:"label=Strict Transport Security"`
	TLS_RPT *svcs.TLS_RPT         `json:"tls_rpt,omitempty" happydomain:"label=TLS Reporting"`
}

func (s *EMail) GetNbResources() int {
	return len(s.MX)
}

func (s *EMail) GenComment(origin string) string {
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
		buffer.WriteString(strings.TrimSuffix(dn, "."+origin))
		if nb > 1 {
			buffer.WriteString(fmt.Sprintf(" ×%d", nb))
		}
	}

	if s.SPF != nil {
		buffer.WriteString(" + SPF")
	}

	if s.DKIM != nil {
		buffer.WriteString(" + DKIM")
	}

	if s.DMARC != nil {
		buffer.WriteString(" + DMARC")
	}

	if s.MTA_STS != nil {
		buffer.WriteString(" + MTA-STS")
	}

	if s.TLS_RPT != nil {
		buffer.WriteString(" + TLS Reporting")
	}

	return buffer.String()
}

func (s *EMail) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	if len(s.MX) > 0 {
		for _, mx := range s.MX {
			rc := utils.NewRecordConfig(domain, "MX", ttl, origin)
			rc.MxPreference = mx.Preference
			rc.SetTarget(utils.DomainFQDN(mx.Target, origin))

			rrs = append(rrs, rc)
		}
	}

	if s.SPF != nil {
		rrs = append(rrs, s.SPF.GenRRs(domain, ttl, origin)...)
	}

	for selector, d := range s.DKIM {
		rrs = append(rrs, (&svcs.DKIMRecord{
			DKIM:     *d,
			Selector: selector,
		}).GenRRs(domain, ttl, origin)...)
	}

	if s.DMARC != nil {
		rrs = append(rrs, s.DMARC.GenRRs(domain, ttl, origin)...)
	}

	if s.MTA_STS != nil {
		rrs = append(rrs, s.MTA_STS.GenRRs(domain, ttl, origin)...)
	}

	if s.TLS_RPT != nil {
		rrs = append(rrs, s.TLS_RPT.GenRRs(domain, ttl, origin)...)
	}
	return
}

func email_analyze(a *svcs.Analyzer) (err error) {
	services := map[string]*EMail{}

	// Handle only MX records
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX}) {
		if record.Type == "MX" {
			dn := record.NameFQDN

			if _, ok := services[dn]; !ok {
				services[dn] = &EMail{}
			}

			services[dn].MX = append(
				services[dn].MX,
				svcs.MX{
					Target:     record.GetTargetField(),
					Preference: record.MxPreference,
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
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &EMail{}
		},
		email_analyze,
		svcs.ServiceInfos{
			Name:        "E-Mail",
			Description: "Sends and receives e-mail with this domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeMX,
			},
			Tabs: true,
			Restrictions: svcs.ServiceRestrictions{
				Single: true,
				NeedTypes: []uint16{
					dns.TypeMX,
				},
			},
		},
		1,
	)
}
