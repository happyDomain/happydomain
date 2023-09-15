// Copyright or © or Copr. happyDNS (2021)
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
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type GoogleVerif struct {
	SiteVerification string `happydomain:"label=Site Verification"`
}

func (s *GoogleVerif) GetNbResources() int {
	return 1
}

func (s *GoogleVerif) GenComment(origin string) string {
	return s.SiteVerification
}

func (s *GoogleVerif) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	rc := utils.NewRecordConfig(domain, "TXT", ttl, origin)
	rc.SetTargetTXT("google-site-verification=" + strings.TrimPrefix(s.SiteVerification, "google-site-verification="))

	rrs = append(rrs, rc)
	return
}

func googleverification_analyze(a *svcs.Analyzer) error {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		domain := record.NameFQDN
		if record.Type == "TXT" && strings.HasPrefix(record.GetTargetTXTJoined(), "google-site-verification=") {
			a.UseRR(record, domain, &GoogleVerif{
				SiteVerification: strings.TrimPrefix(record.GetTargetTXTJoined(), "google-site-verification="),
			})
		}
	}
	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &GoogleVerif{}
		},
		googleverification_analyze,
		svcs.ServiceInfos{
			Name:        "Google Verification",
			Description: "Temporary record to prove that you control the domain.",
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
