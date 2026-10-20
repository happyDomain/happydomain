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

package abstract_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"

	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
	"git.happydns.org/happyDomain/services/abstract"
)

const testLogo = `<?xml version="1.0" encoding="UTF-8"?>
<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">
  <title>Example</title>
  <defs><circle id="c" cx="50" cy="50" r="40"/></defs>
  <use xlink:href="#c" fill="#123456"/>
</svg>`

// testCertificate returns a freshly generated self-signed certificate, PEM
// encoded. Its content does not matter: the service only checks it parses.
func testCertificate(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func contentName(content []byte, ext string) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:16] + ext
}

func hostedBIMI(logo, cert []byte) *abstract.BIMI {
	return &abstract.BIMI{
		Logo:        logo,
		Certificate: cert,
		Record:      mkTXT("default._bimi", ""),
		HostCNAME:   mkCNAME("bimi", "happydomain.example.com."),
	}
}

func TestBIMI_ValidateLogo(t *testing.T) {
	if err := abstract.ValidateLogo([]byte(testLogo)); err != nil {
		t.Errorf("ValidateLogo(valid logo) = %v; want nil", err)
	}

	for name, logo := range map[string]string{
		"empty":            ``,
		"not xml":          `this is not a logo`,
		"truncated":        `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title>`,
		"not svg":          `<html baseProfile="tiny-ps"><title>x</title></html>`,
		"no profile":       `<svg version="1.2" xmlns="http://www.w3.org/2000/svg"><title>x</title></svg>`,
		"wrong profile":    `<svg version="1.2" baseProfile="tiny" xmlns="http://www.w3.org/2000/svg"><title>x</title></svg>`,
		"script":           `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><script>alert(1)</script></svg>`,
		"script uppercase": `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><SCRIPT>alert(1)</SCRIPT></svg>`,
		"foreignObject":    `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><foreignObject><p>x</p></foreignObject></svg>`,
		"event handler":    `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><title>x</title></svg>`,
		"nested handler":   `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><rect OnClick="alert(1)"/></svg>`,
		"external href":    `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><title>x</title><image xlink:href="https://evil.example/x.png"/></svg>`,
		"javascript href":  `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><a href="javascript:alert(1)"><rect/></a></svg>`,
		"entity":           `<!DOCTYPE svg [<!ENTITY x "y">]><svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>&x;</title></svg>`,
		"oversize":         `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><desc>` + strings.Repeat("a", abstract.BIMIMaxLogoSize) + `</desc></svg>`,
		// Accepted by a filter on active content alone, refused by the profile.
		"stylesheet pi":   `<?xml-stylesheet href="https://evil.example/x.css"?><svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title></svg>`,
		"style element":   `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><style>rect{fill:red}</style></svg>`,
		"image element":   `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><image href="#a"/></svg>`,
		"animation":       `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><rect><animate attributeName="x" to="1"/></rect></svg>`,
		"foreign element": `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>x</title><x:p xmlns:x="http://www.w3.org/1999/xhtml">x</x:p></svg>`,
		"no title":        `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><rect/></svg>`,
	} {
		if err := abstract.ValidateLogo([]byte(logo)); err == nil {
			t.Errorf("ValidateLogo(%s) = nil; want an error", name)
		}
	}
}

func TestBIMI_ValidateCertificate(t *testing.T) {
	cert := testCertificate(t)

	if err := abstract.ValidateCertificate(cert); err != nil {
		t.Errorf("ValidateCertificate(one certificate) = %v; want nil", err)
	}
	chain := append(append([]byte{}, cert...), testCertificate(t)...)
	if err := abstract.ValidateCertificate(chain); err != nil {
		t.Errorf("ValidateCertificate(chain) = %v; want nil", err)
	}

	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("secret")})
	for name, content := range map[string][]byte{
		"empty":          nil,
		"garbage":        []byte("not a certificate"),
		"private key":    key,
		"cert then key":  append(append([]byte{}, cert...), key...),
		"bad der":        pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("junk")}),
		"trailing bytes": append(append([]byte{}, cert...), []byte("<script>")...),
		"oversize":       append(append([]byte{}, cert...), []byte(strings.Repeat("\n", abstract.BIMIMaxCertificateSize))...),
	} {
		if err := abstract.ValidateCertificate(content); err == nil {
			t.Errorf("ValidateCertificate(%s) = nil; want an error", name)
		}
	}
}

