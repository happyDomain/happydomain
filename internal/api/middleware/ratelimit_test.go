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
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"git.happydns.org/happyDomain/internal/api/middleware"
)

// newTestRouter builds a gin engine with the given limiter guarding a single
// endpoint that returns 200 when allowed through.
func newTestRouter(limiter *middleware.IPRateLimiter) *gin.Engine {
	r := gin.New()
	r.Use(limiter.Middleware())
	r.POST("/recovery", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

// doRequest issues a POST /recovery from the given source IP and returns the
// response status.
func doRequest(r *gin.Engine, ip string) int {
	req := httptest.NewRequest(http.MethodPost, "/recovery", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// TestRateLimiter_AllowsWithinBurst verifies that requests up to the burst size
// are allowed through.
func TestRateLimiter_AllowsWithinBurst(t *testing.T) {
	// A very slow refill rate ensures the burst is the only allowance during
	// the test, making the assertions deterministic.
	limiter := middleware.NewIPRateLimiter(rate.Every(time.Hour), 5)
	r := newTestRouter(limiter)

	for i := range 5 {
		if code := doRequest(r, "10.0.0.1"); code != http.StatusOK {
			t.Fatalf("request %d within burst should be allowed, got %d", i+1, code)
		}
	}
}

// TestRateLimiter_BlocksBeyondBurst verifies that exceeding the burst yields 429.
func TestRateLimiter_BlocksBeyondBurst(t *testing.T) {
	limiter := middleware.NewIPRateLimiter(rate.Every(time.Hour), 3)
	r := newTestRouter(limiter)

	for i := range 3 {
		if code := doRequest(r, "10.0.0.2"); code != http.StatusOK {
			t.Fatalf("request %d within burst should be allowed, got %d", i+1, code)
		}
	}

	if code := doRequest(r, "10.0.0.2"); code != http.StatusTooManyRequests {
		t.Fatalf("request beyond burst should be rejected with 429, got %d", code)
	}
}

// TestRateLimiter_PerIPIsolation ensures one client exhausting its bucket does
// not affect other clients.
func TestRateLimiter_PerIPIsolation(t *testing.T) {
	limiter := middleware.NewIPRateLimiter(rate.Every(time.Hour), 1)
	r := newTestRouter(limiter)

	// Exhaust the bucket for the first IP.
	if code := doRequest(r, "10.0.0.3"); code != http.StatusOK {
		t.Fatalf("first request for IP A should be allowed, got %d", code)
	}
	if code := doRequest(r, "10.0.0.3"); code != http.StatusTooManyRequests {
		t.Fatalf("second request for IP A should be rejected, got %d", code)
	}

	// A different IP must still have its full allowance.
	if code := doRequest(r, "10.0.0.4"); code != http.StatusOK {
		t.Fatalf("first request for IP B should be allowed despite IP A being throttled, got %d", code)
	}
}
