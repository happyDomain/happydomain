// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package google // import "git.happydns.org/happyDomain/services/providers/google"

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type GSuite struct {
	ValidationCode string `json:"validationCode,omitempty" happydomain:"label=Validation Code,placeholder=abcdef0123.mx-verification.google.com.,description=The verification code will be displayed during the initial domain setup and will not be usefull after Google validation."`
}

func (s *GSuite) GenKnownSvcs() []happydns.ServiceBody {
	knownSvc := &svcs.MXs{
		MX: []svcs.MX{
			svcs.MX{Target: "aspmx.l.google.com.", Preference: 1},
			svcs.MX{Target: "alt1.aspmx.l.google.com.", Preference: 5},
			svcs.MX{Target: "alt2.aspmx.l.google.com.", Preference: 5},
			svcs.MX{Target: "alt3.aspmx.l.google.com.", Preference: 10},
			svcs.MX{Target: "alt4.aspmx.l.google.com.", Preference: 10},
		},
	}

	if len(s.ValidationCode) > 0 {
		knownSvc.MX = append(knownSvc.MX, svcs.MX{
			Target:     s.ValidationCode,
			Preference: 15,
		})
	}

	return []happydns.ServiceBody{knownSvc, &svcs.SPF{
		Version:    1,
		Directives: []string{"include:_spf.google.com", "~all"},
	}}
}

func (s *GSuite) GetNbResources() int {
	return 1
}

func (s *GSuite) GenComment() string {
	var comments []string
	for _, svc := range s.GenKnownSvcs() {
		comments = append(comments, svc.GenComment())
	}
	return strings.Join(comments, ", ")
}

func (s *GSuite) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	for _, svc := range s.GenKnownSvcs() {
		srrs, err := svc.GetRecords(domain, ttl, origin)
		if err != nil {
			return nil, err
		}
		rrs = append(rrs, srrs...)
	}
	return
}

func gsuite_analyze(a *svcs.Analyzer) (err error) {
	var googlemx []string

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX}) {
		if mx, ok := record.(*dns.MX); ok {
			if strings.ToLower(mx.Mx) == "aspmx.l.google.com." {
				googlemx = append(googlemx, record.Header().Name)
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
		func() happydns.ServiceBody {
			return &GSuite{}
		},
		gsuite_analyze,
		happydns.ServiceInfos{
			Name:        "G Suite",
			Description: "The suite of cloud computing, productivity and collaboration tools by Google.",
			Family:      happydns.SERVICE_FAMILY_PROVIDER,
			Categories: []string{
				"email",
			},
			Restrictions: happydns.ServiceRestrictions{
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
