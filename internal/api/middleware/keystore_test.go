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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/secret"
)

func TestKeystoreChoice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var seen string
	r := gin.New()
	r.Use(KeystoreChoice())
	r.POST("/x", func(c *gin.Context) { seen = secret.KeystoreFromContext(c.Request.Context()) })

	for query, want := range map[string]string{"?keystore=abc": "abc", "": "", "?keystore=": ""} {
		seen = "unset"
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x"+query, nil))
		if seen != want {
			t.Errorf("POST /x%s: keystore = %q, want %q", query, seen, want)
		}
	}
}
