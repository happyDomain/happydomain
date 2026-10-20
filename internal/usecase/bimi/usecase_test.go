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

package bimi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/miekg/dns"

	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"

	"git.happydns.org/happyDomain/internal/usecase/hostedservice/hostedservicetest"
)

const logo = `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>Example</title></svg>`

func certificate(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "example.com"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func hostCNAME() *dns.CNAME {
	return &dns.CNAME{
		Hdr:    dns.RR_Header{Name: "bimi", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
		Target: "happydomain.example.com.",
	}
}

// newUsecase wires an Usecase over a single domain owning a single zone that
// holds the given service.
func newUsecase(t *testing.T, service happydns.ServiceBody) *Usecase {
	t.Helper()

	domains, zones := hostedservicetest.NewStore(t, "example.com.", service)

	return NewUsecase(domains, zones)
}

func TestFiles(t *testing.T) {
	cert := certificate(t)
	svc := &abstract.BIMI{Logo: []byte(logo), Certificate: cert, HostCNAME: hostCNAME()}
	uc := newUsecase(t, svc)

	for _, host := range []string{"bimi.example.com.", "bimi.example.com", "BIMI.example.com."} {
		got, err := uc.Logo(host, svc.LogoName())
		if err != nil {
			t.Fatalf("Logo(%q): %v", host, err)
		}
		if string(got) != logo {
			t.Errorf("Logo(%q) = %q; want the logo", host, got)
		}
	}

	got, err := uc.Certificate("bimi.example.com.", svc.CertificateName())
	if err != nil {
		t.Fatalf("Certificate: %v", err)
	}
	if string(got) != string(cert) {
		t.Errorf("Certificate() = %q; want the certificate", got)
	}
}

func TestFiles_NotFound(t *testing.T) {
	svc := &abstract.BIMI{Logo: []byte(logo), Certificate: certificate(t), HostCNAME: hostCNAME()}
	uc := newUsecase(t, svc)

	for name, call := range map[string]func() ([]byte, error){
		"stale logo name":        func() ([]byte, error) { return uc.Logo("bimi.example.com.", "0000000000000000.svg") },
		"wrong certificate name": func() ([]byte, error) { return uc.Certificate("bimi.example.com.", svc.LogoName()) },
		"unknown domain":         func() ([]byte, error) { return uc.Logo("bimi.unknown.example.", svc.LogoName()) },
		// User content is only ever served from the bimi. host.
		"bare domain":                func() ([]byte, error) { return uc.Logo("example.com.", svc.LogoName()) },
		"other host":                 func() ([]byte, error) { return uc.Logo("mta-sts.example.com.", svc.LogoName()) },
		"certificate on bare domain": func() ([]byte, error) { return uc.Certificate("example.com.", svc.CertificateName()) },
		"certificate on other host":  func() ([]byte, error) { return uc.Certificate("mta-sts.example.com.", svc.CertificateName()) },
	} {
		if _, err := call(); !errors.Is(err, happydns.ErrNotFound) {
			t.Errorf("%s: error = %v; want ErrNotFound", name, err)
		}
	}
}

func TestFiles_NotHosted(t *testing.T) {
	svc := &abstract.BIMI{Logo: []byte(logo)}
	uc := newUsecase(t, svc)

	if _, err := uc.Logo("bimi.example.com.", svc.LogoName()); !errors.Is(err, happydns.ErrNotFound) {
		t.Errorf("Logo(no CNAME) error = %v; want ErrNotFound", err)
	}
	if managed, err := uc.IsManaged("bimi.example.com."); err != nil || managed {
		t.Errorf("IsManaged(no CNAME) = (%v, %v); want (false, nil)", managed, err)
	}
}

func TestIsManaged(t *testing.T) {
	uc := newUsecase(t, &abstract.BIMI{Logo: []byte(logo), HostCNAME: hostCNAME()})

	for fqdn, want := range map[string]bool{
		"bimi.example.com.":     true,
		"bimi.example.com":      true,
		"BIMI.example.com.":     true,
		"bimi.unknown.example.": false,
		// Never authorise a certificate for a name we do not answer on, even
		// when the domain itself is hosted here.
		"example.com.":         false,
		"mta-sts.example.com.": false,
	} {
		got, err := uc.IsManaged(fqdn)
		if err != nil {
			t.Fatalf("IsManaged(%q): %v", fqdn, err)
		}
		if got != want {
			t.Errorf("IsManaged(%q) = %v; want %v", fqdn, got, want)
		}
	}
}

func TestStorageError(t *testing.T) {
	boom := errors.New("storage is down")
	domains, zones := hostedservicetest.NewStore(t, "example.com.", &abstract.BIMI{Logo: []byte(logo), HostCNAME: hostCNAME()})
	domains.Err = boom
	uc := NewUsecase(domains, zones)

	if managed, err := uc.IsManaged("bimi.example.com."); !errors.Is(err, boom) || managed {
		t.Errorf("IsManaged() = (%v, %v); want (false, the storage error)", managed, err)
	}
	if _, err := uc.Logo("bimi.example.com.", "x.svg"); !errors.Is(err, boom) {
		t.Errorf("Logo() error = %v; want the storage error", err)
	}
}

var _ happydns.BIMIUsecase = (*Usecase)(nil)

// A new logo changes the name in the zone being edited, but the live record
// still points to the one of the published zone until it is applied.
func TestFiles_UnappliedLogo(t *testing.T) {
	const newLogo = `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>New</title></svg>`

	live := &abstract.BIMI{Logo: []byte(logo), HostCNAME: hostCNAME()}
	edited := &abstract.BIMI{Logo: []byte(newLogo), HostCNAME: hostCNAME()}

	wipID, pubID := happydns.Identifier{0x01}, happydns.Identifier{0x02}
	zone := func(id happydns.Identifier, svc *abstract.BIMI) *happydns.Zone {
		return &happydns.Zone{
			ZoneMeta: happydns.ZoneMeta{Id: id},
			Services: map[happydns.Subdomain][]*happydns.Service{"": {{Service: svc}}},
		}
	}
	uc := NewUsecase(
		&hostedservicetest.FakeDomains{ByName: map[string][]*happydns.Domain{
			"example.com.": {{DomainName: "example.com.", ZoneHistory: []happydns.Identifier{wipID, pubID}}},
		}},
		&hostedservicetest.FakeZones{ByID: map[string]*happydns.Zone{
			wipID.String(): zone(wipID, edited),
			pubID.String(): zone(pubID, live),
		}},
	)

	for name, svc := range map[string]*abstract.BIMI{"published": live, "edited": edited} {
		got, err := uc.Logo("bimi.example.com.", svc.LogoName())
		if err != nil {
			t.Fatalf("Logo(%s name): %v", name, err)
		}
		if string(got) != string(svc.Logo) {
			t.Errorf("Logo(%s name) = %q; want %q", name, got, svc.Logo)
		}
	}
}
