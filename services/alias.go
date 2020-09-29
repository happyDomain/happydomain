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
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/utils"
)

type CNAME struct {
	Target string
}

func (s *CNAME) GetNbResources() int {
	return 1
}

func (s *CNAME) GenComment(origin string) string {
	return strings.TrimSuffix(s.Target, "."+origin)
}

func (s *CNAME) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	rrs = append(rrs, &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   utils.DomainJoin(domain),
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Target: utils.DomainFQDN(s.Target, origin),
	})
	return
}

type SpecialCNAME struct {
	SubDomain string
	Target    string
}

func (s *SpecialCNAME) GetNbResources() int {
	return 1
}

func (s *SpecialCNAME) GenComment(origin string) string {
	return "(" + s.SubDomain + ") -> " + strings.TrimSuffix(s.Target, "."+origin)
}

func (s *SpecialCNAME) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	rrs = append(rrs, &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   utils.DomainJoin(s.SubDomain, domain),
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Target: utils.DomainFQDN(s.Target, origin),
	})
	return
}

func specialalias_analyze(a *Analyzer) error {
	// Try handle specials domains using CNAME
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "_"}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.Header().Name)
		if cname, ok := record.(*dns.CNAME); len(subdomains) == 4 && ok {
			a.useRR(record, subdomains[3], &SpecialCNAME{
				SubDomain: fmt.Sprintf("_%s._%s", subdomains[1], subdomains[2]),
				Target:    cname.Target,
			})
		}
	}
	return nil
}

func alias_analyze(a *Analyzer) error {
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME}) {
		if cname, ok := record.(*dns.CNAME); ok {
			newrr := &CNAME{
				Target: strings.TrimSuffix(cname.Target, "."+a.origin),
			}

			a.useRR(record, cname.Header().Name, newrr)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &SpecialCNAME{}
		},
		specialalias_analyze,
		ServiceInfos{
			Name:        "SubAlias",
			Description: "A service alias to another domain/service.",
			Categories: []string{
				"internal",
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
			},
		},
		99999997,
	)
	RegisterService(
		func() happydns.Service {
			return &CNAME{}
		},
		alias_analyze,
		ServiceInfos{
			Name:        "Alias",
			Description: "An alias to another domain.",
			Categories: []string{
				"internal",
			},
			Restrictions: ServiceRestrictions{
				Alone:  true,
				Single: true,
			},
		},
		99999998,
	)
}
