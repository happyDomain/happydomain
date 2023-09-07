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

package google // import "git.happydns.org/happyDomain/services/providers/google"

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

type GSuite struct {
	ValidationCode string `json:"validationCode,omitempty" happydomain:"label=Validation Code,placeholder=abcdef0123.mx-verification.google.com.,description=The verification code will be displayed during the initial domain setup and will not be usefull after Google validation."`
}

func (s *GSuite) GenKnownSvcs() []happydns.Service {
	knownSvc := &abstract.EMail{
		MX: []svcs.MX{
			svcs.MX{Target: "aspmx.l.google.com.", Preference: 1},
			svcs.MX{Target: "alt1.aspmx.l.google.com.", Preference: 5},
			svcs.MX{Target: "alt2.aspmx.l.google.com.", Preference: 5},
			svcs.MX{Target: "alt3.aspmx.l.google.com.", Preference: 10},
			svcs.MX{Target: "alt4.aspmx.l.google.com.", Preference: 10},
		},
		SPF: &svcs.SPF{
			Content: "include:_spf.google.com ~all",
		},
	}

	if len(s.ValidationCode) > 0 {
		knownSvc.MX = append(knownSvc.MX, svcs.MX{
			Target:     s.ValidationCode,
			Preference: 15,
		})
	}

	return []happydns.Service{knownSvc}
}

func (s *GSuite) GetNbResources() int {
	return 1
}

func (s *GSuite) GenComment(origin string) string {
	var comments []string
	for _, svc := range s.GenKnownSvcs() {
		comments = append(comments, svc.GenComment(origin))
	}
	return strings.Join(comments, ", ")
}

func (s *GSuite) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	for _, svc := range s.GenKnownSvcs() {
		rrs = append(rrs, svc.GenRRs(domain, ttl, origin)...)
	}
	return
}

func gsuite_analyze(a *svcs.Analyzer) (err error) {
	var googlemx []string

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX}) {
		if mx, ok := record.(*dns.MX); ok {
			if strings.ToLower(mx.Mx) == "aspmx.l.google.com." {
				googlemx = append(googlemx, mx.Header().Name)
				break
			}
		}
	}

	if len(googlemx) > 0 {
		for _, dn := range googlemx {
			googlerr := &GSuite{}

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX, Domain: dn}) {
				if mx, ok := record.(*dns.MX); ok {
					if strings.HasSuffix(mx.Mx, "mx-verification.google.com.") {
						googlerr.ValidationCode = mx.Mx
						if err = a.UseRR(
							record,
							dn,
							googlerr,
						); err != nil {
							return
						}
					} else if strings.HasSuffix(mx.Mx, "google.com.") {
						if err = a.UseRR(
							record,
							dn,
							googlerr,
						); err != nil {
							return
						}
					}
				}
			}

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeTXT, Domain: dn}) {
				if txt, ok := record.(*dns.TXT); ok {
					content := strings.Join(txt.Txt, "")
					if strings.HasPrefix(content, "v=spf1") && strings.Contains(content, "_spf.google.com") {
						if err = a.UseRR(
							record,
							dn,
							googlerr,
						); err != nil {
							return
						}
					}
				}
			}
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.Service {
			return &GSuite{}
		},
		gsuite_analyze,
		svcs.ServiceInfos{
			Name:        "G Suite",
			Description: "The suite of cloud computing, productivity and collaboration tools by Google.",
			Family:      svcs.Provider,
			Categories: []string{
				"cloud",
				"email",
			},
			Restrictions: svcs.ServiceRestrictions{
				ExclusiveRR: []string{
					"abstract.EMail",
					"svcs.MX",
				},
				Single: true,
				NeedTypes: []uint16{
					dns.TypeMX,
				},
			},
		},
		0,
	)
}
