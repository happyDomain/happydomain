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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// stubWatchlist answers every call with err, or with a watch, and remembers
// the users it was called for when users is set.
type stubWatchlist struct {
	err   error
	users *[]*happydns.User
}

func (s stubWatchlist) seen(u *happydns.User) {
	if s.users != nil {
		*s.users = append(*s.users, u)
	}
}

func (s stubWatchlist) answer() (*happydns.DomainWatch, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &happydns.DomainWatch{DomainName: "example.com", Phase: happydns.DomainWatchRegistered}, nil
}

func (s stubWatchlist) ListDomainWatches(u *happydns.User) ([]*happydns.DomainWatch, error) {
	s.seen(u)
	w, err := s.answer()
	if err != nil {
		return nil, err
	}
	return []*happydns.DomainWatch{w}, nil
}
func (s stubWatchlist) GetDomainWatch(u *happydns.User, _ happydns.Identifier) (*happydns.DomainWatch, error) {
	s.seen(u)
	return s.answer()
}
func (s stubWatchlist) CreateDomainWatch(_ context.Context, u *happydns.User, _ happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	s.seen(u)
	return s.answer()
}
func (s stubWatchlist) UpdateDomainWatch(u *happydns.User, _ happydns.Identifier, _ happydns.DomainWatchInput) (*happydns.DomainWatch, error) {
	s.seen(u)
	return s.answer()
}
func (s stubWatchlist) DeleteDomainWatch(u *happydns.User, _ happydns.Identifier) error {
	s.seen(u)
	return s.err
}
func (s stubWatchlist) CheckDomainWatch(_ context.Context, u *happydns.User, _ happydns.Identifier) (*happydns.DomainWatch, error) {
	s.seen(u)
	return s.answer()
}

var loggedUser = &happydns.User{Id: happydns.Identifier{0x01}}

func callWatchlist(uc happydns.DomainWatchUsecase, method, watchId, body string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/watchlist", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", loggedUser)
	if watchId != "" {
		c.Params = gin.Params{{Key: "watchId", Value: watchId}}
	}

	wc := NewDomainWatchController(uc)
	switch {
	case method == http.MethodGet && watchId == "":
		wc.ListDomainWatches(c)
	case method == http.MethodPost && watchId == "":
		wc.AddDomainWatch(c)
	case method == http.MethodGet:
		wc.GetDomainWatch(c)
	case method == http.MethodPut:
		wc.UpdateDomainWatch(c)
	case method == http.MethodDelete:
		wc.DeleteDomainWatch(c)
	default:
		wc.CheckDomainWatch(c)
	}
	// As gin does at the end of a request, for a status without body.
	c.Writer.WriteHeaderNow()
	return w
}

func TestDomainWatchController(t *testing.T) {
	watch := happydns.Identifier{0x02}
	id := watch.String()
	ok := stubWatchlist{}
	failing := stubWatchlist{err: happydns.InternalError{Err: errors.New("storage down")}}

	tests := []struct {
		name    string
		uc      happydns.DomainWatchUsecase
		method  string
		watchId string
		body    string
		want    int
	}{
		{"list", ok, http.MethodGet, "", "", http.StatusOK},
		{"list failing", failing, http.MethodGet, "", "", http.StatusInternalServerError},
		{"add", ok, http.MethodPost, "", `{"domain":"example.com"}`, http.StatusCreated},
		{"add malformed", ok, http.MethodPost, "", `nope`, http.StatusBadRequest},
		{"add invalid domain", stubWatchlist{err: happydns.ValidationError{Msg: "invalid"}}, http.MethodPost, "", `{"domain":"co.uk"}`, http.StatusBadRequest},
		{"add duplicate", stubWatchlist{err: happydns.CustomError{Err: errors.New("already"), Status: http.StatusConflict}}, http.MethodPost, "", `{"domain":"example.com"}`, http.StatusConflict},
		{"add failing", failing, http.MethodPost, "", `{"domain":"example.com"}`, http.StatusInternalServerError},
		{"get", ok, http.MethodGet, id, "", http.StatusOK},
		{"get invalid id", ok, http.MethodGet, "!!", "", http.StatusBadRequest},
		{"get not found", stubWatchlist{err: fmt.Errorf("wrapped: %w", happydns.ErrDomainWatchNotFound)}, http.MethodGet, id, "", http.StatusNotFound},
		{"update", ok, http.MethodPut, id, `{"note":"hello"}`, http.StatusOK},
		{"update too long a note", stubWatchlist{err: happydns.ValidationError{Msg: "too long"}}, http.MethodPut, id, `{"note":"hello"}`, http.StatusBadRequest},
		{"update malformed", ok, http.MethodPut, id, `nope`, http.StatusBadRequest},
		{"update invalid id", ok, http.MethodPut, "!!", `{"note":"hello"}`, http.StatusBadRequest},
		{"delete", ok, http.MethodDelete, id, "", http.StatusNoContent},
		{"delete not found", stubWatchlist{err: happydns.ErrDomainWatchNotFound}, http.MethodDelete, id, "", http.StatusNotFound},
		{"delete invalid id", ok, http.MethodDelete, "!!", "", http.StatusBadRequest},
		{"check", ok, http.MethodPost, id, "", http.StatusOK},
		{"check not found", stubWatchlist{err: happydns.ErrDomainWatchNotFound}, http.MethodPost, id, "", http.StatusNotFound},
		{"check failing", failing, http.MethodPost, id, "", http.StatusInternalServerError},
		{"check invalid id", ok, http.MethodPost, "!!", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := callWatchlist(tt.uc, tt.method, tt.watchId, tt.body); w.Code != tt.want {
				t.Errorf("status = %d, expected %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestDomainWatchControllerAnswersTheWatches(t *testing.T) {
	var list []happydns.DomainWatch
	if w := callWatchlist(stubWatchlist{}, http.MethodGet, "", ""); json.Unmarshal(w.Body.Bytes(), &list) != nil || len(list) != 1 || list[0].DomainName != "example.com" {
		t.Errorf("list body = %s, want an array of the watches", w.Body.String())
	}

	var added happydns.DomainWatch
	if w := callWatchlist(stubWatchlist{}, http.MethodPost, "", `{"domain":"example.com"}`); json.Unmarshal(w.Body.Bytes(), &added) != nil || added.DomainName != "example.com" || added.Phase != happydns.DomainWatchRegistered {
		t.Errorf("add body = %s, want the watch", w.Body.String())
	}
}

// Every call is made on behalf of the logged user: it is what keeps the
// watches of a user private.
func TestDomainWatchControllerActsForTheLoggedUser(t *testing.T) {
	watch := happydns.Identifier{0x02}
	id := watch.String()
	var users []*happydns.User
	uc := stubWatchlist{users: &users}

	for _, call := range []struct{ method, watchId, body string }{
		{http.MethodGet, "", ""},
		{http.MethodPost, "", `{"domain":"example.com"}`},
		{http.MethodGet, id, ""},
		{http.MethodPut, id, `{"note":"hello"}`},
		{http.MethodDelete, id, ""},
		{http.MethodPost, id, ""},
	} {
		callWatchlist(uc, call.method, call.watchId, call.body)
	}

	if len(users) != 6 {
		t.Fatalf("the use case was called %d times, want 6", len(users))
	}
	for i, u := range users {
		if u != loggedUser {
			t.Errorf("call %d made for %+v, want the logged user", i, u)
		}
	}
}
