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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

type fakeBIMI struct {
	happydns.BIMIUsecase

	logo, cert []byte
	err        error

	// Arguments of the last call.
	host, name, kind string
}

func (f *fakeBIMI) Logo(host, name string) ([]byte, error) {
	f.host, f.name, f.kind = host, name, "logo"
	return f.logo, f.err
}

func (f *fakeBIMI) Certificate(host, name string) ([]byte, error) {
	f.host, f.name, f.kind = host, name, "certificate"
	return f.cert, f.err
}

func serveBIMI(t *testing.T, uc *fakeBIMI, host, file string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/bimi/"+file, nil)
	c.Request.Host = host
	c.Params = gin.Params{{Key: "file", Value: file}}

	NewBIMIController(uc).File(c)

	return w
}

func TestBIMIFile(t *testing.T) {
	uc := &fakeBIMI{logo: []byte("<svg/>"), cert: []byte("-----BEGIN CERTIFICATE-----")}

	for file, tc := range map[string]struct {
		kind, contentType, body string
	}{
		"0123456789abcdef.svg": {"logo", "image/svg+xml", "<svg/>"},
		"0123456789abcdef.pem": {"certificate", "application/pem-certificate-chain", "-----BEGIN CERTIFICATE-----"},
	} {
		w := serveBIMI(t, uc, "bimi.example.com:443", file)

		if w.Code != http.StatusOK || w.Body.String() != tc.body {
			t.Errorf("%s: got %d %q; want 200 with the %s", file, w.Code, w.Body.String(), tc.kind)
		}
		if uc.kind != tc.kind || uc.name != file || uc.host != "bimi.example.com." {
			t.Errorf("%s: usecase asked for the %s %q on %q; want the %s %q on the Host without its port, fully qualified", file, uc.kind, uc.name, uc.host, tc.kind, file)
		}
		for header, want := range map[string]string{
			"Content-Type":            tc.contentType,
			"Cache-Control":           bimiCacheControl,
			"Content-Security-Policy": bimiCSP,
			"X-Content-Type-Options":  "nosniff",
		} {
			if got := w.Header().Get(header); got != want {
				t.Errorf("%s: %s = %q; want %q", file, header, got, want)
			}
		}
	}
}

func TestBIMIFile_Errors(t *testing.T) {
	for name, tc := range map[string]struct {
		uc   *fakeBIMI
		host string
		file string
		want int
	}{
		"missing host":    {&fakeBIMI{}, "", "x.svg", http.StatusBadRequest},
		"unknown kind":    {&fakeBIMI{logo: []byte("<svg/>")}, "bimi.example.com", "x.html", http.StatusNotFound},
		"not found":       {&fakeBIMI{err: happydns.ErrNotFound}, "bimi.example.com", "x.svg", http.StatusNotFound},
		"usecase failure": {&fakeBIMI{err: errors.New("storage is down")}, "bimi.example.com", "x.pem", http.StatusInternalServerError},
	} {
		w := serveBIMI(t, tc.uc, tc.host, tc.file)

		if w.Code != tc.want {
			t.Errorf("%s: status = %d; want %d", name, w.Code, tc.want)
		}
		// A negative answer must never be cached: the user may upload the
		// file right after.
		if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("%s: Cache-Control = %q; want no-store", name, cc)
		}
	}
}
