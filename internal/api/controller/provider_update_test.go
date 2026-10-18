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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// updatedBody holds a credential, as provider bodies do.
type updatedBody struct {
	ApiKey happydns.Secret `json:"apikey" happydomain:"secret"`
}

func (*updatedBody) InstantiateProvider() (happydns.ProviderActuator, error) { return nil, nil }

// updatingProviders gives back, for an update, the provider it would store.
type updatingProviders struct {
	happydns.ProviderUsecase
}

func (updatingProviders) ShowReferences(context.Context, *happydns.Provider) error { return nil }

func (updatingProviders) UpdateProviderFromMessage(_ context.Context, id happydns.Identifier, user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	p := &happydns.Provider{ProviderMeta: msg.ProviderMeta, Provider: &updatedBody{ApiKey: happydns.NewSecret("new-key")}}
	p.Id = id
	p.Owner = user.Id
	return p, nil
}

// The answer to an update is the provider as written, its credentials
// withheld: a client keeping it would otherwise bring the previous values back
// on its next save.
func TestUpdateProviderAnswersTheUpdatedProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	old := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{0x02}, Owner: user.Id, Type: "DDNSServer", Comment: "before"}}
	pc := NewProviderController(updatingProviders{})

	update := old.ProviderMeta
	update.Comment = "after"
	body, err := json.Marshal(&happydns.ProviderMessage{ProviderMeta: update, Provider: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/providers/x", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", user)
	c.Set("provider", old)

	pc.UpdateProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProvider = %d %s", w.Code, w.Body.String())
	}

	var got struct {
		happydns.ProviderMeta
		Provider struct {
			ApiKey string `json:"apikey"`
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Comment != "after" {
		t.Errorf("answered comment = %q, want the updated one", got.Comment)
	}
	if got.Provider.ApiKey != happydns.RedactedSecret || strings.Contains(w.Body.String(), "new-key") {
		t.Errorf("answer = %s, want the credential withheld", w.Body.String())
	}
}

// referencingProviders shows the credential of a provider as the reference
// its keystore keeps.
type referencingProviders struct {
	happydns.ProviderUsecase
}

func (referencingProviders) ShowReferences(_ context.Context, p *happydns.Provider) error {
	p.Provider.(*updatedBody).ApiKey.SetReference("ovh/key")
	return nil
}

// A reference is shown to the owner of the provider: it is not secret.
func TestGetProviderShowsReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	var sealed happydns.Secret
	if err := json.Unmarshal([]byte(`"hds:1:AQ:b3ZoL2tleQ"`), &sealed); err != nil {
		t.Fatal(err)
	}
	p := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{0x02}, Owner: user.Id}, Provider: &updatedBody{ApiKey: sealed}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/providers/x", nil)
	c.Set("LoggedUser", user)
	c.Set("provider", p)

	NewProviderController(referencingProviders{}).GetProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GetProvider = %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"apikey":"ovh/key"`) {
		t.Errorf("answer = %s, want the reference", w.Body.String())
	}
}

// failingReferences cannot reach the keystore of the provider.
type failingReferences struct {
	happydns.ProviderUsecase
}

func (failingReferences) ShowReferences(context.Context, *happydns.Provider) error {
	return errors.New("keystore unreachable")
}

// A keystore that cannot show its references still gets the credentials
// withheld: the answer goes on, without them.
func TestGetProviderWithholdsWhenReferencesFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	p := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{0x02}, Owner: user.Id}, Provider: &updatedBody{ApiKey: happydns.NewSecret("stored-key")}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/providers/x", nil)
	c.Set("LoggedUser", user)
	c.Set("provider", p)

	NewProviderController(failingReferences{}).GetProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GetProvider = %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"apikey":"`+happydns.RedactedSecret+`"`) || strings.Contains(w.Body.String(), "stored-key") {
		t.Errorf("answer = %s, want the credential withheld", w.Body.String())
	}
}

// finishedWizard ends the wizard on the provider it holds.
type finishedWizard struct {
	p *happydns.Provider
}

func (w finishedWizard) NextProviderSettingsState(context.Context, *happydns.ProviderSettingsState, string, *happydns.User) (*happydns.Provider, *happydns.ProviderSettingsResponse, error) {
	return w.p, nil, nil
}

// The provider the wizard wrote is answered as GET answers it: references
// shown, credentials withheld.
func TestNextProviderSettingsStateShowsReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &happydns.User{Id: happydns.Identifier{0x01}}

	for name, tc := range map[string]struct {
		providers happydns.ProviderUsecase
		want      string
	}{
		"reference":  {referencingProviders{}, `"apikey":"ovh/key"`},
		"credential": {updatingProviders{}, `"apikey":"` + happydns.RedactedSecret + `"`},
	} {
		t.Run(name, func(t *testing.T) {
			p := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{0x02}, Owner: user.Id}, Provider: &updatedBody{ApiKey: happydns.NewSecret("stored-key")}}
			psc := NewProviderSettingsController(finishedWizard{p}, tc.providers)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/providers/_specs/x/settings", bytes.NewBufferString(`{}`))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("LoggedUser", user)
			c.Set("providerspecs", happydns.ProviderBody(&updatedBody{}))
			c.Set("providertype", "x")

			psc.NextProviderSettingsState(c)
			if w.Code != http.StatusOK {
				t.Fatalf("NextProviderSettingsState = %d %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "stored-key") {
				t.Errorf("answer = %s, want %s", w.Body.String(), tc.want)
			}
		})
	}
}
