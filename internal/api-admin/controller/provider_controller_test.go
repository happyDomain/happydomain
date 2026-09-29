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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	happydns "git.happydns.org/happyDomain/model"
)

// adminSecretBody holds one secret of each kind a provider can have: a
// happydns.Secret, and a plain string tagged `secret`, which is what providers
// not yet converted use.
type adminSecretBody struct {
	Host     string          `json:"host"`
	ApiKey   happydns.Secret `json:"apikey"`
	Password string          `json:"password" happydomain:"label=Password,secret"`
}

func (*adminSecretBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

const (
	adminSealedToken = "hds:1:AQIDBA:c2VhbGVk"
	adminPassword    = "tagged-plaintext"
)

var (
	adminProviderID = happydns.Identifier{1}
	adminOwnerID    = happydns.Identifier{2}
)

// newAdminProvider builds a provider as read from storage, with a fresh body on
// every call like ParseProvider does. apikey is the stored value of the
// happydns.Secret field.
func newAdminProvider(t *testing.T, comment, apikey string) *happydns.Provider {
	t.Helper()

	raw, err := json.Marshal(map[string]string{"host": "h", "apikey": apikey, "password": adminPassword})
	if err != nil {
		t.Fatal(err)
	}

	var body adminSecretBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}

	return &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "adminSecretBody", Id: adminProviderID, Owner: adminOwnerID, Comment: comment},
		Provider:     &body,
	}
}

// fakeProviderUsecase stores a single provider, as its comment and the stored
// value of its happydns.Secret field.
type fakeProviderUsecase struct {
	happydns.ProviderUsecase

	t       *testing.T
	comment string
	apikey  string
}

func (f *fakeProviderUsecase) CreateProvider(_ context.Context, _ *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	f.comment = msg.Comment
	return newAdminProvider(f.t, f.comment, f.apikey), nil
}

func (f *fakeProviderUsecase) GetUserProvider(_ context.Context, _ *happydns.User, _ happydns.Identifier) (*happydns.Provider, error) {
	return newAdminProvider(f.t, f.comment, f.apikey), nil
}

func (f *fakeProviderUsecase) UpdateProviderFromMessage(_ context.Context, _ happydns.Identifier, _ *happydns.User, msg *happydns.ProviderMessage) error {
	f.comment = msg.Comment
	return nil
}

// adminProviderResponse is a provider as the admin API answers it.
type adminProviderResponse struct {
	Comment  string `json:"_comment"`
	Provider struct {
		Host     string `json:"host"`
		ApiKey   string `json:"apikey"`
		Password string `json:"password"`
	} `json:"Provider"`
}

// checkRedacted fails unless the response carries every secret redacted, and
// nothing of their stored values.
func checkRedacted(t *testing.T, w *httptest.ResponseRecorder, stored ...string) adminProviderResponse {
	t.Helper()

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	for _, s := range stored {
		if strings.Contains(w.Body.String(), s) {
			t.Errorf("admin API leaks %q: %s", s, w.Body.String())
		}
	}

	var got adminProviderResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Provider.ApiKey != happydns.RedactedSecret || got.Provider.Password != happydns.RedactedSecret {
		t.Errorf("Provider = %+v, want every secret redacted", got.Provider)
	}
	if got.Provider.Host != "h" {
		t.Errorf("Provider.Host = %q, want the non-secret field kept", got.Provider.Host)
	}
	return got
}

// adminRequestBody is a provider as a client submits it, with every secret
// left redacted.
func adminRequestBody(t *testing.T, comment string) string {
	t.Helper()

	b, err := json.Marshal(happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{Type: "adminSecretBody", Id: adminProviderID, Owner: adminOwnerID, Comment: comment},
		Provider:     json.RawMessage(`{"host":"h","apikey":"` + happydns.RedactedSecret + `","password":"` + happydns.RedactedSecret + `"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func newAdminContext(method, body string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/users/x/providers/x", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user", &happydns.User{Id: adminOwnerID})
	return w, c
}

// The admin API redacts like the user API: a legacy plaintext value cannot be
// encoded, and the operator has no use for ciphertext either.
func TestAdminGetProviderRedactsSecrets(t *testing.T) {
	for name, apikey := range map[string]string{
		"legacy plaintext": "legacy-plaintext",
		"sealed":           adminSealedToken,
	} {
		t.Run(name, func(t *testing.T) {
			w, c := newAdminContext(http.MethodGet, "")
			c.Set("provider", newAdminProvider(t, "", apikey))

			NewProviderController(nil, nil).GetProvider(c)

			checkRedacted(t, w, apikey, adminPassword)
		})
	}
}

func TestAdminAddProviderRedactsSecrets(t *testing.T) {
	uc := &fakeProviderUsecase{t: t, apikey: adminSealedToken}

	w, c := newAdminContext(http.MethodPost, adminRequestBody(t, "new"))

	NewProviderController(uc, nil).AddProvider(c)

	checkRedacted(t, w, adminSealedToken, adminPassword)
}

// UpdateProvider answers with the provider as stored after the update, not as
// it was before, and still redacted.
func TestAdminUpdateProviderAnswersUpdatedAndRedacted(t *testing.T) {
	uc := &fakeProviderUsecase{t: t, comment: "before", apikey: adminSealedToken}

	w, c := newAdminContext(http.MethodPut, adminRequestBody(t, "after"))
	c.Set("provider", newAdminProvider(t, uc.comment, uc.apikey))

	NewProviderController(uc, nil).UpdateProvider(c)

	got := checkRedacted(t, w, adminSealedToken, adminPassword)
	if got.Comment != "after" {
		t.Errorf("_comment = %q, want the updated %q", got.Comment, "after")
	}
}
