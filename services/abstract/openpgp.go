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

package abstract

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type OpenPGP struct {
	Username string          `json:"username,omitempty"`
	Record   *dns.OPENPGPKEY `json:"openpgpkey"`
}

func (s *OpenPGP) GetNbResources() int {
	return 1
}

func (s *OpenPGP) GenComment() string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *OpenPGP) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	if s.Username != "" {
		identifier := fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		if !strings.HasPrefix(domain, identifier) {
			return nil, fmt.Errorf("Invalid prefix")
		}
	}

	return []happydns.Record{s.Record}, nil
}

type SMimeCert struct {
	Username string      `json:"username,omitempty"`
	Record   *dns.SMIMEA `json:"smimea"`
}

func (s *SMimeCert) GetNbResources() int {
	return 1
}

func (s *SMimeCert) GenComment() string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *SMimeCert) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	if s.Username != "" {
		identifier := fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		if !strings.HasPrefix(domain, identifier) {
			return nil, fmt.Errorf("Invalid prefix")
		}
	}

	return []happydns.Record{s.Record}, nil
}

func openpgpkey_analyze(a *svcs.Analyzer) (err error) {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeOPENPGPKEY, Contains: "._openpgpkey."}) {
		if record.Header().Rrtype == dns.TypeOPENPGPKEY {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._openpgpkey")+13:]

			err = a.UseRR(
				record,
				domain,
				&OpenPGP{
					Record: helpers.RRRelative(record, domain).(*dns.OPENPGPKEY),
				},
			)
			if err != nil {
				return
			}
		}
	}
	return
}

func smimea_analyze(a *svcs.Analyzer) (err error) {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeSMIMEA, Contains: "._smimecert."}) {
		if record.Header().Rrtype == dns.TypeSMIMEA {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._smimecert")+12:]

			err = a.UseRR(
				record,
				domain,
				&SMimeCert{
					Record: helpers.RRRelative(record, domain).(*dns.SMIMEA),
				},
			)
			if err != nil {
				return
			}
		}
	}

	return nil
}

func init() {
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &OpenPGP{}
		},
		openpgpkey_analyze,
		happydns.ServiceInfos{
			Name:        "PGP Key",
			Description: "Let users retrieve PGP key automatically.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeOPENPGPKEY,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeOPENPGPKEY,
				},
			},
		},
		1,
	)
	svcs.RegisterService(
		func() happydns.ServiceBody {
			return &SMimeCert{}
		},
		smimea_analyze,
		happydns.ServiceInfos{
			Name:        "SMimeCert",
			Description: "Publish S/MIME certificate.",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeSMIMEA,
			},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSMIMEA,
				},
			},
		},
		1,
	)
}
