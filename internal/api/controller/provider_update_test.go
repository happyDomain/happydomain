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
