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

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
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

func (s *Server) GenComment() string {
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

func (s *Server) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	if s.A != nil && len(s.A.A) != 0 {
		rrs = append(rrs, s.A)
	}
	if s.AAAA != nil && len(s.AAAA.AAAA) != 0 {
		rrs = append(rrs, s.AAAA)
	}

	for _, sshfp := range s.SSHFP {
		rrs = append(rrs, sshfp)
	}

	return
}

func server_analyze(a *svcs.Analyzer) error {
	pool := map[string][]happydns.Record{}

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeA}, svcs.AnalyzerRecordFilter{Type: dns.TypeAAAA}, svcs.AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.Header().Name

		pool[domain] = append(pool[domain], record)
	}

next_pool:
	for dn, rrs := range pool {
		s := &Server{}

		for _, rr := range rrs {
			if a, ok := rr.(*dns.A); ok {
				if s.A != nil {
					continue next_pool
				}

				s.A = a
			} else if aaaa, ok := rr.(*dns.AAAA); ok {
				if s.AAAA != nil {
					continue next_pool
				}

				s.AAAA = aaaa
			} else if sshfp, ok := rr.(*dns.SSHFP); ok {
				s.SSHFP = append(s.SSHFP, sshfp)
			}
		}

		// Register the use only now, to avoid registering multi-A/AAAA
		for _, rr := range rrs {
			if s.A != nil {
				s.A = helpers.RRRelativeSubdomain(s.A, a.GetOrigin(), dn).(*dns.A)
			}
			if s.AAAA != nil {
				s.AAAA = helpers.RRRelativeSubdomain(s.AAAA, a.GetOrigin(), dn).(*dns.AAAA)
			}
			for i := range s.SSHFP {
				s.SSHFP[i] = helpers.RRRelativeSubdomain(s.SSHFP[i], a.GetOrigin(), dn).(*dns.SSHFP)
			}

			a.UseRR(rr, dn, s)
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &Server{}
		},
		server_analyze,
		happydns.ServiceInfos{
			Name:        "Server",
			Description: "A system to respond to specific requests.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"server",
			},
			RecordTypes: []uint16{
				dns.TypeA,
				dns.TypeAAAA,
				dns.TypeSSHFP,
			},
			Restrictions: happydns.ServiceRestrictions{
				GLUE: true,
			},
		},
		100,
	)
}
