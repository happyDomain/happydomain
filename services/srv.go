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
	"regexp"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/utils"
)

type SRV struct {
	Target   string `json:"target"`
	Port     uint16 `json:"port"`
	Weight   uint16 `json:"weight"`
	Priority uint16 `json:"priority"`
}

func (s *SRV) GetNbResources() int {
	return 1
}

func (s *SRV) GenComment(origin string) string {
	return fmt.Sprintf("%s:%d", strings.TrimSuffix(s.Target, "."+origin), s.Port)
}

func (s *SRV) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rr := utils.NewRecordConfig(domain, "SRV", ttl, origin)
	rr.SrvPriority = s.Priority
	rr.SrvWeight = s.Weight
	rr.SrvPort = s.Port
	rr.SetTarget(utils.DomainFQDN(s.Target, origin))

	rrs = append(rrs, rr)
	return
}

func ParseSRV(record *models.RecordConfig) (ret *SRV) {
	if record.Type == "SRV" {
		ret = &SRV{
			Priority: record.SrvPriority,
			Weight:   record.SrvWeight,
			Port:     record.SrvPort,
			Target:   record.GetTargetField(),
		}
	}

	return
}

var (
	SRV_DOMAIN = regexp.MustCompile(`^_([^.]+)\._(tcp|udp)\.(.+)$`)
)

type UnknownSRV struct {
	Name  string `json:"name"`
	Proto string `json:"proto"`
	SRV   []*SRV `json:"srv"`
}

func (s *UnknownSRV) GetNbResources() int {
	return len(s.SRV)
}

func (s *UnknownSRV) GenComment(origin string) string {
	return fmt.Sprintf("%s (%s)", s.Name, s.Proto)
}

func (s *UnknownSRV) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, service := range s.SRV {
		rrs = append(rrs, service.GenRRs(utils.DomainJoin(fmt.Sprintf("_%s._%s", s.Name, s.Proto), domain), ttl, origin)...)
	}
	return
}

func srv_analyze(a *Analyzer) error {
	srvDomains := map[string]map[string]*UnknownSRV{}

	for _, record := range a.SearchRR(AnalyzerRecordFilter{Type: dns.TypeSRV}) {
		subdomains := SRV_DOMAIN.FindStringSubmatch(record.NameFQDN)
		if srv := ParseSRV(record); len(subdomains) == 4 && srv != nil {
			svc := subdomains[1] + "." + subdomains[2]
			domain := subdomains[3]

			if _, ok := srvDomains[domain]; !ok {
				srvDomains[domain] = map[string]*UnknownSRV{}
			}

			if _, ok := srvDomains[domain][svc]; !ok {
				srvDomains[domain][svc] = &UnknownSRV{
					Name:  subdomains[1],
					Proto: subdomains[2],
				}
			}

			srvDomains[domain][svc].SRV = append(srvDomains[domain][svc].SRV, srv)

			a.UseRR(
				record,
				subdomains[3],
				srvDomains[domain][svc],
			)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &UnknownSRV{}
		},
		srv_analyze,
		ServiceInfos{
			Name:        "Service Record",
			Description: "Indicates to dedicated software the existance of the given service in the domain.",
			Categories: []string{
				"service",
			},
			RecordTypes: []uint16{
				dns.TypeSRV,
			},
			Restrictions: ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSRV,
				},
			},
		},
		99999,
	)
}
