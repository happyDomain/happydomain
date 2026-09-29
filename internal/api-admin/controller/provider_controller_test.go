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

	happydns "git.happydns.org/happyDomain/model"
)

type adminSecretBody struct {
	Host   string          `json:"host"`
	ApiKey happydns.Secret `json:"apikey"`
}

func (*adminSecretBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

// The admin API redacts like the user API: a legacy plaintext value cannot be
// encoded, and the operator has no use for ciphertext either.
func TestAdminGetProviderRedactsSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body adminSecretBody
	// As read from storage: an unprefixed value is legacy plaintext.
	if err := json.Unmarshal([]byte(`{"host":"h","apikey":"legacy-plaintext"}`), &body); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/providers/x", nil)
	c.Set("provider", &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "adminSecretBody", Id: happydns.Identifier{1}, Owner: happydns.Identifier{2}},
		Provider:     &body,
	})

	NewProviderController(nil, nil).GetProvider(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "legacy-plaintext") {
		t.Errorf("admin API leaks the secret: %s", w.Body.String())
	}

	var got struct {
		Provider struct {
			Host   string `json:"host"`
			ApiKey string `json:"apikey"`
		} `json:"Provider"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Provider.ApiKey != happydns.RedactedSecret || got.Provider.Host != "h" {
		t.Errorf("Provider = %+v, want the secret redacted and the rest kept", got.Provider)
	}
}
