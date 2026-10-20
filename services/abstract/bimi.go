// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/miekg/dns"

	svgps "git.happydns.org/svg-tiny-ps"

	"git.happydns.org/happyDomain/internal/helpers"
	svc "git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/model"
	svcs "git.happydns.org/happyDomain/services"
)

// The BIMI files are served under this path, on the bimi. host of the domain
// they belong to.
const (
	BIMIHostPrefix = "bimi."
	BIMIPathPrefix = "/bimi/"
)

// BIMIMaxLogoSize is the largest logo accepted: the BIMI draft recommends
// receivers ignore SVG files over 32 KiB. BIMIMaxCertificateSize leaves room
// for a VMC/CMC with its whole chain.
const (
	BIMIMaxLogoSize        = 32 * 1024
	BIMIMaxCertificateSize = 64 * 1024
)

// BIMI publishes a complete BIMI setup: the `<selector>._bimi.<domain>` TXT
// record *and* the files it points to, the SVG Tiny PS logo and the optional
// Verified Mark (or Common Mark) Certificate, served by happyDomain at
// https://bimi.<domain>/bimi/ behind the `bimi.<domain>` CNAME.
//
// The low-level svcs.BIMI service only carries the TXT record; it remains the
// right choice for someone who hosts the files themselves.
//
// Unlike the other hosted services, the TXT content is generated here from the
// files: their names are derived from their content, and only the backend can
// guarantee the record points at what it actually serves. The editor owns the
// record header (selector, TTL) and the CNAME.
type BIMI struct {
	Logo        []byte `json:"logo,omitempty" happydomain:"label=Logo,description=SVG Tiny Portable/Secure logo."`
	Certificate []byte `json:"vmc,omitempty" happydomain:"label=Mark Certificate,description=Verified Mark Certificate or Common Mark Certificate (PEM)."`

	Record    *happydns.TXT `json:"txt,omitempty"`
	HostCNAME *dns.CNAME    `json:"hostCNAME,omitempty"`
}

// IsHosted reports whether the bimi. CNAME is configured, without which nothing
// can reach the files.
func (s *BIMI) IsHosted() bool {
	return cnameConfigured(s.HostCNAME)
}

// selector cuts the relative name by hand: SplitBIMIName expects the trailing
// domain, which a relative record name does not have.
func (s *BIMI) selector() string {
	if !txtConfigured(s.Record) {
		return ""
	}
	selector, _, _ := strings.Cut(s.Record.Hdr.Name, "._bimi")
	return selector
}

func (s *BIMI) GetNbResources() int {
	n := 0
	if txtConfigured(s.Record) {
		n++
	}
	if cnameConfigured(s.HostCNAME) {
		n++
	}
	return n
}

func (s *BIMI) GenComment() string {
	var b strings.Builder

	b.WriteString(s.selector())
	if len(s.Logo) > 0 {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString("hosted logo")
		if len(s.Certificate) > 0 {
			b.WriteString(" + certificate")
		}
	}

	return b.String()
}

// contentName names a file after its content, so that a new file gets a new
// URL: receivers cache logos aggressively, and would otherwise keep showing
// the previous one.
func contentName(content []byte, ext string) string {
	if len(content) == 0 {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:16] + ext
}

// LogoName is empty when there is no logo.
func (s *BIMI) LogoName() string {
	return contentName(s.Logo, ".svg")
}

// CertificateName is empty when there is no certificate.
func (s *BIMI) CertificateName() string {
	return contentName(s.Certificate, ".pem")
}

// LogoFile returns the logo when it is hosted and served under the given name,
// nil otherwise. A logo that does not validate is never served.
func (s *BIMI) LogoFile(name string) []byte {
	if !s.IsHosted() || name == "" || name != s.LogoName() || ValidateLogo(s.Logo) != nil {
		return nil
	}
	return s.Logo
}

// CertificateFile returns the certificate when it is hosted and served under
// the given name, nil otherwise. A certificate that does not validate is never
// served.
func (s *BIMI) CertificateFile(name string) []byte {
	if !s.IsHosted() || name == "" || name != s.CertificateName() || ValidateCertificate(s.Certificate) != nil {
		return nil
	}
	return s.Certificate
}

func fileURL(fqdn, name string) string {
	return "https://" + BIMIHostPrefix + strings.TrimSuffix(fqdn, ".") + BIMIPathPrefix + name
}

// GetRecords generates the TXT record from the files.
//
// The stored TXT content is kept when it already says the same thing, so that
// an analyzed zone is republished byte-for-byte identical. It is also kept when
// there is no logo: after a zone import, the files cannot be recovered from
// DNS, and the record stands until the user uploads them again.
func (s *BIMI) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
	var rrs []happydns.Record

	if txtConfigured(s.Record) {
		record := *s.Record

		if len(s.Logo) > 0 {
			if !s.IsHosted() {
				return nil, fmt.Errorf("BIMI: the %s CNAME is required to serve the logo", BIMIHostPrefix)
			}
			if err := ValidateLogo(s.Logo); err != nil {
				return nil, fmt.Errorf("BIMI logo: %w", err)
			}

			fqdn := helpers.DomainFQDN(domain, origin)
			want := svcs.BIMIFields{Version: 1, Location: fileURL(fqdn, s.LogoName())}
			if len(s.Certificate) > 0 {
				if err := ValidateCertificate(s.Certificate); err != nil {
					return nil, fmt.Errorf("BIMI certificate: %w", err)
				}
				want.Authority = fileURL(fqdn, s.CertificateName())
			}

			var stored svcs.BIMIFields
			err := stored.Analyze(record.Txt)
			if err == nil {
				// The service has no field for the evidence: keep the stored one.
				want.Evidence = stored.Evidence
			}
			if err != nil || stored != want {
				record.Txt = want.String()
			}
		}

		if record.Txt != "" {
			rrs = append(rrs, &record)
		}
	}
	if cnameConfigured(s.HostCNAME) {
		cname := *s.HostCNAME
		rrs = append(rrs, &cname)
	}

	return rrs, nil
}

