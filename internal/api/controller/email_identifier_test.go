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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func postEmailIdentifier(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/email-identifier", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	NewEmailIdentifierController().ComputeEmailIdentifier(c)

	return w
}

func TestComputeEmailIdentifier(t *testing.T) {
	w := postEmailIdentifier(t, `{"username":"hugh"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, expected %d", w.Code, http.StatusOK)
	}

	var res emailIdentifierResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unable to decode response: %s", err)
	}

	// RFC 7929 sec. 4 example: hugh@example.com
	expected := "c93f1e400f26708f98cb19d936620da35eec8f72e57f9eec01c1afd6"
	if res.Identifier != expected {
		t.Errorf("identifier = %q, expected %q", res.Identifier, expected)
	}
}

func TestComputeEmailIdentifierBadRequest(t *testing.T) {
	for _, body := range []string{`{}`, `{"username":""}`, `not json`} {
		if w := postEmailIdentifier(t, body); w.Code != http.StatusBadRequest {
			t.Errorf("status for %q = %d, expected %d", body, w.Code, http.StatusBadRequest)
		}
	}
}
