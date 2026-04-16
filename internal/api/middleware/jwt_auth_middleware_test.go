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
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/api/middleware"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// stubAuthUsecase is a no-op implementation of happydns.AuthenticationUsecase.
// The middleware should never reach a method of this stub when the token is a
// session ID, and should never reach it on a malformed JWT either (it returns
// after logging). We still assert it was not called by leaving the methods
// panicking — if any test trips one, we know the branching logic regressed.
type stubAuthUsecase struct{}

func (stubAuthUsecase) AuthenticateUserWithPassword(_ happydns.LoginRequest) (*happydns.User, error) {
	panic("AuthenticateUserWithPassword should not be called in these tests")
}

func (stubAuthUsecase) CompleteAuthentication(_ happydns.UserInfo) (*happydns.User, error) {
	panic("CompleteAuthentication should not be called in these tests")
}

// captureLog redirects the default logger to a buffer for the duration of fn.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})

	fn()
	return buf.String()
}

func newRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.JwtAuthMiddleware(stubAuthUsecase{}, "HS256", []byte("test-secret")))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func Test_JwtAuthMiddleware_SessionIDTokenIsSilent(t *testing.T) {
	r := newRouter()

	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+sessionUC.NewSessionID())
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	})

	if strings.Contains(output, "bad JWT claims") {
		t.Errorf("expected no %q log for a session-ID token, got:\n%s", "bad JWT claims", output)
	}
	if strings.TrimSpace(output) != "" {
		t.Errorf("expected no log output at all for a session-ID token, got:\n%s", output)
	}
}

func Test_JwtAuthMiddleware_MalformedTokenStillLogs(t *testing.T) {
	r := newRouter()

	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		// Contains a dot, so it can't match the session-ID shape and will be
		// routed to the JWT parser, which will fail.
		req.Header.Set("Authorization", "Bearer not.a.jwt")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	})

	if !strings.Contains(output, "bad JWT claims") {
		t.Errorf("expected %q log for a malformed JWT, got:\n%s", "bad JWT claims", output)
	}
}

func Test_JwtAuthMiddleware_NoAuthHeaderIsSilent(t *testing.T) {
	r := newRouter()

	output := captureLog(t, func() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	})

	if strings.TrimSpace(output) != "" {
		t.Errorf("expected no log output without an Authorization header, got:\n%s", output)
	}
}
