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

package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

type claimEverything struct{}

func (claimEverything) IsManaged(string) (bool, error) { return true, nil }

func caddyAskStatus(t *testing.T, rl gin.HandlerFunc, validators ...happydns.HostedDomainValidator) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	DeclareCaddyRoutes(router.Group("/api"), rl, validators...)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/caddy/ask?domain=mta-sts.example.com", nil))
	return w.Code
}

func TestDeclareCaddyRoutes(t *testing.T) {
	passthrough := func(c *gin.Context) { c.Next() }

	if got := caddyAskStatus(t, passthrough, claimEverything{}); got != http.StatusOK {
		t.Errorf("with a validator, GET /api/caddy/ask = %d; want %d", got, http.StatusOK)
	}

	// Nothing could ever answer positively: the endpoint is not exposed.
	if got := caddyAskStatus(t, passthrough); got != http.StatusNotFound {
		t.Errorf("without validator, GET /api/caddy/ask = %d; want %d", got, http.StatusNotFound)
	}
}

func TestDeclareCaddyRoutes_RateLimited(t *testing.T) {
	deny := func(c *gin.Context) { c.AbortWithStatus(http.StatusTooManyRequests) }

	if got := caddyAskStatus(t, deny, claimEverything{}); got != http.StatusTooManyRequests {
		t.Errorf("GET /api/caddy/ask = %d; want the rate limiter to apply (%d)", got, http.StatusTooManyRequests)
	}
}
