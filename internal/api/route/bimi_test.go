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

package route

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/usecase/bimi"
	"git.happydns.org/happyDomain/internal/usecase/hostedservice/hostedservicetest"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

const bimiLogo = `<svg version="1.2" baseProfile="tiny-ps" xmlns="http://www.w3.org/2000/svg"><title>Example</title></svg>`

func bimiCertificate(t *testing.T) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "example.com"}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// bimiUsecase returns a real BIMI usecase over example.com, hosting bimiLogo.
func bimiUsecase(t *testing.T) (happydns.BIMIUsecase, *abstract.BIMI) {
	t.Helper()

	svc := &abstract.BIMI{
		Logo:        []byte(bimiLogo),
		Certificate: bimiCertificate(t),
		HostCNAME: &dns.CNAME{
			Hdr:    dns.RR_Header{Name: "bimi", Rrtype: dns.TypeCNAME, Class: dns.ClassINET},
			Target: "happydomain.example.com.",
		},
	}
	domains, zones := hostedservicetest.NewStore(t, "example.com.", svc)
	return bimi.NewUsecase(domains, zones), svc
}

func bimiRouteStatus(t *testing.T, rl gin.HandlerFunc, uc happydns.BIMIUsecase, path string) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	DeclareBIMIRoutes(router.Group("/"), rl, uc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = "bimi.example.com"
	router.ServeHTTP(w, req)
	return w.Code
}

func TestDeclareBIMIRoutes(t *testing.T) {
	passthrough := func(c *gin.Context) { c.Next() }
	uc, svc := bimiUsecase(t)

	if got := bimiRouteStatus(t, passthrough, uc, "/bimi/"+svc.LogoName()); got != http.StatusOK {
		t.Errorf("GET /bimi/<logo> = %d; want %d", got, http.StatusOK)
	}
	if got := bimiRouteStatus(t, passthrough, uc, "/bimi/"+svc.CertificateName()); got != http.StatusOK {
		t.Errorf("GET /bimi/<certificate> = %d; want %d", got, http.StatusOK)
	}
	if got := bimiRouteStatus(t, passthrough, uc, "/bimi/0000000000000000.pem"); got != http.StatusNotFound {
		t.Errorf("GET /bimi/<stale certificate> = %d; want %d", got, http.StatusNotFound)
	}
	if got := bimiRouteStatus(t, passthrough, uc, "/bimi/0000000000000000.svg"); got != http.StatusNotFound {
		t.Errorf("GET /bimi/<stale logo> = %d; want %d", got, http.StatusNotFound)
	}

	// The usecase is optional: without it the path is not exposed.
	if got := bimiRouteStatus(t, passthrough, nil, "/bimi/"+svc.LogoName()); got != http.StatusNotFound {
		t.Errorf("without usecase, GET /bimi/<logo> = %d; want %d", got, http.StatusNotFound)
	}
}

func TestDeclareBIMIRoutes_RateLimited(t *testing.T) {
	deny := func(c *gin.Context) { c.AbortWithStatus(http.StatusTooManyRequests) }
	uc, svc := bimiUsecase(t)

	if got := bimiRouteStatus(t, deny, uc, "/bimi/"+svc.LogoName()); got != http.StatusTooManyRequests {
		t.Errorf("GET /bimi/<logo> = %d; want the rate limiter to apply (%d)", got, http.StatusTooManyRequests)
	}
}

// The Caddy hook authorises a certificate for the bimi. host of a domain
// hosting its BIMI files, and for nothing else.
func TestDeclareCaddyRoutes_BIMI(t *testing.T) {
	uc, _ := bimiUsecase(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	DeclareCaddyRoutes(router.Group("/api"), func(c *gin.Context) { c.Next() }, uc)

	for domain, want := range map[string]int{
		"bimi.example.com":     http.StatusOK,
		"bimi.unknown.example": http.StatusNotFound,
		"example.com":          http.StatusNotFound,
		"mta-sts.example.com":  http.StatusNotFound,
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/caddy/ask?domain="+domain, nil))
		if w.Code != want {
			t.Errorf("GET /api/caddy/ask?domain=%s = %d; want %d", domain, w.Code, want)
		}
	}
}
