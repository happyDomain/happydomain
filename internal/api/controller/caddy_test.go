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
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

// prefixValidator claims every FQDN carrying the given prefix.
type prefixValidator struct {
	prefix string
	err    error
}

func (v prefixValidator) IsManaged(fqdn string) (bool, error) {
	if v.err != nil {
		return false, v.err
	}
	return len(fqdn) > len(v.prefix) && fqdn[:len(v.prefix)] == v.prefix, nil
}

func askStatus(t *testing.T, ctrl *CaddyController, query string) int {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/caddy/ask"+query, nil)

	ctrl.Ask(c)

	return w.Code
}

func TestCaddyAsk(t *testing.T) {
	ctrl := NewCaddyController(
		prefixValidator{prefix: "autoconfig."},
		prefixValidator{prefix: "mta-sts."},
	)

	tests := []struct {
		query string
		want  int
	}{
		{"?domain=autoconfig.example.com", http.StatusOK},
		// The second validator must be consulted once the first declines.
		{"?domain=mta-sts.example.com", http.StatusOK},
		{"?domain=example.com", http.StatusNotFound},
		{"?domain=", http.StatusBadRequest},
		{"", http.StatusBadRequest},
	}

	for _, tt := range tests {
		if got := askStatus(t, ctrl, tt.query); got != tt.want {
			t.Errorf("Ask(%q) = %d; want %d", tt.query, got, tt.want)
		}
	}
}

func TestCaddyAsk_ValidatorError(t *testing.T) {
	ctrl := NewCaddyController(prefixValidator{prefix: "mta-sts.", err: errors.New("storage is down")})

	if got := askStatus(t, ctrl, "?domain=mta-sts.example.com"); got != http.StatusInternalServerError {
		t.Errorf("Ask() = %d; want %d", got, http.StatusInternalServerError)
	}
}

// Optional usecases are passed straight through, so nil ones must not make the
// endpoint answer (nor panic).
func TestCaddyAsk_NoValidators(t *testing.T) {
	var missing happydns.HostedDomainValidator

	ctrl := NewCaddyController(missing)
	if ctrl.HasValidators() {
		t.Fatal("HasValidators() = true; want false")
	}

	if got := askStatus(t, ctrl, "?domain=mta-sts.example.com"); got != http.StatusNotFound {
		t.Errorf("Ask() = %d; want %d", got, http.StatusNotFound)
	}
}
