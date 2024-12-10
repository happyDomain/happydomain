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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type GSuite struct {
	ValidationCode string `json:"validationCode,omitempty" happydomain:"label=Validation Code,placeholder=abcdef0123.mx-verification.google.com.,description=The verification code will be displayed during the initial domain setup and will not be usefull after Google validation."`
}

func (s *GSuite) GenKnownSvcs() []happydns.Service {
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

	return []happydns.Service{knownSvc, &svcs.SPF{
		Directives: []string{"include:_spf.google.com", "~all"},
	}}
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

func (s *GSuite) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	for _, svc := range s.GenKnownSvcs() {
		rrs = append(rrs, svc.GenRRs(domain, ttl, origin)...)
	}
	return
}

func gsuite_analyze(a *svcs.Analyzer) (err error) {
	var googlemx []string

	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX}) {
		if record.Type == "MX" {
			if strings.ToLower(record.GetTargetField()) == "aspmx.l.google.com." {
				googlemx = append(googlemx, record.NameFQDN)
				break
			}
		}
	}

	if len(googlemx) > 0 {
		for _, dn := range googlemx {
			googlerr := &GSuite{}

			for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeMX, Domain: dn}) {
				if record.Type == "MX" {
					if strings.HasSuffix(record.GetTargetField(), "mx-verification.google.com.") {
						googlerr.ValidationCode = record.GetTargetField()
						if err = a.UseRR(
							record,
							dn,
							googlerr,
						); err != nil {
							return
						}
					} else if strings.HasSuffix(record.GetTargetField(), "google.com.") {
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
				if record.Type == "TXT" {
					content := record.GetTargetTXTJoined()
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
