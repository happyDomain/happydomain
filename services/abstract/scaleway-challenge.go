// Copyright or © or Copr. happyDNS (2021)
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

package abstract

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/utils"
)

type ScalewayChallenge struct {
	Challenge string
}

func (s *ScalewayChallenge) GetNbResources() int {
	return 1
}

func (s *ScalewayChallenge) GenComment(origin string) string {
	return s.Challenge
}

func (s *ScalewayChallenge) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	rrs = append(rrs, &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   utils.DomainJoin("_scaleway-challenge", domain),
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Txt: []string{s.Challenge},
	})
	return
}

func scalewaychallenge_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Prefix: "_scaleway-challenge"}) {
		domain := strings.TrimPrefix(record.Header().Name, "_scaleway-challenge.")
		if txt, ok := record.(*dns.TXT); ok {
			a.UseRR(record, domain, &ScalewayChallenge{
				Challenge: strings.Join(txt.Txt, ""),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &ScalewayChallenge{}
		},
		scalewaychallenge_analyze,
		svcs.ServiceInfos{
			Name:        "Scaleway Challenge",
			Description: "Temporary record to prove that you are in possession of the domain.",
			Family:      svcs.Abstract,
			Categories: []string{
				"temporary",
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
			},
		},
		2,
	)
}