func TestBIMI_FileNames(t *testing.T) {
	cert := testCertificate(t)
	s := hostedBIMI([]byte(testLogo), cert)

	if got, want := s.LogoName(), contentName([]byte(testLogo), ".svg"); got != want {
		t.Errorf("LogoName() = %q; want %q", got, want)
	}
	if got, want := s.CertificateName(), contentName(cert, ".pem"); got != want {
		t.Errorf("CertificateName() = %q; want %q", got, want)
	}

	// The name is the first 64 bits of the SHA-256 of the content, pinned
	// against a known digest so that a change of scheme is noticed.
	abc := &abstract.BIMI{Logo: []byte("abc"), Certificate: []byte("abc")}
	if got := abc.LogoName(); got != "ba7816bf8f01cfea.svg" {
		t.Errorf("LogoName() = %q; want ba7816bf8f01cfea.svg", got)
	}
	if got := abc.CertificateName(); got != "ba7816bf8f01cfea.pem" {
		t.Errorf("CertificateName() = %q; want ba7816bf8f01cfea.pem", got)
	}

	empty := &abstract.BIMI{}
	if empty.LogoName() != "" || empty.CertificateName() != "" {
		t.Errorf("names of an empty service = (%q, %q); want empty", empty.LogoName(), empty.CertificateName())
	}
}

func TestBIMI_LogoFile(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)

	if got := s.LogoFile(s.LogoName()); string(got) != testLogo {
		t.Errorf("LogoFile(current name) = %q; want the logo", got)
	}
	for _, name := range []string{"", "0000000000000000.svg", strings.TrimSuffix(s.LogoName(), ".svg") + ".pem"} {
		if got := s.LogoFile(name); got != nil {
			t.Errorf("LogoFile(%q) = %q; want nil", name, got)
		}
	}

	notHosted := hostedBIMI([]byte(testLogo), nil)
	notHosted.HostCNAME = nil
	if got := notHosted.LogoFile(notHosted.LogoName()); got != nil {
		t.Errorf("LogoFile() without the bimi. CNAME = %q; want nil", got)
	}

	// Stored content that does not validate is never served, whatever path
	// it reached storage by.
	bad := hostedBIMI([]byte(`<svg baseProfile="tiny-ps"><script>alert(1)</script></svg>`), nil)
	if got := bad.LogoFile(bad.LogoName()); got != nil {
		t.Errorf("LogoFile() of an invalid logo = %q; want nil", got)
	}
}

func TestBIMI_CertificateFile(t *testing.T) {
	cert := testCertificate(t)
	s := hostedBIMI([]byte(testLogo), cert)

	if got := s.CertificateFile(s.CertificateName()); string(got) != string(cert) {
		t.Errorf("CertificateFile(current name) = %q; want the certificate", got)
	}
	if got := s.CertificateFile("0000000000000000.pem"); got != nil {
		t.Errorf("CertificateFile(stale name) = %q; want nil", got)
	}
	if got := hostedBIMI([]byte(testLogo), nil).CertificateFile(""); got != nil {
		t.Errorf("CertificateFile() without certificate = %q; want nil", got)
	}
}

func TestBIMI_GetRecords(t *testing.T) {
	cert := testCertificate(t)
	s := hostedBIMI([]byte(testLogo), cert)

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if len(rrs) != 2 {
		t.Fatalf("GetRecords() returned %d records; want 2", len(rrs))
	}

	txt, ok := rrs[0].(*happydns.TXT)
	if !ok {
		t.Fatalf("first record is a %T; want *happydns.TXT", rrs[0])
	}
	want := "v=BIMI1;l=https://bimi.example.com/bimi/" + s.LogoName() + ";a=https://bimi.example.com/bimi/" + s.CertificateName()
	if txt.Txt != want {
		t.Errorf("TXT = %q; want %q", txt.Txt, want)
	}
	if txt.Hdr.Name != "default._bimi" {
		t.Errorf("TXT name = %q; want %q", txt.Hdr.Name, "default._bimi")
	}
	if rrs[1].Header().Name != "bimi" {
		t.Errorf("second record name = %q; want %q", rrs[1].Header().Name, "bimi")
	}

	// Generating the record must not touch the stored one.
	if s.Record.Txt != "" {
		t.Errorf("stored TXT was rewritten to %q", s.Record.Txt)
	}
}

