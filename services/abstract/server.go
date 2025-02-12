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
	"git.happydns.org/happyDomain/utils"
)

type Server struct {
	A     *dns.A       `json:"A,omitempty"`
	AAAA  *dns.AAAA    `json:"AAAA,omitempty"`
	SSHFP []*dns.SSHFP `json:"SSHFP,omitempty"`
}

func (s *Server) GetNbResources() int {
	i := 0

	if s.A != nil {
		i += 1
	}

	if s.AAAA != nil {
		i += 1
	}

	return i + len(s.SSHFP)
}

func (s *Server) GenComment(origin string) string {
	var buffer bytes.Buffer

	if s.A != nil && len(s.A.A) != 0 {
		buffer.WriteString(s.A.A.String())
		if s.AAAA != nil && len(s.AAAA.AAAA) != 0 {
			buffer.WriteString("; ")
		}
	}

	if s.AAAA != nil && len(s.AAAA.AAAA) != 0 {
		buffer.WriteString(s.AAAA.AAAA.String())
	}

	if s.SSHFP != nil {
		buffer.WriteString(fmt.Sprintf(" + %d SSHFP", len(s.SSHFP)))
	}

	return buffer.String()
}

func (s *Server) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if s.A != nil && len(s.A.A) != 0 {
		rc, err := models.RRtoRC(s.A, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate A record: %w", err)
		}
		rrs = append(rrs, &rc)
	}
	if s.AAAA != nil && len(s.AAAA.AAAA) != 0 {
		rc, err := models.RRtoRC(s.AAAA, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate AAAA record: %w", err)
		}
		rrs = append(rrs, &rc)
	}
	if len(s.SSHFP) > 0 {
		sshfp_rrs, err := utils.RRstoRCs(s.SSHFP, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate SSHFP records: %w", err)
		}
		rrs = append(rrs, sshfp_rrs...)
	}

	return
}

func server_analyze(a *svcs.Analyzer) error {
	pool := map[string]models.Records{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeA}, svcs.AnalyzerRecordFilter{Type: dns.TypeAAAA}, svcs.AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.NameFQDN

		pool[domain] = append(pool[domain], record)
	}

next_pool:
	for dn, rrs := range pool {
		s := &Server{}

		for _, rr := range rrs {
			if rr.Type == "A" {
				if s.A != nil {
					continue next_pool
				}

				s.A = rr.ToRR().(*dns.A)
			} else if rr.Type == "AAAA" {
				if s.AAAA != nil {
					continue next_pool
				}

				s.AAAA = rr.ToRR().(*dns.AAAA)
			} else if rr.Type == "SSHFP" {
				s.SSHFP = append(s.SSHFP, rr.ToRR().(*dns.SSHFP))
			}
		}

		for _, rr := range rrs {
			a.UseRR(rr, dn, s)
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &Server{}
		},
		server_analyze,
		svcs.ServiceInfos{
			Name:        "Server",
			Description: "A system to respond to specific requests.",
			Family:      svcs.Abstract,
			Categories: []string{
				"server",
			},
			RecordTypes: []uint16{
				dns.TypeA,
				dns.TypeAAAA,
				dns.TypeSSHFP,
			},
			Restrictions: svcs.ServiceRestrictions{
				GLUE: true,
			},
		},
		100,
	)
}
