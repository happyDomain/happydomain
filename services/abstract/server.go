// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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

package abstract

import (
	"bytes"
	"fmt"
	"net"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type Server struct {
	A     *net.IP       `json:"A,omitempty" happydomain:"label=ipv4,description=Server's IPv4"`
	AAAA  *net.IP       `json:"AAAA,omitempty" happydomain:"label=ipv6,description=Server's IPv6"`
	SSHFP []*svcs.SSHFP `json:"SSHFP,omitempty" happydomain:"label=SSH Fingerprint,description=Server's SSH fingerprint"`
}

func (s *Server) GetNbResources() int {
	return 1
}

func (s *Server) GenComment(origin string) string {
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

func (s *Server) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	if s.A != nil && len(*s.A) != 0 {
		rc := utils.NewRecordConfig(domain, "A", ttl, origin)
		rc.SetTargetIP(*s.A)

		rrs = append(rrs, rc)
	}
	if s.AAAA != nil && len(*s.AAAA) != 0 {
		rc := utils.NewRecordConfig(domain, "AAAA", ttl, origin)
		rc.SetTargetIP(*s.AAAA)

		rrs = append(rrs, rc)
	}
	for _, sshfp := range s.SSHFP {
		rc := utils.NewRecordConfig(domain, "SSHFP", ttl, origin)
		rc.SshfpAlgorithm = sshfp.Algorithm
		rc.SshfpFingerprint = sshfp.Type
		rc.SetTarget(sshfp.FingerPrint)

		rrs = append(rrs, rc)
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

				addr := rr.GetTargetIP()
				s.A = &addr
			} else if rr.Type == "AAAA" {
				if s.AAAA != nil {
					continue next_pool
				}

				addr := rr.GetTargetIP()
				s.AAAA = &addr
			} else if rr.Type == "SSHFP" {
				s.SSHFP = append(s.SSHFP, &svcs.SSHFP{
					Algorithm:   rr.SshfpAlgorithm,
					Type:        rr.SshfpFingerprint,
					FingerPrint: rr.GetTargetField(),
				})
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
