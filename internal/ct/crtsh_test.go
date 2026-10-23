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
	"os"
	"slices"
	"testing"
	"time"
)

func TestCrtShSearch(t *testing.T) {
	var request *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		body, err := os.ReadFile("testdata/crtsh.json")
		if err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
	defer srv.Close()

	issuances, err := CrtSh{Config{BaseURL: srv.URL}}.Search(context.Background(), Query{
		Domain:            "example.com.",
		IncludeSubdomains: true,
		ExcludeExpired:    true,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	q := request.URL.Query()
	if request.URL.Path != "/" || q.Get("q") != "%.example.com" || q.Get("output") != "json" || q.Get("exclude") != "expired" || q.Get("deduplicate") != "Y" {
		t.Errorf("request = %s; want the subdomains of example.com, as JSON, unexpired and deduplicated", request.URL)
	}

	if len(issuances) != 2 {
		t.Fatalf("%d issuances; want 2", len(issuances))
	}
	first := issuances[0]
	if first.ID != "14829316542" || first.Issuer != "C=US, O=Let's Encrypt, CN=R11" || first.Serial != "04a1b2c3d4e5f60718293a4b5c6d7e8f9a0b" ||
		!slices.Equal(first.DNSNames, []string{"example.com", "www.example.com"}) ||
		!first.NotBefore.Equal(time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)) || !first.NotAfter.Equal(time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("first issuance = %+v", first)
	}

	want := []string{"dev.example.com", "example.com", "mail.example.com", "www.example.com"}
	if got := Names(issuances, "example.com"); !slices.Equal(got, want) {
		t.Errorf("names = %v; want %v", got, want)
	}
}

func TestCrtShRefusesPatterns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("crt.sh asked for %s", r.URL)
	}))
	defer srv.Close()

	for _, domain := range []string{"%.com.", "ex_mple.com."} {
		if _, err := (CrtSh{Config{BaseURL: srv.URL}}).Search(context.Background(), Query{Domain: domain}); err == nil {
			t.Errorf("searching %q succeeded; want it refused", domain)
		}
	}
}

func TestCrtShUnreadableDates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"name_value":"www.example.com","not_before":"yesterday","not_after":""}]`))
	}))
	defer srv.Close()

	issuances, err := CrtSh{Config{BaseURL: srv.URL}}.Search(context.Background(), Query{Domain: "example.com."})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(issuances) != 1 || !issuances[0].NotBefore.IsZero() || !issuances[0].NotAfter.IsZero() || !slices.Equal(issuances[0].DNSNames, []string{"www.example.com"}) {
		t.Errorf("issuances = %+v; want the names kept, with zero dates", issuances)
	}
}

func TestCrtShQuery(t *testing.T) {
	var request *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	issuances, err := CrtSh{Config{BaseURL: srv.URL}}.Search(context.Background(), Query{Domain: "example.com"})
	if err != nil || len(issuances) != 0 {
		t.Fatalf("Search = %v, %v; want nothing", issuances, err)
	}
	if q := request.URL.Query(); q.Get("q") != "example.com" || q.Has("exclude") {
		t.Errorf("query = %v; want the domain alone, expired certificates included", q)
	}
}

func TestCrtShErrors(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer down.Close()

	if _, err := (CrtSh{Config{BaseURL: down.URL}}).Search(context.Background(), Query{Domain: "example.com"}); !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v; want ErrUnavailable", err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>Sorry, something went wrong</html>"))
	}))
	defer bad.Close()

	if _, err := (CrtSh{Config{BaseURL: bad.URL}}).Search(context.Background(), Query{Domain: "example.com"}); err == nil {
		t.Error("an answer that is not JSON is accepted")
	}
}
