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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// sshStubResolver answers FetchSSHHostKeys only, the other methods of the
// embedded interface are never reached.
type sshStubResolver struct {
	happydns.ResolverUsecase
	resp *happydns.SSHHostKeysResponse
	err  error
}

func (s sshStubResolver) FetchSSHHostKeys(happydns.SSHHostKeysRequest) (*happydns.SSHHostKeysResponse, error) {
	return s.resp, s.err
}

func postSSHHostKeys(t *testing.T, uc happydns.ResolverUsecase, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/resolver/ssh-hostkeys", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	NewResolverController(uc).FetchSSHHostKeys(c)

	return w
}

func TestFetchSSHHostKeysController(t *testing.T) {
	ok := sshStubResolver{resp: &happydns.SSHHostKeysResponse{Host: "example.org", Port: 22, Status: "ok"}}

	tests := []struct {
		name string
		uc   happydns.ResolverUsecase
		body string
		want int
	}{
		{"success", ok, `{"host":"example.org"}`, http.StatusOK},
		{"malformed JSON", ok, `not json`, http.StatusBadRequest},
		{"invalid host", sshStubResolver{err: happydns.ValidationError{Msg: "host is required"}}, `{"host":""}`, http.StatusBadRequest},
		{"internal error", sshStubResolver{err: errors.New("boom")}, `{"host":"example.org"}`, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := postSSHHostKeys(t, tt.uc, tt.body); w.Code != tt.want {
				t.Errorf("status = %d, expected %d (%s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
