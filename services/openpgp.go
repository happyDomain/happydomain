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

package svcs

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happydns/model"
)

type OpenPGP struct {
	Username   string `json:"username,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	PublicKey  []byte `json:"pubkey"`
}

func (s *OpenPGP) GetNbResources() int {
	return 1
}

func (s *OpenPGP) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *OpenPGP) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	if len(s.PublicKey) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rrs = append(rrs, &dns.OPENPGPKEY{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("_%s._openpgpkey.%s", s.Identifier, domain),
				Rrtype: dns.TypeOPENPGPKEY,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			PublicKey: base64.StdEncoding.EncodeToString(s.PublicKey),
		})
	}
	return
}

type SMimeCert struct {
	Username     string `json:"username,omitempty"`
	Identifier   string `json:"identifier,omitempty"`
	CertUsage    uint8  `json:"certusage"`
	Selector     uint8  `json:"selector"`
	MatchingType uint8  `json:"matchingtype"`
	Certificate  []byte `json:"certificate"`
}

func (s *SMimeCert) GetNbResources() int {
	return 1
}

func (s *SMimeCert) GenComment(origin string) string {
	return fmt.Sprintf("%s", s.Username)
}

func (s *SMimeCert) GenRRs(domain string, ttl uint32, origin string) (rrs []dns.RR) {
	if len(s.Certificate) > 0 {
		if s.Username != "" {
			s.Identifier = fmt.Sprintf("%x", sha256.Sum224([]byte(s.Username)))
		}

		rrs = append(rrs, &dns.SMIMEA{
			Hdr: dns.RR_Header{
				Name:   fmt.Sprintf("_%s._smimecert.%s", s.Identifier, domain),
				Rrtype: dns.TypeSMIMEA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Usage:        s.CertUsage,
			Selector:     s.Selector,
			MatchingType: s.MatchingType,
			Certificate:  hex.EncodeToString(s.Certificate),
		})
	}
	return
}

func openpgpkey_analyze(a *Analyzer) (err error) {
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeOPENPGPKEY, Contains: "._openpgpkey."}) {
		if openpgpkey, ok := record.(*dns.OPENPGPKEY); ok {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._openpgpkey")+13:]

			identifier := strings.TrimSuffix(record.Header().Name, "._openpgpkey."+domain)

			var pubkey []byte
			pubkey, err = base64.StdEncoding.DecodeString(openpgpkey.PublicKey)
			if err != nil {
				return
			}

			err = a.useRR(
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

func smimea_analyze(a *Analyzer) (err error) {
	for _, record := range a.searchRR(AnalyzerRecordFilter{Type: dns.TypeSMIMEA, Contains: "._smimecert."}) {
		if smimecert, ok := record.(*dns.SMIMEA); ok {
			domain := record.Header().Name
			domain = domain[strings.Index(domain, "._smimecert")+12:]

			identifier := strings.TrimSuffix(record.Header().Name, "._smimecert."+domain)

			var cert []byte
			cert, err = hex.DecodeString(smimecert.Certificate)
			if err != nil {
				return
			}

			err = a.useRR(
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
	RegisterService(
		func() happydns.Service {
			return &OpenPGP{}
		},
		openpgpkey_analyze,
		ServiceInfos{
			Name:        "PGP Key",
			Description: "Let users retrieve PGP key automatically.",
			Categories: []string{
				"email",
			},
		},
		1,
	)
	RegisterService(
		func() happydns.Service {
			return &SMimeCert{}
		},
		smimea_analyze,
		ServiceInfos{
			Name:        "SMimeCert",
			Description: "Publish S/MIME certificate.",
			Categories: []string{
				"email",
			},
		},
		1,
	)
}
