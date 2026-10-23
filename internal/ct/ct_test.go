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

package ct

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestNames(t *testing.T) {
	issuances := []Issuance{
		{DNSNames: []string{"example.com", "*.example.com", "WWW.Example.com."}},
		{DNSNames: []string{"mail.example.com", " api.example.com ", "*.dev.example.com"}},
		{DNSNames: []string{"example.org", "notexample.com", "www.example.com.evil.net"}},
	}

	got := Names(issuances, "Example.com.")
	want := []string{"api.example.com", "dev.example.com", "example.com", "mail.example.com", "www.example.com"}
	if !slices.Equal(got, want) {
		t.Errorf("Names = %v; want %v", got, want)
	}
}

func TestNamesEmpty(t *testing.T) {
	if got := Names(nil, "example.com"); len(got) != 0 {
		t.Errorf("Names(nil) = %v; want none", got)
	}
}

func TestGet(t *testing.T) {
	var gotUA, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := Config{}.get(context.Background(), srv.URL, http.Header{"Authorization": {"Bearer k"}})
	if err != nil || string(body) != "ok" {
		t.Fatalf("get = %q, %v; want ok", body, err)
	}
	if gotUA != DefaultUserAgent {
		t.Errorf("User-Agent = %q; want the default one", gotUA)
	}
	if gotAuth != "Bearer k" {
		t.Errorf("Authorization = %q; want the header given", gotAuth)
	}

	if _, err := (Config{UserAgent: "custom"}).get(context.Background(), srv.URL, nil); err != nil || gotUA != "custom" {
		t.Errorf("User-Agent = %q, %v; want custom", gotUA, err)
	}
}

func TestGetErrors(t *testing.T) {
	for _, tc := range []struct {
		status     int
		retryAfter string
		want       error
		wantDelay  time.Duration
	}{
		{http.StatusTooManyRequests, "120", ErrRateLimited, 2 * time.Minute},
		{http.StatusBadGateway, "", ErrUnavailable, 0},
		{http.StatusServiceUnavailable, "soon", ErrUnavailable, 0},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.retryAfter != "" {
				w.Header().Set("Retry-After", tc.retryAfter)
			}
			w.WriteHeader(tc.status)
		}))

		_, err := Config{}.get(context.Background(), srv.URL, nil)
		srv.Close()

		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v; want %v", tc.status, err, tc.want)
		}
		var herr *HTTPError
		if !errors.As(err, &herr) || herr.RetryAfter != tc.wantDelay {
			t.Errorf("status %d: err = %#v; want an HTTPError retrying after %s", tc.status, err, tc.wantDelay)
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := (Config{}).get(context.Background(), srv.URL, nil); err == nil || errors.Is(err, ErrUnavailable) || errors.Is(err, ErrRateLimited) {
		t.Errorf("status 404: err = %v; want a plain HTTPError", err)
	}
}

func TestGetUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	if _, err := (Config{}).get(context.Background(), url, nil); !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v; want ErrUnavailable", err)
	}
}

func TestGetBodyLimit(t *testing.T) {
	defer func(size int64) { maxBodySize = size }(maxBodySize)
	maxBodySize = 16

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 17)))
	}))
	defer srv.Close()

	if _, err := (Config{}).get(context.Background(), srv.URL, nil); err == nil {
		t.Error("an answer over the limit is accepted")
	}
}

func TestGetBodyAtLimit(t *testing.T) {
	defer func(size int64) { maxBodySize = size }(maxBodySize)
	maxBodySize = 16

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", 16)))
	}))
	defer srv.Close()

	if body, err := (Config{}).get(context.Background(), srv.URL, nil); err != nil || len(body) != 16 {
		t.Errorf("get = %d bytes, %v; want an answer of the limit accepted", len(body), err)
	}
}

func TestGetCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Config{}).get(ctx, srv.URL, nil); !errors.Is(err, ErrUnavailable) || !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want ErrUnavailable, because canceled", err)
	}
}

// countingTransport counts the requests it carries.
type countingTransport struct{ n int }

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.n++
	return http.DefaultTransport.RoundTrip(r)
}

func TestGetClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	transport := &countingTransport{}
	if _, err := (Config{Client: &http.Client{Transport: transport}}).get(context.Background(), srv.URL, nil); err != nil || transport.n != 1 {
		t.Errorf("get = %v, %d requests through the client given; want 1", err, transport.n)
	}
}

func TestBaseURL(t *testing.T) {
	if got := (Config{}).baseURL("https://default.test"); got != "https://default.test" {
		t.Errorf("baseURL = %q; want the default", got)
	}
	if got := (Config{BaseURL: "http://127.0.0.1:8080/"}).baseURL("https://default.test"); got != "http://127.0.0.1:8080" {
		t.Errorf("baseURL = %q; want the one given, without trailing slash", got)
	}
}

func TestHTTPErrorMessage(t *testing.T) {
	if msg := (&HTTPError{StatusCode: http.StatusForbidden}).Error(); !strings.Contains(msg, "403 Forbidden") {
		t.Errorf("Error() = %q; want the status", msg)
	}
}
