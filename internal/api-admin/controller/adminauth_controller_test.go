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

package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/adminauth"
	happydns "git.happydns.org/happyDomain/model"
)

// newLoginRouter builds a router serving the admin login endpoint with a
// cleartext admin password, which keeps the tests fast: what is exercised here
// is the throttling around verification, not the hashing itself.
func newLoginRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	cfg := &happydns.Options{
		AdminPasswordHash: "s3cr3t",
		JWTSecretKey:      []byte("this-is-a-32-byte-long-secret!!!"),
	}

	router := gin.New()
	router.POST("/api/admin-login", NewAdminAuthController(cfg).Login)

	return router
}

func postLogin(t *testing.T, router *gin.Engine, client, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/admin-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = client + ":54321"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestAdminLoginSucceeds(t *testing.T) {
	w := postLogin(t, newLoginRouter(), "192.0.2.1", `{"password":"s3cr3t"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"token"`) {
		t.Errorf("response carries no token: %s", w.Body.String())
	}
}

func TestAdminLoginLocksOutGuessing(t *testing.T) {
	router := newLoginRouter()

	// The first attempts are answered normally, then the client is locked out
	// instead of being handed an unlimited guessing oracle.
	var locked bool
	for i := range 10 {
		w := postLogin(t, router, "192.0.2.1", `{"password":"wrong"}`)

		switch w.Code {
		case http.StatusUnauthorized:
			continue
		case http.StatusTooManyRequests:
			locked = true
			if w.Header().Get("Retry-After") == "" {
				t.Error("429 response carries no Retry-After header")
			}
		default:
			t.Fatalf("attempt %d: status = %d, want 401 or 429 (%s)", i, w.Code, w.Body.String())
		}

		if locked {
			break
		}
	}

	if !locked {
		t.Fatal("10 wrong passwords in a row never triggered a lockout")
	}

	// The lockout must hold even when the right password is finally presented,
	// otherwise it does not slow a dictionary run down at all.
	if w := postLogin(t, router, "192.0.2.1", `{"password":"s3cr3t"}`); w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d while locked out", w.Code, http.StatusTooManyRequests)
	}

	// A different client is unaffected.
	if w := postLogin(t, router, "192.0.2.2", `{"password":"s3cr3t"}`); w.Code != http.StatusOK {
		t.Errorf("unrelated client got status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminLoginRejectsOversizedBody(t *testing.T) {
	body := fmt.Sprintf(`{"password":"%s"}`, strings.Repeat("a", adminauth.MaxLoginBodyBytes+1))

	w := postLogin(t, newLoginRouter(), "192.0.2.1", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for a body over the limit", w.Code, http.StatusBadRequest)
	}
}

func TestAdminLoginDisabledWithoutPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/admin-login", NewAdminAuthController(&happydns.Options{}).Login)

	if w := postLogin(t, router, "192.0.2.1", `{"password":"whatever"}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d when no admin password is configured", w.Code, http.StatusNotFound)
	}
}

// TestAdminLoginDoesNotReadWholeBodyBeforeRejecting guards the DoS half of the
// throttle: a locked-out client is answered without the server draining, and
// therefore buffering, whatever it chose to send.
func TestAdminLoginDoesNotReadWholeBodyBeforeRejecting(t *testing.T) {
	router := newLoginRouter()

	for range 10 {
		if w := postLogin(t, router, "192.0.2.1", `{"password":"wrong"}`); w.Code == http.StatusTooManyRequests {
			break
		}
	}

	counted := &countingReader{}
	req := httptest.NewRequest(http.MethodPost, "/api/admin-login", counted)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:54321"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if counted.read > 0 {
		t.Errorf("%d bytes of the body were read from a locked-out client, want 0", counted.read)
	}
}

// countingReader is an endless body that reports how much of it was consumed.
type countingReader struct {
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	if r.read > 1<<20 {
		return 0, io.EOF
	}

	for i := range p {
		p[i] = 'a'
	}
	r.read += len(p)

	return len(p), nil
}
