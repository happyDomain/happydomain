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
	"encoding/json"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/service"
	"git.happydns.org/happyDomain/model"
)

type GSuite struct {
	MX           []*dns.MX `json:"mx,omitempty"`
	ValidationMX *dns.MX   `json:"validationMX,omitempty"`
}

func (s *GSuite) GetNbResources() int {
	nb := len(s.MX)
	if s.ValidationMX != nil {
		nb += 1
	}
	return nb
}

func (s *GSuite) GenComment() string {
	return "5 MX + SPF"
}

// GetSPFDirectives implements happydns.SPFContributor.
func (s *GSuite) GetSPFDirectives() []string {
	return []string{"include:_spf.google.com"}
}

// GetSPFAllPolicy implements happydns.SPFContributor.
func (s *GSuite) GetSPFAllPolicy() string {
	return ""
}

func (s *GSuite) Initialize() (any, error) {
	for i, mx := range []string{
		"aspmx.l.google.com.",
		"alt1.aspmx.l.google.com.",
		"alt2.aspmx.l.google.com.",
		"alt3.aspmx.l.google.com.",
		"alt4.aspmx.l.google.com.",
	} {
		rr := helpers.NewRecord("", "MX", 0, "")
		rr.(*dns.MX).Mx = mx
		if i == 0 {
			rr.(*dns.MX).Preference = 1
		} else if i < 3 {
			rr.(*dns.MX).Preference = 5
		} else {
			rr.(*dns.MX).Preference = 10
		}

		s.MX = append(s.MX, rr.(*dns.MX))
	}

	return s, nil
}

func (s *GSuite) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, e error) {
	for _, mx := range s.MX {
		rrs = append(rrs, mx)
	}

	if s.ValidationMX != nil {
		rrs = append(rrs, s.ValidationMX)
	}

	return
}

// UnmarshalJSON provides backward compatibility by silently ignoring the
// old "txt" SPF field that was previously stored in GSuite services.
func (s *GSuite) UnmarshalJSON(data []byte) error {
	type gsuiteAlias GSuite
	aux := &struct {
		*gsuiteAlias
		SPF json.RawMessage `json:"txt"`
	}{
		gsuiteAlias: (*gsuiteAlias)(s),
	}
	return json.Unmarshal(data, aux)
}

func gsuite_analyze(a *svc.Analyzer) (err error) {
	var googlemx []string

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeMX}) {
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

			for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeMX, Domain: dn}) {
				if mx, ok := record.(*dns.MX); ok {
					if strings.HasSuffix(mx.Mx, "mx-verification.google.com.") {
						googlerr.ValidationMX = mx
						if err = a.UseRR(
							record,
							dn,
							googlerr,
						); err != nil {
							return
						}
					} else if strings.HasSuffix(mx.Mx, "google.com.") {
						googlerr.MX = append(googlerr.MX, mx)
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

			if err = a.ClaimSPFDirective(dn, "include:_spf.google.com", googlerr); err != nil {
				return
			}
		}
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody {
			return &GSuite{}
		},
		gsuite_analyze,
		happydns.ServiceInfos{
			Name:        "G Suite",
			Description: "The suite of cloud computing, productivity and collaboration tools by Google.",
			Icon:        "/api/service_specs/google.GSuite/icon.png",
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
