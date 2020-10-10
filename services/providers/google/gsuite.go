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

package google // import "happydns.org/services/providers/google"

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
	"git.happydns.org/happydns/services"
	"git.happydns.org/happydns/services/abstract"
)

type GSuite struct {
	ValidationCode string `json:"validationCode,omitempty" happydns:"label=Validation Code,placeholder=abcdef0123.mx-verification.google.com.,description=The verification code will be displayed during the initial domain setup and will not be usefull after Google validation."`
}

func (s *GSuite) GenKnownSvcs() []happydns.Service {
	knownSvc := &abstract.EMail{
		MX: []svcs.MX{
			svcs.MX{Target: "ASPMX.L.GOOGLE.COM.", Preference: 1},
			svcs.MX{Target: "ALT1.ASPMX.L.GOOGLE.COM.", Preference: 5},
			svcs.MX{Target: "ALT2.ASPMX.L.GOOGLE.COM.", Preference: 5},
			svcs.MX{Target: "ALT3.ASPMX.L.GOOGLE.COM.", Preference: 10},
			svcs.MX{Target: "ALT4.ASPMX.L.GOOGLE.COM.", Preference: 10},
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

func gsuite_analyze(a *svcs.Analyzer) error {
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
