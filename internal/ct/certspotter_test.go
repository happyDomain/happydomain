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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"
)

// certSpotterServer serves the recorded pages of issuances, the next one
// after the last ID of the previous one.
func certSpotterServer(t *testing.T, requests *[]*http.Request) *httptest.Server {
	pages := map[string]string{
		"":           "testdata/certspotter_page1.json",
		"3906571999": "testdata/certspotter_page2.json",
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r)

		if r.URL.Path != "/v1/issuances" {
			http.NotFound(w, r)
			return
		}

		file, ok := pages[r.URL.Query().Get("after")]
		if !ok {
			w.Write([]byte("[]"))
			return
		}

		body, err := os.ReadFile(file)
		if err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
}

func TestCertSpotterSearch(t *testing.T) {
	var requests []*http.Request
	srv := certSpotterServer(t, &requests)
	defer srv.Close()

	issuances, err := CertSpotter{Config{BaseURL: srv.URL, APIKey: "secret"}}.Search(context.Background(), Query{
		Domain:            "example.com.",
		IncludeSubdomains: true,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(requests) != 3 {
		t.Fatalf("%d requests; want the two pages, then the empty one", len(requests))
	}
	q := requests[0].URL.Query()
	if q.Get("domain") != "example.com" || q.Get("include_subdomains") != "true" || !slices.Equal(q["expand"], []string{"dns_names", "issuer"}) {
		t.Errorf("query = %v; want the domain without trailing dot, its subdomains, names and issuer", q)
	}
	for _, r := range requests {
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret" {
			t.Errorf("Authorization = %q; want the API key", auth)
		}
	}
	if after := requests[2].URL.Query().Get("after"); after != "3906572500" {
		t.Errorf("last page asked after %q; want the last ID of the previous one", after)
	}

	if len(issuances) != 3 {
		t.Fatalf("%d issuances; want 3", len(issuances))
	}
	first := issuances[0]
	if first.ID != "3906571584" || first.Issuer != "Let's Encrypt" || !slices.Equal(first.DNSNames, []string{"example.com", "www.example.com"}) ||
		first.CertSHA256 != "4d5c9e2f8a1b3c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d" ||
		!first.NotBefore.Equal(time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)) || !first.NotAfter.Equal(time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("first issuance = %+v", first)
	}
	if issuances[1].Issuer != "C=US, O=Example CA, CN=Example Issuing CA" {
		t.Errorf("issuer = %q; want the name, lacking a friendly one", issuances[1].Issuer)
	}

	want := []string{"dev.example.com", "example.com", "mail.example.com", "www.example.com"}
	if got := Names(issuances, "example.com"); !slices.Equal(got, want) {
		t.Errorf("names = %v; want %v", got, want)
	}
}

func TestCertSpotterPagingStops(t *testing.T) {
	for name, tc := range map[string]struct {
		page      func(request int) string
		want      int
		truncated bool
	}{
		"empty ID":    {func(int) string { return `[{"id":"","dns_names":["a.example.com"]}]` }, 1, false},
		"repeated ID": {func(int) string { return `[{"id":"1","dns_names":["a.example.com"]}]` }, 2, false},
		"endless IDs": {func(n int) string { return fmt.Sprintf(`[{"id":"%d","dns_names":["a.example.com"]}]`, n) }, certSpotterMaxPages, true},
	} {
		t.Run(name, func(t *testing.T) {
			var requests int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Write([]byte(tc.page(requests)))
			}))
			defer srv.Close()

			issuances, err := CertSpotter{Config{BaseURL: srv.URL}}.Search(context.Background(), Query{Domain: "example.com."})
			if truncated := errors.Is(err, ErrTruncated); truncated != tc.truncated || (err != nil && !truncated) {
				t.Fatalf("Search: %v; want truncated %v", err, tc.truncated)
			}
			if requests != tc.want || len(issuances) != tc.want {
				t.Errorf("%d requests and %d issuances; want %d of each", requests, len(issuances), tc.want)
			}
		})
	}
}

func TestCertSpotterNoAPIKey(t *testing.T) {
	var requests []*http.Request
	srv := certSpotterServer(t, &requests)
	defer srv.Close()

	if _, err := (CertSpotter{Config{BaseURL: srv.URL}}).Search(context.Background(), Query{Domain: "example.com"}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	if auth := requests[0].Header.Get("Authorization"); auth != "" {
		t.Errorf("Authorization = %q; want none without API key", auth)
	}
	if q := requests[0].URL.Query(); q.Has("include_subdomains") {
		t.Errorf("query = %v; want the domain alone", q)
	}
}

func TestCertSpotterErrors(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		body, _ := os.ReadFile("testdata/certspotter_page1.json")
		w.Write(body)
	}))
	defer srv.Close()

	issuances, err := CertSpotter{Config{BaseURL: srv.URL}}.Search(context.Background(), Query{Domain: "example.com"})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v; want ErrRateLimited", err)
	}
	if len(issuances) != 2 {
		t.Errorf("%d issuances; want those of the first page, despite the error", len(issuances))
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"not_an_array"}`))
	}))
	defer bad.Close()

	if _, err := (CertSpotter{Config{BaseURL: bad.URL}}).Search(context.Background(), Query{Domain: "example.com"}); err == nil {
		t.Error("an answer that is not a list of issuances is accepted")
	}
}

func TestCertSpotterQuery(t *testing.T) {
	var requests []*http.Request
	srv := certSpotterServer(t, &requests)
	defer srv.Close()

	// Cert Spotter only knows the valid certificates: ExcludeExpired needs no parameter.
	if _, err := (CertSpotter{Config{BaseURL: srv.URL}}).Search(context.Background(), Query{Domain: "example.com", ExcludeExpired: true}); err != nil {
		t.Fatalf("Search: %v", err)
	}

	q := requests[0].URL.Query()
	if q.Get("domain") != "example.com" || q.Has("include_subdomains") || len(q) != 2 {
		t.Errorf("query = %v; want the domain and the expansions only", q)
	}
}
