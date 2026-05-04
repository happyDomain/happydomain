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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// stubSessionUsecase implements happydns.SessionUsecase for testing.
type stubSessionUsecase struct {
	getSession   *happydns.Session
	getErr       error
	listSessions []*happydns.Session
	listErr      error
	createResult *happydns.Session
	createErr    error
	updateErr    error
	deleteErr    error
	closeErr     error

	// Captured arguments for assertion.
	createdDescription string
	deletedSID         string
	updatedSID         string
}

func (s *stubSessionUsecase) CloseUserSessions(user *happydns.User) error {
	return s.closeErr
}
func (s *stubSessionUsecase) CreateUserSession(user *happydns.User, description string) (*happydns.Session, error) {
	s.createdDescription = description
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createResult, nil
}
func (s *stubSessionUsecase) DeleteUserSession(user *happydns.User, sid string) error {
	s.deletedSID = sid
	return s.deleteErr
}
func (s *stubSessionUsecase) GetUserSession(user *happydns.User, sid string) (*happydns.Session, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getSession, nil
}
func (s *stubSessionUsecase) ListUserSessions(user *happydns.User) ([]*happydns.Session, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listSessions, nil
}
func (s *stubSessionUsecase) UpdateUserSession(user *happydns.User, sid string, fn func(*happydns.Session)) error {
	s.updatedSID = sid
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.getSession != nil {
		fn(s.getSession)
	}
	return nil
}

// newSessionTestContext builds a gin test context with the given user and request body.
func newSessionTestContext(user *happydns.User, method, target string, body []byte, params gin.Params) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
		c.Request = httptest.NewRequest(method, target, reqBody)
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, target, nil)
	}
	if user != nil {
		c.Set("LoggedUser", user)
	}
	c.Params = params
	return w, c
}

func newTestUser(t *testing.T) *happydns.User {
	t.Helper()
	uid, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatalf("NewRandomIdentifier() error: %v", err)
	}
	return &happydns.User{Id: uid, Email: "user@example.com"}
}

// --- SessionHandler ---

func TestSessionHandler(t *testing.T) {
	user := newTestUser(t)
	sess := &happydns.Session{Id: "abc", IdUser: user.Id, Description: "desk"}

	tests := []struct {
		name       string
		user       *happydns.User
		stub       stubSessionUsecase
		wantStatus int
		wantInCtx  bool
	}{
		{
			name:       "unauthenticated",
			user:       nil,
			stub:       stubSessionUsecase{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "session not found",
			user:       user,
			stub:       stubSessionUsecase{getErr: happydns.ErrSessionNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ok",
			user:       user,
			stub:       stubSessionUsecase{getSession: sess},
			wantStatus: http.StatusOK,
			wantInCtx:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			w, c := newSessionTestContext(tt.user, "GET", "/session/abc", nil, gin.Params{{Key: "sid", Value: "abc"}})

			sc.SessionHandler(c)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantInCtx {
				if v, ok := c.Get("session"); !ok {
					t.Error("expected session in context")
				} else if got := v.(*happydns.Session); got.Id != sess.Id {
					t.Errorf("context session id = %q, want %q", got.Id, sess.Id)
				}
			}
		})
	}
}

// --- ClearUserSessions ---

func TestClearUserSessions(t *testing.T) {
	user := newTestUser(t)

	tests := []struct {
		name       string
		user       *happydns.User
		stub       stubSessionUsecase
		wantStatus int
	}{
		{"unauthenticated", nil, stubSessionUsecase{}, http.StatusBadRequest},
		{"close error", user, stubSessionUsecase{closeErr: fmt.Errorf("boom")}, http.StatusInternalServerError},
		{"success", user, stubSessionUsecase{}, http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			_, c := newSessionTestContext(tt.user, "DELETE", "/sessions", nil, nil)

			sc.ClearUserSessions(c)

			if got := c.Writer.Status(); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
		})
	}
}

// --- GetSessions ---

