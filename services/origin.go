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
	"time"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
)

type Origin struct {
	Ns      string        `json:"mname" happydns:"label=Name Server,placeholder=ns0,required,description=The domain name of the name server that was the original or primary source of data for this zone."`
	Mbox    string        `json:"rname" happydns:"label=Contact Email,required,description=A <domain-name> which specifies the mailbox of the person responsible for this zone."`
	Serial  uint32        `json:"serial" happydns:"label=Zone Serial,required,description=The unsigned 32 bit version number of the original copy of the zone.  Zone transfers preserve this value.  This value wraps and should be compared using sequence space arithmetic."`
	Refresh time.Duration `json:"refresh" happydns:"label=Slave Refresh Time,required,description=The time interval before the zone should be refreshed by others name servers than the primary."`
	Retry   time.Duration `json:"retry" happydns:"label=Retry Interval on failed refresh,required,description=The time interval a slave name server should elapse before a failed refresh should be retried."`
	Expire  time.Duration `json:"expire" happydns:"label=Authoritative Expiry,required,description=Time value that specifies the upper limit on the time interval that can elapse before the zone is no longer authoritative."`
	Negttl  time.Duration `json:"nxttl" happydns:"label=Negative Caching Time,required,description=Maximal time a resolver should cache a negative authoritative answer (such as NXDOMAIN ...)."`
}

func (s *Origin) GetNbResources() int {
	return 1
}

func (s *Origin) GenComment(origin string) string {
	return fmt.Sprintf("%s %s %d", strings.TrimSuffix(s.Ns, "."+origin), strings.TrimSuffix(s.Mbox, "."+origin), s.Serial)
}

func (s *Origin) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	ns := s.Ns
	if ns[len(ns)-1] != '.' {
		ns += origin
	}
	mbox := s.Mbox
	if mbox[len(mbox)-1] != '.' {
		mbox += origin
	}
	rrs = append(rrs, &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   domain,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Ns:      ns,
		Mbox:    mbox,
		Serial:  s.Serial,
		Refresh: uint32(s.Refresh.Seconds()),
		Retry:   uint32(s.Retry.Seconds()),
		Expire:  uint32(s.Expire.Seconds()),
		Minttl:  uint32(s.Negttl.Seconds()),
	})
	return
}

func origin_analyze(a *Analyzer) error {
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeSOA}) {
		if soa, ok := record.(*dns.SOA); ok {
			a.useRR(
				record,
				soa.Header().Name,
				&Origin{
					Ns:      soa.Ns,
					Mbox:    soa.Mbox,
					Serial:  soa.Serial,
					Refresh: time.Duration(soa.Refresh) * time.Second,
					Retry:   time.Duration(soa.Retry) * time.Second,
					Expire:  time.Duration(soa.Expire) * time.Second,
					Negttl:  time.Duration(soa.Minttl) * time.Second,
				},
			)
		}
	}
	return nil
}

func init() {
	RegisterService(
		func() happydns.Service {
			return &Origin{}
		},
		origin_analyze,
		ServiceInfos{
			Name:        "Origin",
			Description: "This is the root of your domain",
			Categories: []string{
				"internal",
			},
		},
		0,
	)
}