func TestBIMI_GetRecords_Subdomain(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)

	rrs, err := s.GetRecords("sub", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	want := "v=BIMI1;l=https://bimi.sub.example.com/bimi/" + s.LogoName()
	if got := rrs[0].(*happydns.TXT).Txt; got != want {
		t.Errorf("TXT = %q; want %q", got, want)
	}
}

// The returned CNAME is a copy: the zone code adjusts the records it gets,
// which must not alter the stored service.
func TestBIMI_GetRecords_CopiesCNAME(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	rrs[1].(*dns.CNAME).Target = "changed.example.net."
	if s.HostCNAME.Target != "happydomain.example.com." {
		t.Errorf("stored CNAME target = %q; the stored service was modified", s.HostCNAME.Target)
	}
}

// A record equivalent to the generated one keeps its formatting, so that an
// analyzed zone is republished byte-for-byte identical.
func TestBIMI_GetRecords_KeepsEquivalentRecord(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)
	stored := "v=BIMI1; l=https://bimi.example.com/bimi/" + s.LogoName()
	s.Record.Txt = stored

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if got := rrs[0].(*happydns.TXT).Txt; got != stored {
		t.Errorf("TXT = %q; want the stored %q", got, stored)
	}
}

// After an import, the files cannot be recovered from DNS: the record is
// published verbatim until the user uploads the logo again.
func TestBIMI_GetRecords_NoLogoKeepsRecord(t *testing.T) {
	s := hostedBIMI(nil, nil)
	s.Record.Txt = "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg"

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if got := rrs[0].(*happydns.TXT).Txt; got != s.Record.Txt {
		t.Errorf("TXT = %q; want %q", got, s.Record.Txt)
	}
}

func TestBIMI_GetRecords_Errors(t *testing.T) {
	for name, s := range map[string]*abstract.BIMI{
		"invalid logo":        hostedBIMI([]byte(`<svg><script/></svg>`), nil),
		"invalid certificate": hostedBIMI([]byte(testLogo), []byte("junk")),
		"no cname": {
			Logo:   []byte(testLogo),
			Record: mkTXT("default._bimi", ""),
		},
	} {
		if _, err := s.GetRecords("", 3600, "example.com."); err == nil {
			t.Errorf("GetRecords(%s) = nil error; want an error", name)
		}
	}
}

// A service whose editor was never filled must not publish the stubs the
// service-spec auto-initializer leaves behind.
func TestBIMI_GetRecords_Stubs(t *testing.T) {
	s := &abstract.BIMI{Record: &happydns.TXT{}, HostCNAME: &dns.CNAME{}}

	if n := s.GetNbResources(); n != 0 {
		t.Errorf("GetNbResources() = %d; want 0", n)
	}
	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if len(rrs) != 0 {
		t.Errorf("GetRecords() returned %d records; want 0", len(rrs))
	}
}

func TestBIMI_IsHosted(t *testing.T) {
	for name, tc := range map[string]struct {
		service *abstract.BIMI
		want    bool
	}{
		"empty":      {&abstract.BIMI{}, false},
		"stub cname": {&abstract.BIMI{HostCNAME: &dns.CNAME{}}, false},
		"hosted":     {&abstract.BIMI{HostCNAME: mkCNAME("bimi", "happydomain.example.com.")}, true},
	} {
		if got := tc.service.IsHosted(); got != tc.want {
			t.Errorf("%s: IsHosted() = %v; want %v", name, got, tc.want)
		}
	}
}

func TestBIMI_GenComment(t *testing.T) {
	cert := testCertificate(t)

	for name, tc := range map[string]struct {
		service *abstract.BIMI
		want    string
	}{
		"empty":   {&abstract.BIMI{}, ""},
		"no logo": {hostedBIMI(nil, nil), "default"},
		"logo":    {hostedBIMI([]byte(testLogo), nil), "default: hosted logo"},
		"vmc":     {hostedBIMI([]byte(testLogo), cert), "default: hosted logo + certificate"},
	} {
		if got := tc.service.GenComment(); got != tc.want {
			t.Errorf("%s: GenComment() = %q; want %q", name, got, tc.want)
		}
	}
}

func countBIMI(services []*happydns.Service) (hosted, plain int) {
	for _, s := range services {
		switch s.Service.(type) {
		case *abstract.BIMI:
			hosted++
		case *svcs.BIMI:
			plain++
		}
	}
	return
}

