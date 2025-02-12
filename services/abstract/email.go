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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
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
	var buffer bytes.Buffer

	buffer.WriteString((&svcs.MXs{MX: s.MX}).GenComment(origin))

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

func (s *EMail) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if len(s.MX) > 0 {
		mx_rrs, err := (&svcs.MXs{MX: s.MX}).GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate MX records: %w", err)
		}
		rrs = append(rrs, mx_rrs...)
	}

	if s.SPF != nil {
		spf_rrs, err := s.SPF.GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate SPF records: %w", err)
		}
		rrs = append(rrs, spf_rrs...)
	}

	for selector, d := range s.DKIM {
		dkim_rrs, err := (&svcs.DKIMRecord{
			DKIM:     *d,
			Selector: selector,
		}).GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate DKIM records for selector %q: %w", selector, err)
		}
		rrs = append(rrs, dkim_rrs...)
	}

	if s.DMARC != nil {
		dmarc_rrs, err := s.DMARC.GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate DMARC records: %w", err)
		}
		rrs = append(rrs, dmarc_rrs...)
	}

	if s.MTA_STS != nil {
		mta_sts_rrs, err := s.MTA_STS.GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate MTA-STS records: %w", err)
		}
		rrs = append(rrs, mta_sts_rrs...)
	}

	if s.TLS_RPT != nil {
		tls_rpt_rrs, err := s.TLS_RPT.GenRRs(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate TLS-RPT records: %w", err)
		}
		rrs = append(rrs, tls_rpt_rrs...)
	}
	return
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &EMail{}
		},
		nil,
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
