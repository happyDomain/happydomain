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
)

type Origin struct {
	Ns      string
	Mbox    string
	Serial  uint32
	Refresh uint32
	Retry   uint32
	Expire  uint32
	Minttl  uint32
}

func (s *Origin) GetNbResources() int {
	return 1
}

func (s *Origin) GenComment(origin string) string {
	return fmt.Sprintf("%s %s %d", strings.TrimSuffix(s.Ns, "."+origin), strings.TrimSuffix(s.Mbox, "."+origin), s.Serial)
}

func (s *Origin) GenRRs(domain string, ttl uint32) (rrs []dns.RR) {
	rrs = append(rrs, &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   domain,
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Ns:      s.Ns,
		Mbox:    s.Mbox,
		Serial:  s.Serial,
		Refresh: s.Refresh,
		Retry:   s.Retry,
		Expire:  s.Expire,
		Minttl:  s.Minttl,
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
					Refresh: soa.Refresh,
					Retry:   soa.Retry,
					Expire:  soa.Expire,
					Minttl:  soa.Minttl,
				},
			)
		}
	}
	return nil
}

func init() {
	RegisterService(
		"git.happydns.org/happydns/services/Origin",
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
		100,
	)
}
