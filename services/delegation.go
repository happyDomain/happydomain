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
	"fmt"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

type DS struct {
	KeyTag     uint16 `json:"keytag" happydns:"label=Key Tag"`
	Algorithm  uint8  `json:"algorithm" happydns:"label=Algorithm"`
	DigestType uint8  `json:"digestType" happydns:"label=Digest Type"`
	Digest     string `json:"digest" happydns:"label=Digest"`
}

type Delegation struct {
	NameServers []string `json:"ns" happydns:"label=Name Servers"`
	DS          []DS     `json:"ds" happydns:"label=Delegation Signer"`
}

func (s *Delegation) GetNbResources() int {
	return len(s.NameServers)
}

func (s *Delegation) GenComment(origin string) string {
	ds := ""
	if s.DS != nil {
		ds = fmt.Sprintf(" + %d DS", len(s.DS))
	}

	return fmt.Sprintf("%d name servers"+ds, len(s.NameServers))
}

func (s *Delegation) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	for _, ns := range s.NameServers {
		rrs = append(rrs, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Ns: utils.DomainFQDN(ns, origin),
		})
	}
	for _, ds := range s.DS {
		rrs = append(rrs, &dns.DS{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			KeyTag:     ds.KeyTag,
			Algorithm:  ds.Algorithm,
			DigestType: ds.DigestType,
			Digest:     ds.Digest,
		})
	}
	return
}

func delegation_analyze(a *Analyzer) error {
	delegations := map[string]*Delegation{}

	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeNS}) {
		if ns, ok := record.(*dns.NS); ok {
			if _, ok := delegations[ns.Header().Name]; !ok {
				delegations[ns.Header().Name] = &Delegation{}
			}

			delegations[ns.Header().Name].NameServers = append(delegations[ns.Header().Name].NameServers, ns.Ns)

			a.useRR(
				record,
				ns.Header().Name,
				delegations[ns.Header().Name],
			)
		}
	}

	for subdomain := range delegations {
		for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeDS, Domain: subdomain}) {
			if ds, ok := record.(*dns.DS); ok {
				delegations[subdomain].DS = append(delegations[subdomain].DS, DS{
					KeyTag:     ds.KeyTag,
					Algorithm:  ds.Algorithm,
					DigestType: ds.DigestType,
					Digest:     ds.Digest,
				})

				a.useRR(
					record,
					subdomain,
					delegations[subdomain],
				)
			}
		}
	}

	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &Delegation{}
		},
		delegation_analyze,
		ServiceInfos{
			Name:        "Delegation",
			Description: "Delegate this subdomain to another name server",
			Categories: []string{
				"internal",
			},
		},
		1,
	)
}
