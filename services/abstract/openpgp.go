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

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
)

type OpenPGP struct {
	Username string          `json:"username,omitempty"`
	Record   *dns.OPENPGPKEY `json:"openpgpkey"`
}

func (s *OpenPGP) GetNbResources() int {
	return 1
}

func (s *OpenPGP) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *OpenPGP) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if s.Username != "" {
		if !strings.Contains(domain, "_openpgpkey") {
			domain = utils.DomainJoin("_openpgpkey", domain)
		}

		identifier := fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		if !strings.HasPrefix(domain, identifier) {
			domain = utils.DomainJoin(identifier, domain)
		}
	}

	rc, err := models.RRtoRC(s.Record, domain)
	if err != nil {
		return nil, err
	}
	rrs = append(rrs, &rc)
	return
}

type SMimeCert struct {
	Username string      `json:"username,omitempty"`
	Record   *dns.SMIMEA `json:"smimea"`
}

func (s *SMimeCert) GetNbResources() int {
	return 1
}

func (s *SMimeCert) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *SMimeCert) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records, e error) {
	if s.Username != "" {
		if !strings.Contains(domain, "_smimecert") {
			domain = utils.DomainJoin("_smimecert", domain)
		}

		identifier := fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		if !strings.HasPrefix(domain, identifier) {
			domain = utils.DomainJoin(identifier, domain)
		}
	}

	rc, err := models.RRtoRC(s.Record, domain)
	if err != nil {
		return nil, err
	}
	rrs = append(rrs, &rc)
	return
}

func openpgpkey_analyze(a *svcs.Analyzer) (err error) {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeOPENPGPKEY, Contains: "._openpgpkey."}) {
		if record.Type == "OPENPGPKEY" {
			domain := record.NameFQDN
			domain = domain[strings.Index(domain, "._openpgpkey")+13:]

			err = a.UseRR(
				record,
				domain,
				&OpenPGP{
					Record: record.ToRR().(*dns.OPENPGPKEY),
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
		if record.Type == "SMIMEA" {
			domain := record.NameFQDN
			domain = domain[strings.Index(domain, "._smimecert")+12:]

			err = a.UseRR(
				record,
				domain,
				&SMimeCert{
					Record: record.ToRR().(*dns.SMIMEA),
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
		func() happydns.Service {
			return &OpenPGP{}
		},
		openpgpkey_analyze,
		svcs.ServiceInfos{
			Name:        "PGP Key",
			Description: "Let users retrieve PGP key automatically.",
			Family:      svcs.Abstract,
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeOPENPGPKEY,
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeOPENPGPKEY,
				},
			},
		},
		1,
	)
	svcs.RegisterService(
		func() happydns.Service {
			return &SMimeCert{}
		},
		smimea_analyze,
		svcs.ServiceInfos{
			Name:        "SMimeCert",
			Description: "Publish S/MIME certificate.",
			Family:      svcs.Abstract,
			Categories: []string{
				"email",
			},
			RecordTypes: []uint16{
				dns.TypeSMIMEA,
			},
			Restrictions: svcs.ServiceRestrictions{
				NearAlone: true,
				NeedTypes: []uint16{
					dns.TypeSMIMEA,
				},
			},
		},
		1,
	)
}