func TestGetSessions(t *testing.T) {
	user := newTestUser(t)
	sessions := []*happydns.Session{
		{Id: "s1", IdUser: user.Id, Description: "one"},
		{Id: "s2", IdUser: user.Id, Description: "two"},
	}

	tests := []struct {
		name       string
		user       *happydns.User
		stub       stubSessionUsecase
		wantStatus int
		wantLen    int
	}{
		{"unauthenticated", nil, stubSessionUsecase{}, http.StatusBadRequest, 0},
		{"list error", user, stubSessionUsecase{listErr: fmt.Errorf("db down")}, http.StatusInternalServerError, 0},
		{"empty", user, stubSessionUsecase{listSessions: []*happydns.Session{}}, http.StatusOK, 0},
		{"two sessions", user, stubSessionUsecase{listSessions: sessions}, http.StatusOK, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			w, c := newSessionTestContext(tt.user, "GET", "/sessions", nil, nil)

			sc.GetSessions(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				return
			}
			var got []happydns.Session
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// --- CreateSession ---

func TestCreateSession(t *testing.T) {
	user := newTestUser(t)
	created := &happydns.Session{Id: "new", IdUser: user.Id, Description: "fresh"}

	validBody, _ := json.Marshal(happydns.SessionInput{Description: "fresh"})

	tests := []struct {
		name       string
		user       *happydns.User
		body       []byte
		stub       stubSessionUsecase
		wantStatus int
		wantDesc   string
	}{
		{"invalid json", user, []byte("{not json"), stubSessionUsecase{}, http.StatusBadRequest, ""},
		{"unauthenticated", nil, validBody, stubSessionUsecase{}, http.StatusBadRequest, ""},
		{"create error", user, validBody, stubSessionUsecase{createErr: fmt.Errorf("oops")}, http.StatusInternalServerError, "fresh"},
		{"success", user, validBody, stubSessionUsecase{createResult: created}, http.StatusOK, "fresh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			w, c := newSessionTestContext(tt.user, "POST", "/sessions", tt.body, nil)

			sc.CreateSession(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantDesc != "" && stub.createdDescription != tt.wantDesc {
				t.Errorf("createdDescription = %q, want %q", stub.createdDescription, tt.wantDesc)
			}
			if tt.wantStatus == http.StatusOK {
				var got happydns.Session
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if got.Id != created.Id {
					t.Errorf("session id = %q, want %q", got.Id, created.Id)
				}
			}
		})
	}
}

// --- UpdateSession ---

func TestUpdateSession(t *testing.T) {
	user := newTestUser(t)
	existing := &happydns.Session{Id: "abc", IdUser: user.Id, Description: "old"}
	expires := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	validBody, _ := json.Marshal(happydns.SessionInput{Description: "new", ExpiresOn: expires})

	tests := []struct {
		name       string
		user       *happydns.User
		body       []byte
		stub       stubSessionUsecase
		wantStatus int
		wantSID    string
	}{
		{"invalid json", user, []byte("{"), stubSessionUsecase{}, http.StatusBadRequest, ""},
		{"unauthenticated", nil, validBody, stubSessionUsecase{}, http.StatusBadRequest, ""},
		{"update error", user, validBody, stubSessionUsecase{updateErr: fmt.Errorf("nope")}, http.StatusInternalServerError, "abc"},
		{"get-after-update error", user, validBody, stubSessionUsecase{getErr: fmt.Errorf("missing")}, http.StatusInternalServerError, "abc"},
		{"success", user, validBody, stubSessionUsecase{getSession: existing}, http.StatusOK, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			w, c := newSessionTestContext(tt.user, "PUT", "/sessions/abc", tt.body, gin.Params{{Key: "sid", Value: "abc"}})

			sc.UpdateSession(c)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantSID != "" && stub.updatedSID != tt.wantSID {
				t.Errorf("updatedSID = %q, want %q", stub.updatedSID, tt.wantSID)
			}
			if tt.wantStatus == http.StatusOK {
				if existing.Description != "new" {
					t.Errorf("existing.Description = %q, want %q (mutator not applied)", existing.Description, "new")
				}
				if !existing.ExpiresOn.Equal(expires) {
					t.Errorf("existing.ExpiresOn = %v, want %v", existing.ExpiresOn, expires)
				}
			}
		})
	}
}

// --- DeleteSession ---

func TestDeleteSession(t *testing.T) {
	user := newTestUser(t)

	tests := []struct {
		name       string
		user       *happydns.User
		stub       stubSessionUsecase
		wantStatus int
		wantSID    string
	}{
		{"unauthenticated", nil, stubSessionUsecase{}, http.StatusBadRequest, ""},
		{"delete error", user, stubSessionUsecase{deleteErr: fmt.Errorf("nope")}, http.StatusInternalServerError, "abc"},
		{"success", user, stubSessionUsecase{}, http.StatusNoContent, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := tt.stub
			sc := NewSessionController(&stub)
			_, c := newSessionTestContext(tt.user, "DELETE", "/sessions/abc", nil, gin.Params{{Key: "sid", Value: "abc"}})

			sc.DeleteSession(c)

			if got := c.Writer.Status(); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if tt.wantSID != "" && stub.deletedSID != tt.wantSID {
				t.Errorf("deletedSID = %q, want %q", stub.deletedSID, tt.wantSID)
			}
		})
	}
}

// --- ShowVersion ---

func TestShowVersion(t *testing.T) {
	prev := HDVersion
	defer func() { HDVersion = prev }()

	HDVersion = happydns.VersionResponse{Version: "1.2.3"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/version", nil)

	ShowVersion(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var got happydns.VersionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", got.Version, "1.2.3")
	}
}
