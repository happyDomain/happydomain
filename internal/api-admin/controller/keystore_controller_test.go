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

func TestAdminKeystoreRoutes(t *testing.T) {
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
	g := r.Group("/secrets/owners/:owner", kc.OwnerHandler)
	g.GET("/keystores", kc.ListKeystores)
	g.POST("/keystores", kc.CreateKeystore)
	g.PUT("/keystores/default", kc.SetDefaultKeystore)
	g.DELETE("/keystores/:ref", kc.DeleteKeystore)
	g.POST("/keystores/:ref/migrate", kc.MigrateKeystore)

	do := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	user := happydns.Identifier{0x01}
	base := "/secrets/owners/" + user.String()

	if w := do(http.MethodGet, "/secrets/owners/!!/keystores", ""); w.Code != http.StatusBadRequest {
		t.Errorf("GET with a bad owner = %d %s", w.Code, w.Body.String())
	}

	w := do(http.MethodPost, base+"/keystores", `{"kind":"`+secrettest.PersonalKind+`","name":"Theirs","params":{"passphrase":"p"}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST = %d %s", w.Code, w.Body.String())
	}
	var created secret.Keystore
	json.Unmarshal(w.Body.Bytes(), &created)

	w = do(http.MethodGet, base+"/keystores", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), created.Ref) {
		t.Errorf("GET = %d %s", w.Code, w.Body.String())
	}

	// The instance itself, owner of the options of the admin scope.
	if w := do(http.MethodGet, "/secrets/owners/instance/keystores", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"ref":"instance"`) {
		t.Errorf("GET instance = %d %s", w.Code, w.Body.String())
	}

	// Once used, the administrator deletes the instance keystore of a user.
	obj := struct {
		Token happydns.Secret `json:"token"`
	}{happydns.NewSecret("v")}
	if err := m.SealObject(t.Context(), secret.SecretContext{Owner: user, ObjectType: "x", ObjectId: "y"}, &obj); err != nil {
		t.Fatal(err)
	}
	if w := do(http.MethodDelete, base+"/keystores/"+secret.KindInstance, ""); w.Code != http.StatusNoContent {
		t.Errorf("DELETE instance = %d %s", w.Code, w.Body.String())
	}
	if w := do(http.MethodDelete, base+"/keystores/"+created.Ref, ""); w.Code != http.StatusNoContent {
		t.Errorf("DELETE personal = %d %s", w.Code, w.Body.String())
	}
}
