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
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happydomain/model"
	"git.happydns.org/happydomain/services"
	"git.happydns.org/happydomain/utils"
)

type Origin struct {
	Ns          string        `json:"mname" happydomain:"label=Name Server,placeholder=ns0,required,description=The domain name of the name server that was the original or primary source of data for this zone."`
	Mbox        string        `json:"rname" happydomain:"label=Contact Email,required,description=A <domain-name> which specifies the mailbox of the person responsible for this zone."`
	Serial      uint32        `json:"serial" happydomain:"label=Zone Serial,required,description=The unsigned 32 bit version number of the original copy of the zone.  Zone transfers preserve this value.  This value wraps and should be compared using sequence space arithmetic."`
	Refresh     time.Duration `json:"refresh" happydomain:"label=Slave Refresh Time,required,description=The time interval before the zone should be refreshed by others name servers than the primary."`
	Retry       time.Duration `json:"retry" happydomain:"label=Retry Interval on failed refresh,required,description=The time interval a slave name server should elapse before a failed refresh should be retried."`
	Expire      time.Duration `json:"expire" happydomain:"label=Authoritative Expiry,required,description=Time value that specifies the upper limit on the time interval that can elapse before the zone is no longer authoritative."`
	Negttl      time.Duration `json:"nxttl" happydomain:"label=Negative Caching Time,required,description=Maximal time a resolver should cache a negative authoritative answer (such as NXDOMAIN ...)."`
	NameServers []string      `json:"ns" happydomain:"label=Zone's Name Servers"`
}

func (s *Origin) GetNbResources() int {
	return len(s.NameServers)
}

func (s *Origin) GenComment(origin string) string {
	ns := ""
	if s.NameServers != nil {
		ns = fmt.Sprintf(" + %d NS", len(s.NameServers))
	}

	return fmt.Sprintf("%s %s %d"+ns, strings.TrimSuffix(s.Ns, "."+origin), strings.TrimSuffix(s.Mbox, "."+origin), s.Serial)
}

func (s *Origin) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	rrs = append(rrs, &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   utils.DomainJoin(domain),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Ns:      utils.DomainFQDN(s.Ns, origin),
		Mbox:    utils.DomainFQDN(s.Mbox, origin),
		Serial:  s.Serial,
		Refresh: uint32(s.Refresh.Seconds()),
		Retry:   uint32(s.Retry.Seconds()),
		Expire:  uint32(s.Expire.Seconds()),
		Minttl:  uint32(s.Negttl.Seconds()),
	})
	for _, ns := range s.NameServers {
		rrs = append(rrs, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   utils.DomainJoin(domain),
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Ns: utils.DomainFQDN(ns, origin),
		})
	}
	return
}

func origin_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeSOA}) {
		if soa, ok := record.(*dns.SOA); ok {
			origin := &Origin{
				Ns:      soa.Ns,
				Mbox:    soa.Mbox,
				Serial:  soa.Serial,
				Refresh: time.Duration(soa.Refresh) * time.Second,
				Retry:   time.Duration(soa.Retry) * time.Second,
				Expire:  time.Duration(soa.Expire) * time.Second,
				Negttl:  time.Duration(soa.Minttl) * time.Second,
			}

			a.UseRR(
				record,
				soa.Header().Name,
				origin,
			)

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeNS, Domain: soa.Header().Name}) {
				if ns, ok := record.(*dns.NS); ok {
					origin.NameServers = append(origin.NameServers, ns.Ns)
					a.UseRR(
						record,
						ns.Header().Name,
						origin,
					)
				}
			}
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &Origin{}
		},
		origin_analyze,
		svcs.ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"internal",
			},
			Restrictions: svcs.ServiceRestrictions{
				RootOnly: true,
				Single:   true,
				NeedTypes: []uint16{
					dns.TypeSOA,
				},
			},
		},
		0,
	)
}
