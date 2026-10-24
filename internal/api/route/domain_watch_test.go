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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// notFoundWatchlist knows no watch.
type notFoundWatchlist struct{}

func (notFoundWatchlist) ListDomainWatches(*happydns.User) ([]*happydns.DomainWatch, error) {
	return []*happydns.DomainWatch{}, nil
}
func (notFoundWatchlist) GetDomainWatch(*happydns.User, happydns.Identifier) (*happydns.DomainWatch, error) {
	return nil, happydns.ErrDomainWatchNotFound
}
func (notFoundWatchlist) CreateDomainWatch(context.Context, *happydns.User, happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	return nil, happydns.ErrDomainWatchNotFound
}
func (notFoundWatchlist) UpdateDomainWatch(*happydns.User, happydns.Identifier, happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	return nil, happydns.ErrDomainWatchNotFound
}
func (notFoundWatchlist) DeleteDomainWatch(*happydns.User, happydns.Identifier) error {
	return happydns.ErrDomainWatchNotFound
}
func (notFoundWatchlist) CheckDomainWatch(context.Context, *happydns.User, happydns.Identifier) (*happydns.DomainWatch, error) {
	return nil, happydns.ErrDomainWatchNotFound
}

// Only the routes querying the registries while the user waits go through the
// lookup rate limiter.
func TestDomainWatchRoutesLimitTheLookups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	auth := router.Group("/", func(c *gin.Context) {
		c.Set("LoggedUser", &happydns.User{Id: happydns.Identifier{0x01}})
	})
	limited := func(c *gin.Context) {
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
	DeclareDomainWatchRoutes(auth, limited, notFoundWatchlist{})

	watch := happydns.Identifier{0x02}
	id := watch.String()
	for _, tc := range []struct {
		method, path string
		limited      bool
	}{
		{http.MethodGet, "/watchlist", false},
		{http.MethodPost, "/watchlist", true},
		{http.MethodGet, "/watchlist/" + id, false},
		{http.MethodPut, "/watchlist/" + id, false},
		{http.MethodDelete, "/watchlist/" + id, false},
		{http.MethodPost, "/watchlist/" + id + "/check", true},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{"domain":"example.com","note":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if got := w.Code == http.StatusTooManyRequests; got != tc.limited {
			t.Errorf("%s %s: status %d, limited = %v, want %v", tc.method, tc.path, w.Code, got, tc.limited)
		}
	}
}