func TestBIMI_Analyze(t *testing.T) {
	services := analyzeMTASTS(t,
		mustRR(t, `default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg"`),
		mustRR(t, `bimi.example.com. 3600 IN CNAME happydomain.example.com.`),
	)

	if hosted, plain := countBIMI(services[""]); hosted != 1 || plain != 0 {
		t.Fatalf("found %d abstract.BIMI and %d svcs.BIMI; want 1 and 0", hosted, plain)
	}

	s := services[""][0].Service.(*abstract.BIMI)
	if s.Record == nil || s.Record.Hdr.Name != "default._bimi" {
		t.Errorf("TXT record = %v; want default._bimi", s.Record)
	}
	if s.HostCNAME == nil || s.HostCNAME.Hdr.Name != "bimi" {
		t.Errorf("CNAME record = %v; want bimi", s.HostCNAME)
	}
}

func TestBIMI_Analyze_StaysLowLevel(t *testing.T) {
	for name, records := range map[string][]string{
		"txt only": {
			`default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg"`,
		},
		"logo elsewhere": {
			`default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://cdn.example.net/logo.svg"`,
			`bimi.example.com. 3600 IN CNAME happydomain.example.com.`,
		},
		"third-party cname": {
			`default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg"`,
			`bimi.example.com. 3600 IN CNAME third-party.example.net.`,
		},
	} {
		var rrs []happydns.Record
		for _, r := range records {
			rrs = append(rrs, mustRR(t, r))
		}
		services := analyzeMTASTS(t, rrs...)

		if hosted, plain := countBIMI(services[""]); hosted != 0 || plain != 1 {
			t.Errorf("%s: found %d abstract.BIMI and %d svcs.BIMI; want 0 and 1", name, hosted, plain)
		}
	}
}

// The bimi. CNAME can only back one hosted service: a second selector pointing
// at it stays with svcs.BIMI.
func TestBIMI_Analyze_TwoSelectors(t *testing.T) {
	services := analyzeMTASTS(t,
		mustRR(t, `default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg"`),
		mustRR(t, `brand._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://bimi.example.com/bimi/fedcba9876543210.svg"`),
		mustRR(t, `bimi.example.com. 3600 IN CNAME happydomain.example.com.`),
	)

	if hosted, plain := countBIMI(services[""]); hosted != 1 || plain != 1 {
		t.Errorf("found %d abstract.BIMI and %d svcs.BIMI; want 1 and 1", hosted, plain)
	}
}

// What GetRecords generates is recognized again by the analyzer, subdomains
// included.
func TestBIMI_Analyze_Roundtrip(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)
	s.Record = mkTXT("default._bimi.sub", "")
	s.HostCNAME = mkCNAME("bimi.sub", "happydomain.example.com.")

	rrs, err := s.GetRecords("sub", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	for _, rr := range rrs {
		rr.Header().Name += ".example.com."
	}
	services := analyzeMTASTS(t, rrs...)

	if hosted, plain := countBIMI(services["sub"]); hosted != 1 || plain != 0 {
		t.Errorf("found %d abstract.BIMI and %d svcs.BIMI; want 1 and 0", hosted, plain)
	}
}

// Without a logo and without any TXT content there is nothing to say yet: an
// empty TXT must not reach the zone.
func TestBIMI_GetRecords_NoLogoEmptyTXT(t *testing.T) {
	s := hostedBIMI(nil, nil)

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	for _, rr := range rrs {
		if txt, ok := rr.(*happydns.TXT); ok {
			t.Errorf("GetRecords() returned the TXT %q; want none", txt.Txt)
		}
	}
}

// The service has no field for the evidence tag: regenerating the record for a
// new logo must not drop the one an analyzed zone carried.
func TestBIMI_GetRecords_KeepsEvidence(t *testing.T) {
	s := hostedBIMI([]byte(testLogo), nil)
	s.Record.Txt = "v=BIMI1; l=https://bimi.example.com/bimi/0123456789abcdef.svg; e=https://bimi.example.com/evidence.pdf"

	rrs, err := s.GetRecords("", 3600, "example.com.")
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	got := rrs[0].(*happydns.TXT).Txt
	if !strings.Contains(got, s.LogoName()) {
		t.Errorf("TXT = %q; want it to point at %s", got, s.LogoName())
	}
	if !strings.Contains(got, "e=https://bimi.example.com/evidence.pdf") {
		t.Errorf("TXT = %q; want the evidence tag kept", got)
	}
}
