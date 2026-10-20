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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

type sshStubResolver struct {
	happydns.ResolverUsecase
}

func (sshStubResolver) FetchSSHHostKeys(happydns.SSHHostKeysRequest) (*happydns.SSHHostKeysResponse, error) {
	return &happydns.SSHHostKeysResponse{Status: "ok"}, nil
}

func TestSSHHostKeysRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	DeclareResolverAuthRoutes(router.Group("/"), sshStubResolver{})

	post := func() int {
		req := httptest.NewRequest(http.MethodPost, "/resolver/ssh-hostkeys", strings.NewReader(`{"host":"example.org"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	for i := 1; i <= 10; i++ {
		if code := post(); code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, code)
		}
	}
	if code := post(); code != http.StatusTooManyRequests {
		t.Fatalf("request 11: status = %d, want 429", code)
	}
}
