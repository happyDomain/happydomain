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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

func TestKeystoreChoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var seen string
	r := gin.New()
	r.Use(KeystoreChoice())
	r.POST("/x", func(c *gin.Context) { seen = secret.KeystoreFromContext(c.Request.Context()) })

	for query, want := range map[string]string{"?keystore=abc": "abc", "": "", "?keystore=": ""} {
		seen = "unset"
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x"+query, nil))
		if seen != want {
			t.Errorf("POST /x%s: keystore = %q, want %q", query, seen, want)
		}
	}
}

// The scope is the public identifier the session is stored under, the one
// tidying checks, never the token the client presents.
func TestUnlockScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	user := &happydns.User{Id: happydns.Identifier([]byte("user")), Email: "user@example.com"}
	created, err := sessionUC.NewService(db).CreateUserSession(user, "script")
	if err != nil {
		t.Fatal(err)
	}

	var seen string
	r := gin.New()
	r.Use(sessions.Sessions(session.COOKIE_NAME, session.NewSessionStore(&happydns.Options{}, db, securecookie.GenerateRandomKey(32))))
	r.Use(UnlockScope())
	r.GET("/x", func(c *gin.Context) { seen = secret.UnlockScopeFromContext(c.Request.Context()) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+created.Token)
	r.ServeHTTP(httptest.NewRecorder(), req)
	if seen != created.Id {
		t.Errorf("scope = %q, want the public session id %q", seen, created.Id)
	}

	seen = "unset"
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	if seen != "" {
		t.Errorf("scope without a session = %q, want none", seen)
	}
}
