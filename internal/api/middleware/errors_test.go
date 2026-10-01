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

package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	internal := happydns.InternalError{Err: errors.New("db password rejected"), UserMessage: "try again later"}

	tests := []struct {
		name       string
		err        error
		wantStatus int
		// wantMessage, when set, is the message expected in the response.
		wantMessage string
	}{
		{"user not found", happydns.ErrUserNotFound, http.StatusNotFound, ""},
		{"wrapped domain not found", fmt.Errorf("lookup: %w", happydns.ErrDomainNotFound), http.StatusNotFound, ""},
		{"user already exists", happydns.ErrUserAlreadyExist, http.StatusConflict, ""},
		{"validation error", happydns.ValidationError{Msg: "bad"}, http.StatusBadRequest, "bad"},
		{"unknown error uses default", errors.New("boom"), http.StatusTeapot, "boom"},

		// Wrapped on the way by a use case: still answered with their own
		// status and message.
		{"wrapped validation error", fmt.Errorf("importing: %w", happydns.ValidationError{Msg: "bad"}), http.StatusBadRequest, "bad"},
		{"wrapped http error", fmt.Errorf("importing: %w", happydns.CustomError{Err: errors.New("later"), Status: http.StatusServiceUnavailable}), http.StatusServiceUnavailable, "later"},
		{"wrapped http error without status", fmt.Errorf("importing: %w", happydns.CustomError{Err: errors.New("oops")}), http.StatusInternalServerError, "oops"},
		{"internal error", internal, http.StatusInternalServerError, "try again later"},
		{"wrapped internal error keeps its cause private", fmt.Errorf("importing: %w", internal), http.StatusInternalServerError, "try again later"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ErrorResponse(c, http.StatusTeapot, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantMessage == "" {
				return
			}
			var resp happydns.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decoding the response: %v", err)
			}
			if resp.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", resp.Message, tt.wantMessage)
			}
		})
	}
}
