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

package svcs

import (
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
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

func (s *CNAME) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "CNAME", ttl, origin)
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))
	rrs = append(rrs, rr)
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

func (s *SpecialCNAME) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(utils.DomainJoin(s.SubDomain, domain), "CNAME", ttl, origin)
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))
	rrs = append(rrs, rr)
	return
}

func specialalias_analyze(a *Analyzer) error {
	// Try handle specials domains using CNAME
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME, Prefix: "_"}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.NameFQDN)
		if record.Type == "CNAME" && len(subdomains) == 4 {
			a.UseRR(record, subdomains[3], &SpecialCNAME{
				SubDomain: fmt.Sprintf("_%s._%s", subdomains[1], subdomains[2]),
				Target:    record.String(),
			})
		}
	}
	return nil
}

func alias_analyze(a *Analyzer) error {
	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeCNAME}) {
		if record.Type == "CNAME" {
			newrr := &CNAME{
				Target: strings.TrimSuffix(record.String(), "."+a.origin),
			}

			a.UseRR(record, record.NameFQDN, newrr)
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
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
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
			Description: "Maps an alias to another (canonical) domain.",
			Categories: []string{
				"internal",
			},
			RecordTypes: []uint16{
				dns.TypeCNAME,
			},
			Restrictions: ServiceRestrictions{
				Alone:  true,
				Single: true,
				NeedTypes: []uint16{
					dns.TypeCNAME,
				},
			},
		},
		99999998,
	)
}
