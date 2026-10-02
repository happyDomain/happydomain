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
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/model"
)

// creatingProviders records the provider it is asked to create.
type creatingProviders struct {
	happydns.ProviderUsecase
	got *happydns.ProviderMessage
}

func (p *creatingProviders) CreateProvider(_ context.Context, user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	p.got = msg
	created := &happydns.Provider{ProviderMeta: msg.ProviderMeta}
	created.Id = happydns.Identifier{0x42}
	created.Owner = user.Id
	return created, nil
}

func providerRequest(t *testing.T, method, body string, user *happydns.User) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/providers", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", user)
	return w, c
}

// The server chooses the identifier and the owner of a new provider: a
// client cannot give them, and is not asked to.
func TestAddProviderWithoutIdentifiers(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	providers := &creatingProviders{}
	pc := NewProviderController(providers, true)

	w, c := providerRequest(t, http.MethodPost, `{"_srctype":"DDNSServer","_comment":"mine","Provider":{"keyname":"ddns"}}`, user)
	pc.AddProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("AddProvider = %d %s, want 200", w.Code, w.Body.String())
	}

	if providers.got == nil {
		t.Fatal("CreateProvider was not called")
	}
	if providers.got.Type != "DDNSServer" || providers.got.Comment != "mine" || string(providers.got.Provider) != `{"keyname":"ddns"}` {
		t.Errorf("CreateProvider got %+v, %s", providers.got.ProviderMeta, providers.got.Provider)
	}
}

// Identifiers a client sends anyway are not passed along: only the server
// chooses them.
func TestAddProviderIgnoresClientIdentifiers(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	providers := &creatingProviders{}
	pc := NewProviderController(providers, true)

	w, c := providerRequest(t, http.MethodPost, `{"_srctype":"DDNSServer","_id":"Ag","_ownerid":"Aw","Provider":{}}`, user)
	pc.AddProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("AddProvider = %d %s, want 200", w.Code, w.Body.String())
	}
	if len(providers.got.Id) != 0 || len(providers.got.Owner) != 0 {
		t.Errorf("CreateProvider got id %v, owner %v; want neither", providers.got.Id, providers.got.Owner)
	}
}

func TestAddProviderRequiresType(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	providers := &creatingProviders{}
	pc := NewProviderController(providers, true)

	w, c := providerRequest(t, http.MethodPost, `{"_comment":"mine","Provider":{}}`, user)
	pc.AddProvider(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("AddProvider = %d %s, want 400", w.Code, w.Body.String())
	}
	if providers.got != nil {
		t.Error("CreateProvider was called without a type")
	}
}

// An update names its provider in the path: the body does not repeat its
// identifier and owner.
func TestUpdateProviderWithoutIdentifiers(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	old := &happydns.Provider{ProviderMeta: happydns.ProviderMeta{Id: happydns.Identifier{0x02}, Owner: user.Id, Type: "DDNSServer", Comment: "before"}}
	pc := NewProviderController(&updatingProviders{}, false)

	w, c := providerRequest(t, http.MethodPut, `{"_srctype":"DDNSServer","_comment":"after","Provider":{}}`, user)
	c.Set("provider", old)
	pc.UpdateProvider(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateProvider = %d %s, want 200", w.Code, w.Body.String())
	}

	var got happydns.ProviderMeta
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Comment != "after" || !got.Id.Equals(old.Id) {
		t.Errorf("answered %+v, want the updated provider %s", got, old.Id.String())
	}
}
