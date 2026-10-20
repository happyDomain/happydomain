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

type fakeMTASTSUsecase struct {
	happydns.MTASTSUsecase
}

func (fakeMTASTSUsecase) Policy(string) ([]byte, error) {
	return []byte("version: STSv1\r\n"), nil
}

func policyRouteStatus(t *testing.T, rl gin.HandlerFunc, uc happydns.MTASTSUsecase) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	DeclareMTASTSRoutes(router.Group("/"), rl, uc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/mta-sts.txt", nil)
	req.Host = "mta-sts.example.com"
	router.ServeHTTP(w, req)
	return w.Code
}

func TestDeclareMTASTSRoutes(t *testing.T) {
	passthrough := func(c *gin.Context) { c.Next() }

	if got := policyRouteStatus(t, passthrough, fakeMTASTSUsecase{}); got != http.StatusOK {
		t.Errorf("GET /.well-known/mta-sts.txt = %d; want %d", got, http.StatusOK)
	}

	// The usecase is optional: without it the path is not exposed.
	if got := policyRouteStatus(t, passthrough, nil); got != http.StatusNotFound {
		t.Errorf("without usecase, GET /.well-known/mta-sts.txt = %d; want %d", got, http.StatusNotFound)
	}
}

func TestDeclareMTASTSRoutes_RateLimited(t *testing.T) {
	deny := func(c *gin.Context) { c.AbortWithStatus(http.StatusTooManyRequests) }

	if got := policyRouteStatus(t, deny, fakeMTASTSUsecase{}); got != http.StatusTooManyRequests {
		t.Errorf("GET /.well-known/mta-sts.txt = %d; want the rate limiter to apply (%d)", got, http.StatusTooManyRequests)
	}
}
