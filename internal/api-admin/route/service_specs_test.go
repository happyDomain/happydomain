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
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/usecase"
	happydns "git.happydns.org/happyDomain/model"
)

// The service specs stay public behind an admin password, as they are on the
// API of the users, while the other routes need a session.
func TestServiceSpecsNeedNoAdminSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &happydns.Options{AdminPasswordHash: "set", JWTSecretKey: []byte("secret")}
	router := gin.New()
	DeclareRoutes(cfg, router, Dependencies{ServiceSpecs: usecase.NewServiceSpecsUsecase()})

	for path, want := range map[string]int{
		"/api/service_specs": http.StatusOK,
		"/api/version":       http.StatusUnauthorized,
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != want {
			t.Errorf("GET %s without admin session: status %d, want %d", path, w.Code, want)
		}
	}
}
