// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package svcs

import (
	"bytes"
	"fmt"
	"net"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

type SSHFP struct {
	Algorithm   uint8  `json:"algorithm"`
	Type        uint8  `json:"type"`
	FingerPrint string `json:"fingerprint"`
}

type Server struct {
	A     *net.IP  `json:"A,omitempty" happydns:"label=ipv4,description=Server's IPv4"`
	AAAA  *net.IP  `json:"AAAA,omitempty" happydns:"label=ipv6,description=Server's IPv6"`
	SSHFP []*SSHFP `json:"SSHFP,omitempty" happydns:"label=SSH Fingerprint,description=Server's SSH fingerprint"`
}

func (s *Server) GetNbResources() int {
	return 1
}

func (s *Server) GenComment(origin string) string {
	var buffer bytes.Buffer

	if s.A != nil {
		buffer.WriteString(s.A.String())
		if s.AAAA != nil {
			buffer.WriteString("; ")
		}
	}

	if s.AAAA != nil {
		buffer.WriteString(s.AAAA.String())
	}

	if s.SSHFP != nil {
		buffer.WriteString(fmt.Sprintf(" + %d SSHFP", len(s.SSHFP)))
	}

	return buffer.String()
}

func (s *Server) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	if s.A != nil {
		rrs = append(rrs, &dns.A{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(domain),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: *s.A,
		})
	}
	if s.AAAA != nil {
		rrs = append(rrs, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(domain),
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: *s.AAAA,
		})
	}
	for _, sshfp := range s.SSHFP {
		rrs = append(rrs, &dns.SSHFP{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(domain),
				Rrtype: dns.TypeSSHFP,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Algorithm:   sshfp.Algorithm,
			Type:        sshfp.Type,
			FingerPrint: sshfp.FingerPrint,
		})
	}

	return
}

func server_analyze(a *Analyzer) error {
	pool := map[string][]dns.RR{}

	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeA}, AnalyzerRecordFilter{Type: dns.TypeAAAA}, AnalyzerRecordFilter{Type: dns.TypeSSHFP}) {
		domain := record.Header().Name

		pool[domain] = append(pool[domain], record)
	}

next_pool:
	for dn, rrs := range pool {
		s := &Server{}

		for _, rr := range rrs {
			if rr.Header().Rrtype == dns.TypeA {
				if s.A != nil {
					continue next_pool
				}

				s.A = &rr.(*dns.A).A
			} else if rr.Header().Rrtype == dns.TypeAAAA {
				if s.AAAA != nil {
					continue next_pool
				}

				s.AAAA = &rr.(*dns.AAAA).AAAA
			} else if rr.Header().Rrtype == dns.TypeSSHFP {
				sshfp := rr.(*dns.SSHFP)
				s.SSHFP = append(s.SSHFP, &SSHFP{
					Algorithm:   sshfp.Algorithm,
					Type:        sshfp.Type,
					FingerPrint: sshfp.FingerPrint,
				})
			}
		}

		for _, rr := range rrs {
			a.useRR(rr, dn, s)
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &Server{}
		},
		server_analyze,
		ServiceInfos{
			Name:        "Server",
			Description: "A computer will respond to some requests.",
			Categories: []string{
				"server",
			},
			Restrictions: ServiceRestrictions{
				GLUE: true,
			},
		},
		100,
	)
}
