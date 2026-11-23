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

package route

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type stubSuggestion struct{ asked []string }

func (s *stubSuggestion) SuggestProviders(_ context.Context, domain string) (*happydns.ProviderSuggestion, error) {
	s.asked = append(s.asked, domain)
	if domain == "" {
		return nil, happydns.ValidationError{Msg: "empty domain name"}
	}
	return &happydns.ProviderSuggestion{Nameservers: []string{}, Providers: []string{}}, nil
}

func suggestionRouter(stub *stubSuggestion) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/")
	// Declared along with the routes it has to coexist with.
	DeclareProviderRoutes(group, nil)
	DeclareProviderSuggestionRoutes(group, stub)
	return router
}

func getSuggestion(router *gin.Engine, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/providers/_suggestions"+query, nil))
	return w
}

func TestProviderSuggestion(t *testing.T) {
	stub := &stubSuggestion{}
	router := suggestionRouter(stub)

	// A classless reverse delegation holds a slash.
	const domain = "0/25.2.0.192.in-addr.arpa."
	w := getSuggestion(router, "?domain="+url.QueryEscape(domain))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if want := `{"nameservers":[],"providers":[]}`; w.Body.String() != want {
		t.Errorf("body = %s, want %s", w.Body.String(), want)
	}
	if len(stub.asked) != 1 || stub.asked[0] != domain {
		t.Errorf("asked %q, want %q", stub.asked, domain)
	}

	if w := getSuggestion(router, ""); w.Code != http.StatusBadRequest {
		t.Errorf("without a domain: status = %d, want 400", w.Code)
	}
}

func TestProviderSuggestionRateLimit(t *testing.T) {
	router := suggestionRouter(&stubSuggestion{})

	for i := 1; i <= 30; i++ {
		if w := getSuggestion(router, "?domain=example.com"); w.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, w.Code)
		}
	}
	if w := getSuggestion(router, "?domain=example.com"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("request 31: status = %d, want 429", w.Code)
	}
}
