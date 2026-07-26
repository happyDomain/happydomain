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

package app

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

// clientIPSeenBy sends a request from remoteAddr carrying the given
// X-Forwarded-For header to an engine configured with trustedProxies, and
// returns the address the handlers would attribute it to.
func clientIPSeenBy(t *testing.T, trustedProxies []string, remoteAddr, forwardedFor string) string {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	if err := setupTrustedProxies(router, &happydns.Options{TrustedProxies: trustedProxies}); err != nil {
		t.Fatalf("setupTrustedProxies(%v) => %s", trustedProxies, err.Error())
	}

	var seen string
	router.GET("/", func(c *gin.Context) {
		seen = c.ClientIP()
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	router.ServeHTTP(httptest.NewRecorder(), req)

	return seen
}

// TestSetupTrustedProxiesIgnoresForgedHeader is the regression test for the
// audit finding: with no trusted proxy declared, a forged X-Forwarded-For must
// not change the identity a caller is rate limited under.
func TestSetupTrustedProxiesIgnoresForgedHeader(t *testing.T) {
	got := clientIPSeenBy(t, nil, "192.0.2.42:1234", "203.0.113.77")
	if got != "192.0.2.42" {
		t.Fatalf("ClientIP() = %q, want the socket address 192.0.2.42", got)
	}
}

// TestSetupTrustedProxiesHonorsDeclaredProxy checks the other half: a declared
// reverse proxy still gets to report the real client.
func TestSetupTrustedProxiesHonorsDeclaredProxy(t *testing.T) {
	for _, trusted := range [][]string{{"192.0.2.42"}, {"192.0.2.0/24"}} {
		got := clientIPSeenBy(t, trusted, "192.0.2.42:1234", "203.0.113.77")
		if got != "203.0.113.77" {
			t.Fatalf("with -trusted-proxy %v, ClientIP() = %q, want 203.0.113.77", trusted, got)
		}
	}

	// A peer that is not the declared proxy gains nothing.
	got := clientIPSeenBy(t, []string{"192.0.2.42"}, "198.51.100.9:1234", "203.0.113.77")
	if got != "198.51.100.9" {
		t.Fatalf("ClientIP() = %q, want the socket address 198.51.100.9", got)
	}
}

// warnedOn reports whether the middleware emits its warning for a request from
// remoteAddr carrying forwardedFor, given the declared trusted proxies.
func warnedOn(t *testing.T, trustedProxies []string, remoteAddr, forwardedFor string) bool {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	if err := setupTrustedProxies(router, &happydns.Options{TrustedProxies: trustedProxies}); err != nil {
		t.Fatalf("setupTrustedProxies(%v) => %s", trustedProxies, err.Error())
	}

	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	router.Use(warnUntrustedForwardedHeaders(&happydns.Options{TrustedProxies: trustedProxies}))
	router.GET("/", func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	router.ServeHTTP(httptest.NewRecorder(), req)

	return logged.Len() > 0
}

func TestWarnUntrustedForwardedHeaders(t *testing.T) {
	if !warnedOn(t, nil, "192.0.2.42:1234", "203.0.113.77") {
		t.Fatal("no warning for a forged header from an undeclared peer")
	}

	if warnedOn(t, nil, "192.0.2.42:1234", "") {
		t.Fatal("warning emitted for a request carrying no client IP header")
	}

	// The regression this guards: a declared proxy that forwards a header gin
	// cannot parse, or a request originating on the proxy host itself, makes
	// ClientIP() fall back to the peer address. Warning there would tell the
	// operator to widen a trust list that is already correct.
	for _, forwardedFor := range []string{"unknown", "10.0.0.5", "203.0.113.77"} {
		if warnedOn(t, []string{"10.0.0.5"}, "10.0.0.5:1234", forwardedFor) {
			t.Fatalf("warning emitted for the declared proxy sending %q", forwardedFor)
		}
	}

	if !warnedOn(t, []string{"10.0.0.0/8"}, "192.0.2.42:1234", "203.0.113.77") {
		t.Fatal("no warning for a peer outside the declared range")
	}
}

func TestSetupTrustedProxiesRejectsInvalidValue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	err := setupTrustedProxies(gin.New(), &happydns.Options{TrustedProxies: []string{"10.0.0.0/33"}})
	if err == nil {
		t.Fatal("setupTrustedProxies([10.0.0.0/33]) = nil error, want an error")
	}
}
