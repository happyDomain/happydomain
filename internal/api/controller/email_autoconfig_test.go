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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

type fakeEmailAutoconfig struct {
	happydns.EmailAutoconfigUsecase

	body []byte
	err  error

	// Domain and email address the last call received.
	domain, email string
}

func (f *fakeEmailAutoconfig) MozillaConfig(domain, email string) ([]byte, error) {
	f.domain, f.email = domain, email
	return f.body, f.err
}

func (f *fakeEmailAutoconfig) AutodiscoverConfig(domain, email string) ([]byte, error) {
	f.domain, f.email = domain, email
	return f.body, f.err
}

func serveEmailAutoconfig(t *testing.T, uc *fakeEmailAutoconfig, handler func(*EmailAutoconfigController, *gin.Context), req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler(NewEmailAutoconfigController(uc), c)

	return w
}

func TestMozillaAutoconfig(t *testing.T) {
	handler := (*EmailAutoconfigController).MozillaAutoconfig

	t.Run("served", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{body: []byte("<clientConfig/>")}
		req := httptest.NewRequest(http.MethodGet, "http://autoconfig.example.com:8081/mail/config-v1.1.xml", nil)

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusOK || w.Body.String() != "<clientConfig/>" {
			t.Errorf("got %d %q; want 200 with the configuration", w.Code, w.Body.String())
		}
		if uc.domain != "autoconfig.example.com." {
			t.Errorf("usecase asked for %q; want the Host without its port, fully qualified", uc.domain)
		}
		if cc := w.Header().Get("Cache-Control"); cc != "" {
			t.Errorf("Cache-Control = %q on a success; want none", cc)
		}
	})

	t.Run("domain taken from the email address", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{body: []byte("<clientConfig/>")}
		req := httptest.NewRequest(http.MethodGet, "http://autoconfig.example.com/mail/config-v1.1.xml?emailaddress=user@example.org", nil)

		serveEmailAutoconfig(t, uc, handler, req)

		if uc.domain != "example.org." || uc.email != "user@example.org" {
			t.Errorf("usecase asked for (%q, %q); want (%q, %q)", uc.domain, uc.email, "example.org.", "user@example.org")
		}
	})

	t.Run("missing domain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/mail/config-v1.1.xml", nil)
		req.Host = ""

		w := serveEmailAutoconfig(t, &fakeEmailAutoconfig{}, handler, req)

		if w.Code != http.StatusBadRequest || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 400 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})

	t.Run("not found", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{err: happydns.ErrNotFound}
		req := httptest.NewRequest(http.MethodGet, "http://autoconfig.example.com/mail/config-v1.1.xml", nil)

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusNotFound || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 404 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})

	t.Run("usecase failure", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{err: errors.New("storage is down")}
		req := httptest.NewRequest(http.MethodGet, "http://autoconfig.example.com/mail/config-v1.1.xml", nil)

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusInternalServerError || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 500 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})
}

func TestMSAutodiscover(t *testing.T) {
	handler := (*EmailAutoconfigController).MSAutodiscover

	t.Run("domain taken from the POST body", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{body: []byte("<Autodiscover/>")}
		body := `<Autodiscover><Request><EMailAddress>user@example.org</EMailAddress></Request></Autodiscover>`
		req := httptest.NewRequest(http.MethodPost, "http://autodiscover.example.com/Autodiscover/Autodiscover.xml", strings.NewReader(body))

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got %d; want 200", w.Code)
		}
		if uc.domain != "example.org." || uc.email != "user@example.org" {
			t.Errorf("usecase asked for (%q, %q); want (%q, %q)", uc.domain, uc.email, "example.org.", "user@example.org")
		}
	})

	t.Run("missing domain", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/Autodiscover/Autodiscover.xml", nil)
		req.Host = ""

		w := serveEmailAutoconfig(t, &fakeEmailAutoconfig{}, handler, req)

		if w.Code != http.StatusBadRequest || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 400 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})

	t.Run("not found", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{err: happydns.ErrNotFound}
		req := httptest.NewRequest(http.MethodGet, "http://autodiscover.example.com/Autodiscover/Autodiscover.xml", nil)

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusNotFound || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 404 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})

	t.Run("usecase failure", func(t *testing.T) {
		uc := &fakeEmailAutoconfig{err: errors.New("storage is down")}
		req := httptest.NewRequest(http.MethodGet, "http://autodiscover.example.com/Autodiscover/Autodiscover.xml", nil)

		w := serveEmailAutoconfig(t, uc, handler, req)

		if w.Code != http.StatusInternalServerError || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("got %d with Cache-Control %q; want 500 with no-store", w.Code, w.Header().Get("Cache-Control"))
		}
	})
}
