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

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
)

// clientKeyFor builds a request coming from remoteAddr and returns the bucket
// it is rate limited under.
func clientKeyFor(t *testing.T, remoteAddr string) string {
	t.Helper()

	gin.SetMode(gin.TestMode)
	c, engine := gin.CreateTestContext(httptest.NewRecorder())
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("SetTrustedProxies(nil) => %s", err.Error())
	}

	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.RemoteAddr = remoteAddr

	return middleware.ClientKey(c)
}

func TestClientKey(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{"IPv4 keys on the exact address", "192.0.2.42:1234", "192.0.2.42"},
		{"IPv6 keys on its /64", "[2001:db8:1:2:3:4:5:6]:1234", "2001:db8:1:2::/64"},
		{"4-in-6 behaves as IPv4", "[::ffff:192.0.2.42]:1234", "192.0.2.42"},
		{"unparseable address falls back", "@", middleware.UnknownClientKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientKeyFor(t, tt.remoteAddr); got != tt.want {
				t.Fatalf("ClientKey(%q) = %q, want %q", tt.remoteAddr, got, tt.want)
			}
		})
	}
}

// TestClientKeyCollapsesIPv6Subnet is the point of the /64 masking: rotating
// addresses inside a single delegated prefix must not buy a fresh bucket.
func TestClientKeyCollapsesIPv6Subnet(t *testing.T) {
	first := clientKeyFor(t, "[2001:db8:1:2::1]:1234")
	second := clientKeyFor(t, "[2001:db8:1:2:dead:beef:cafe:1]:4321")

	if first != second {
		t.Fatalf("two addresses of the same /64 give distinct keys: %q != %q", first, second)
	}

	other := clientKeyFor(t, "[2001:db8:1:3::1]:1234")
	if other == first {
		t.Fatalf("addresses of distinct /64 share the key %q", other)
	}
}
