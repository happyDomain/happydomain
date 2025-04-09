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
	"net"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type Server struct {
	A     *net.IP       `json:"A,omitempty" happydomain:"label=ipv4,description=Server's IPv4"`
	AAAA  *net.IP       `json:"AAAA,omitempty" happydomain:"label=ipv6,description=Server's IPv6"`
	SSHFP []*svcs.SSHFP `json:"SSHFP,omitempty" happydomain:"label=SSH Fingerprint,description=Server's SSH fingerprint"`
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

	if s.A != nil && len(*s.A) != 0 {
		buffer.WriteString(s.A.String())
		if s.AAAA != nil && len(*s.AAAA) != 0 {
			buffer.WriteString("; ")
		}
	}

	if s.AAAA != nil && len(*s.AAAA) != 0 {
		buffer.WriteString(s.AAAA.String())
	}

	if s.SSHFP != nil {
		buffer.WriteString(fmt.Sprintf(" + %d SSHFP", len(s.SSHFP)))
	}

	return buffer.String()
}

func (s *Server) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	if s.A != nil && len(*s.A) != 0 {
		rr := utils.NewRecord(domain, "A", ttl, origin)
		rr.(*dns.A).A = *s.A

		rrs = append(rrs, rr)
	}
	if s.AAAA != nil && len(*s.AAAA) != 0 {
		rr := utils.NewRecord(domain, "AAAA", ttl, origin)
		rr.(*dns.AAAA).AAAA = *s.AAAA

		rrs = append(rrs, rr)
	}
	if len(s.SSHFP) > 0 {
		sshfp_rrs, err := (&svcs.SSHFPs{SSHFP: s.SSHFP}).GetRecords(domain, ttl, origin)
		if err != nil {
			return nil, fmt.Errorf("unable to generate SSHFP records: %w", err)
		}
		rrs = append(rrs, sshfp_rrs...)
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

				addr := a.A
				s.A = &addr
			} else if aaaa, ok := rr.(*dns.AAAA); ok {
				if s.AAAA != nil {
					continue next_pool
				}

				addr := aaaa.AAAA
				s.AAAA = &addr
			} else if sshfp, ok := rr.(*dns.SSHFP); ok {
				s.SSHFP = append(s.SSHFP, &svcs.SSHFP{
					Algorithm:   sshfp.Algorithm,
					Type:        sshfp.Type,
					FingerPrint: sshfp.FingerPrint,
				})
			}
		}

		// Register the use only now, to avoid registering multi-A/AAAA
		for _, rr := range rrs {
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
