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

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

func keystoreRouter(t *testing.T, user *happydns.User) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	kc := NewKeystoreController(usecase.NewKeystoreUsecase(m, nil))

	r := gin.New()
	g := r.Group("", func(c *gin.Context) { c.Set("LoggedUser", user) })
	g.GET("/keystores", kc.ListKeystores)
	g.POST("/keystores", kc.CreateKeystore)
	g.GET("/keystores/kinds", kc.ListKeystoreKinds)
	g.PUT("/keystores/default", kc.SetDefaultKeystore)
	g.DELETE("/keystores/:ref", kc.DeleteKeystore)
	g.POST("/keystores/:ref/migrate", kc.MigrateKeystore)
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestKeystoreRoutes(t *testing.T) {
	r := keystoreRouter(t, &happydns.User{Id: happydns.Identifier{0x01}})

	if w := do(r, http.MethodGet, "/keystores/kinds", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), secrettest.PersonalKind) {
		t.Errorf("GET kinds = %d %s", w.Code, w.Body.String())
	}

	w := do(r, http.MethodPost, "/keystores", `{"kind":"`+secrettest.PersonalKind+`","name":"Mine","params":{"passphrase":"p"}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST = %d %s", w.Code, w.Body.String())
	}
	var created secret.Keystore
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Name != "Mine" || created.Scope != secret.ScopePersonal {
		t.Fatalf("created = %+v, %v", created, err)
	}
	if strings.Contains(w.Body.String(), "passphrase") || strings.Contains(w.Body.String(), "keyring") {
		t.Errorf("the creation answer leaks what the store keeps: %s", w.Body.String())
	}

	if w := do(r, http.MethodPost, "/keystores", `{"kind":"`+secrettest.PersonalKind+`","name":"Bad","params":{"passphrase":""}}`); w.Code != http.StatusBadRequest {
		t.Errorf("POST refused by the store = %d %s", w.Code, w.Body.String())
	}

	if w := do(r, http.MethodPut, "/keystores/default", `{"ref":"`+created.Ref+`"}`); w.Code != http.StatusNoContent {
		t.Errorf("PUT default = %d %s", w.Code, w.Body.String())
	}

	w = do(r, http.MethodGet, "/keystores", "")
	var list []usecase.KeystoreUsage
	if err := json.Unmarshal(w.Body.Bytes(), &list); w.Code != http.StatusOK || err != nil || len(list) != 2 {
		t.Fatalf("GET = %d %s", w.Code, w.Body.String())
	}
	for _, ks := range list {
		if (ks.Ref == created.Ref) != ks.Default {
			t.Errorf("keystore %s default = %v", ks.Ref, ks.Default)
		}
	}

	if w := do(r, http.MethodPost, "/keystores/"+created.Ref+"/migrate", `{"to":"`+secret.KindInstance+`"}`); w.Code != http.StatusOK {
		t.Errorf("POST migrate = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodDelete, "/keystores/"+secret.KindInstance, ""); w.Code != http.StatusForbidden {
		t.Errorf("DELETE instance = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodDelete, "/keystores/"+created.Ref, ""); w.Code != http.StatusNoContent {
		t.Errorf("DELETE = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodDelete, "/keystores/"+created.Ref, ""); w.Code != http.StatusNotFound {
		t.Errorf("DELETE again = %d %s", w.Code, w.Body.String())
	}

	// Another user sees none of it.
	other := keystoreRouter(t, &happydns.User{Id: happydns.Identifier{0x02}})
	if w := do(other, http.MethodDelete, "/keystores/"+created.Ref, ""); w.Code != http.StatusNotFound {
		t.Errorf("DELETE by another user = %d %s", w.Code, w.Body.String())
	}
}