// ValidateLogo checks the logo against the SVG Tiny Portable/Secure profile the
// BIMI draft requires. The profile has no script, event handler, style,
// animation, foreign namespace or reference outside the document, which is
// what makes the file safe to serve as a plain image from a host of the user's
// domain. Violations the profile only discourages are not blocking.
func ValidateLogo(logo []byte) error {
	if len(logo) == 0 {
		return errors.New("the logo is empty")
	}
	if len(logo) > BIMIMaxLogoSize {
		return fmt.Errorf("the logo is %d bytes, over the %d bytes limit", len(logo), BIMIMaxLogoSize)
	}

	// The profile validator walks malformed markup leniently, on the
	// assumption that well-formedness is checked elsewhere: this is that check.
	decoder := xml.NewDecoder(bytes.NewReader(logo))
	for {
		if _, err := decoder.Token(); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("the logo is not valid XML: %w", err)
		}
	}

	problems, err := svgps.Validate(logo)
	if err != nil {
		return fmt.Errorf("the logo is not valid XML: %w", err)
	}
	for _, problem := range problems {
		if problem.Severity != svgps.SeverityError {
			continue
		}
		if problem.Line > 0 {
			return fmt.Errorf("the logo is not SVG Tiny PS, line %d: %s", problem.Line, problem.Text)
		}
		return fmt.Errorf("the logo is not SVG Tiny PS: %s", problem.Text)
	}

	return nil
}

// ValidateCertificate checks the content is a PEM chain made of certificates
// only, each one parsing as X.509. Whether it is a valid mark certificate for
// the domain is the receivers' call; this only ensures nothing else, such as a
// private key pasted by mistake, gets published.
func ValidateCertificate(content []byte) error {
	if len(content) > BIMIMaxCertificateSize {
		return fmt.Errorf("the certificate is %d bytes, over the %d bytes limit", len(content), BIMIMaxCertificateSize)
	}

	n := 0
	rest := content
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("the certificate file must only contain certificates, found %s", block.Type)
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return fmt.Errorf("invalid certificate: %w", err)
		}
		n++
	}

	if len(bytes.TrimSpace(rest)) > 0 {
		return errors.New("the certificate file contains data that is not PEM")
	}
	if n == 0 {
		return errors.New("no certificate found")
	}

	return nil
}

// bimi_analyze reconstructs a BIMI from a zone import. It only claims a TXT
// record whose logo is under https://bimi.<domain>/bimi/, together with the
// `bimi.<domain>` CNAME pointing at this instance: that pair is the
// unambiguous signal the files are hosted by happyDomain. Everything else is
// left to svcs.BIMI.
func bimi_analyze(a *svc.Analyzer) error {
	claimed := map[string]bool{}

	for _, record := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeTXT}) {
		_, domain, ok := svcs.SplitBIMIName(record.Header().Name)
		if !ok || claimed[domain] {
			continue
		}
		txt, ok := record.(*happydns.TXT)
		if !ok {
			continue
		}

		var fields svcs.BIMIFields
		if err := fields.Analyze(txt.Txt); err != nil {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(fields.Location), strings.ToLower(fileURL(domain, ""))) {
			continue
		}

		var hostCNAME *dns.CNAME
		for _, candidate := range a.SearchRR(svc.AnalyzerRecordFilter{Type: dns.TypeCNAME, Domain: BIMIHostPrefix + domain}) {
			if cname, ok := candidate.(*dns.CNAME); ok && hostingTargetMatches(cname.Target) {
				hostCNAME = cname
				break
			}
		}
		if hostCNAME == nil {
			continue
		}

		origin := a.GetOrigin()
		service := &BIMI{
			Record:    helpers.RRRelativeSubdomain(record, origin, domain).(*happydns.TXT),
			HostCNAME: helpers.RRRelativeSubdomain(hostCNAME, origin, domain).(*dns.CNAME),
		}

		if err := a.UseRR(record, domain, service); err != nil {
			return err
		}
		if err := a.UseRR(hostCNAME, domain, service); err != nil {
			return err
		}
		claimed[domain] = true
	}

	return nil
}

func init() {
	svc.RegisterService(
		func() happydns.ServiceBody { return &BIMI{} },
		bimi_analyze,
		happydns.ServiceInfos{
			Name:        "BIMI (hosted logo)",
			Family:      happydns.SERVICE_FAMILY_ABSTRACT,
			Categories:  []string{"email"},
			RecordTypes: []uint16{dns.TypeTXT, dns.TypeCNAME},
			Restrictions: happydns.ServiceRestrictions{
				NearAlone: true,
				Single:    true,
				NeedTypes: []uint16{dns.TypeTXT, dns.TypeCNAME},
			},
		},
		1,
	)
}
