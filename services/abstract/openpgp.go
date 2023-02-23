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

package abstract

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/StackExchange/dnscontrol/v4/models"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/utils"
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

func (s *OpenPGP) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	if len(s.PublicKey) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rc := utils.NewRecordConfig(utils.DomainJoin(fmt.Sprintf("%s._openpgpkey", s.Identifier), domain), "OPENPGPKEY", ttl, origin)
		rc.SetTargetOpenPGPKey(base64.StdEncoding.EncodeToString(s.PublicKey))

		rrs = append(rrs, rc)
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

func (s *SMimeCert) GenRRs(domain string, ttl uint32, origin string) (rrs models.Records) {
	if len(s.Certificate) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rc := utils.NewRecordConfig(utils.DomainJoin(fmt.Sprintf("%s._smimecert", s.Identifier), domain), "SMIMEA", ttl, origin)
		rc.SetTarget(fmt.Sprintf("%d %d %d %s", s.CertUsage, s.Selector, s.MatchingType, hex.EncodeToString(s.Certificate)))

		rrs = append(rrs, rc)
	}
	return
}

func openpgpkey_analyze(a *svcs.Analyzer) (err error) {
	for _, record := range a.SearchRR(svcs.AnalyzerRecordFilter{Type: dns.TypeOPENPGPKEY, Contains: "._openpgpkey."}) {
		if record.Type == "OPENPGPKEY" {
			domain := record.NameFQDN
			domain = domain[strings.Index(domain, "._openpgpkey")+13:]

			identifier := strings.TrimSuffix(record.NameFQDN, "._openpgpkey."+domain)

			var pubkey []byte
			pubkey, err = base64.StdEncoding.DecodeString(strings.Join(strings.Fields(strings.TrimSuffix(strings.TrimPrefix(record.GetOpenPGPKeyField(), "("), ")")), ""))
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
		if record.Type == "SMIMEA" {
			domain := record.NameFQDN
			domain = domain[strings.Index(domain, "._smimecert")+12:]

			smimecert := record.ToRR().(*dns.SMIMEA)

			identifier := strings.TrimSuffix(record.NameFQDN, "._smimecert."+domain)

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
