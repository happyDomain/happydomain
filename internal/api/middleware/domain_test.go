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
	"git.happydns.org/happyDomain/model"
)

// domainOwnerOnlyStatus runs DomainOwnerOnly on a request where domain is owned
// by ownerID and the caller is callerID, and reports the resulting status code
// along with whether the guarded handler ran.
func domainOwnerOnlyStatus(t *testing.T, ownerID, callerID happydns.Identifier) (int, bool) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	did, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier: %v", err)
	}

	reached := false
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.GET("/",
		func(c *gin.Context) {
			c.Set("LoggedUser", &happydns.User{Id: callerID})
			c.Set("domain", &happydns.Domain{Id: did, Owner: ownerID})
		},
		middleware.DomainOwnerOnly(),
		func(c *gin.Context) {
			reached = true
			c.Status(http.StatusOK)
		},
	)

	c.Request = httptest.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, c.Request)

	return w.Code, reached
}

func TestDomainOwnerOnly_Owner(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()

	code, reached := domainOwnerOnlyStatus(t, uid, uid)
	if !reached {
		t.Error("expected the guarded handler to run for the domain owner")
	}
	if code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, code)
	}
}

func TestDomainOwnerOnly_Grantee(t *testing.T) {
	ownerID, _ := happydns.NewRandomIdentifier()
	granteeID, _ := happydns.NewRandomIdentifier()

	code, reached := domainOwnerOnlyStatus(t, ownerID, granteeID)
	if reached {
		t.Error("expected the guarded handler to be skipped for an invited user")
	}
	if code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, code)
	}
}

func TestDomainOwnerOnly_NoDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reached := false
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.GET("/", middleware.DomainOwnerOnly(), func(c *gin.Context) {
		reached = true
	})

	c.Request = httptest.NewRequest("GET", "/", nil)
	engine.ServeHTTP(w, c.Request)

	if reached {
		t.Error("expected the guarded handler to be skipped when no domain is resolved")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
