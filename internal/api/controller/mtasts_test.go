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

package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

type fakeMTASTS struct {
	happydns.MTASTSUsecase

	body []byte
	err  error

	// Domain the last call received.
	domain string
}

func (f *fakeMTASTS) Policy(domain string) ([]byte, error) {
	f.domain = domain
	return f.body, f.err
}

func servePolicy(t *testing.T, uc *fakeMTASTS, host string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/.well-known/mta-sts.txt", nil)
	c.Request.Host = host

	NewMTASTSController(uc).Policy(c)

	return w
}

func TestMTASTSPolicy(t *testing.T) {
	uc := &fakeMTASTS{body: []byte("version: STSv1\r\n")}

	w := servePolicy(t, uc, "mta-sts.example.com:443")

	if w.Code != http.StatusOK || w.Body.String() != "version: STSv1\r\n" {
		t.Errorf("got %d %q; want 200 with the policy", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q; want text/plain; charset=utf-8", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != policyCacheControl {
		t.Errorf("Cache-Control = %q; want %q", cc, policyCacheControl)
	}
	if uc.domain != "mta-sts.example.com." {
		t.Errorf("usecase asked for %q; want the Host without its port, fully qualified", uc.domain)
	}
}

func TestMTASTSPolicy_Errors(t *testing.T) {
	for name, tc := range map[string]struct {
		uc   *fakeMTASTS
		host string
		want int
	}{
		"missing host":    {&fakeMTASTS{}, "", http.StatusBadRequest},
		"not found":       {&fakeMTASTS{err: happydns.ErrNotFound}, "mta-sts.example.com", http.StatusNotFound},
		"usecase failure": {&fakeMTASTS{err: errors.New("storage is down")}, "mta-sts.example.com", http.StatusInternalServerError},
	} {
		w := servePolicy(t, tc.uc, tc.host)

		if w.Code != tc.want {
			t.Errorf("%s: status = %d; want %d", name, w.Code, tc.want)
		}
		// A negative answer must never be cached: the admin may configure the
		// policy right after.
		if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("%s: Cache-Control = %q; want no-store", name, cc)
		}
	}
}
