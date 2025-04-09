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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

type OpenPGP struct {
	Username   string              `json:"username,omitempty"`
	Identifier string              `json:"identifier,omitempty"`
	PublicKey  happydns.HexaString `json:"pubkey"`
}

func (s *OpenPGP) GetNbResources() int {
	return 1
}

func (s *OpenPGP) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *OpenPGP) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, err error) {
	if len(s.PublicKey) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rr := utils.NewRecord(utils.DomainJoin(fmt.Sprintf("%s._openpgpkey", s.Identifier), domain), "OPENPGPKEY", ttl, origin)
		rr.(*dns.OPENPGPKEY).PublicKey = base64.StdEncoding.EncodeToString(s.PublicKey)

		rrs = append(rrs, rr)
	}

	return
}

type SMimeCert struct {
	Username     string              `json:"username,omitempty"`
	Identifier   string              `json:"identifier,omitempty"`
	CertUsage    uint8               `json:"certusage"`
	Selector     uint8               `json:"selector"`
	MatchingType uint8               `json:"matchingtype"`
	Certificate  happydns.HexaString `json:"certificate"`
}

func (s *SMimeCert) GetNbResources() int {
	return 1
}

func (s *SMimeCert) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *SMimeCert) GetRecords(domain string, ttl uint32, origin string) (rrs []happydns.Record, err error) {
	if len(s.Certificate) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rr := utils.NewRecord(utils.DomainJoin(fmt.Sprintf("%s._smimecert", s.Identifier), domain), "SMIMEA", ttl, origin)
		rr.(*dns.SMIMEA).Usage = s.CertUsage
		rr.(*dns.SMIMEA).Selector = s.Selector
		rr.(*dns.SMIMEA).MatchingType = s.MatchingType
		rr.(*dns.SMIMEA).Certificate = hex.EncodeToString(s.Certificate)

		rrs = append(rrs, rr)
	}

	return
}

func openpgpkey_analyze(a *svcs.Analyzer) (err error) {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeOPENPGPKEY, Contains: "._openpgpkey."}) {
		if openpgpkey, ok := record.(*dns.OPENPGPKEY); ok {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._openpgpkey")+13:]

			identifier := strings.TrimSuffix(record.Header().Name, "._openpgpkey."+domain)

			var pubkey []byte
			pubkey, err = base64.StdEncoding.DecodeString(strings.Join(strings.Fields(strings.TrimSuffix(strings.TrimPrefix(openpgpkey.PublicKey, "("), ")")), ""))
			if err != nil {
				return
			}

			err = a.UseRR(
				record,
				domain,
				&OpenPGP{
					Identifier: identifier,
					PublicKey:  pubkey,
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
		if smimecert, ok := record.(*dns.SMIMEA); ok {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._smimecert")+12:]

			identifier := strings.TrimSuffix(record.Header().Name, "._smimecert."+domain)

			var cert []byte
			cert, err = hex.DecodeString(smimecert.Certificate)
			if err != nil {
				return
			}

			err = a.UseRR(
				record,
				domain,
				&SMimeCert{
					Identifier:   identifier,
					CertUsage:    smimecert.Usage,
					Selector:     smimecert.Selector,
					MatchingType: smimecert.MatchingType,
					Certificate:  cert,
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
