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

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"git.happydns.org/happyDomain/internal/api/middleware"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	sessionUC "git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

func keystoreManager(t *testing.T) *secret.Manager {
	t.Helper()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithUpdatableStore(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db})))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// keystoreRouter serves the keystores of m to user: two users given the same
// m share its storage.
func keystoreRouter(m *secret.Manager, user *happydns.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	kc := NewKeystoreController(usecase.NewKeystoreUsecase(m, nil))

	r := gin.New()
	g := r.Group("", func(c *gin.Context) { c.Set("LoggedUser", user) })
	g.GET("/keystores", kc.ListKeystores)
	g.POST("/keystores", kc.CreateKeystore)
	g.GET("/keystores/kinds", kc.ListKeystoreKinds)
	g.PUT("/keystores/default", kc.SetDefaultKeystore)
	g.PUT("/keystores/:ref", kc.UpdateKeystore)
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
	m := keystoreManager(t)
	r := keystoreRouter(m, &happydns.User{Id: happydns.Identifier{0x01}})

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

	if w := do(r, http.MethodPut, "/keystores/"+created.Ref, `{"params":{"passphrase":"q"}}`); w.Code != http.StatusBadRequest {
		t.Errorf("PUT a kind not updated = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPut, "/keystores/nope", `{"params":{}}`); w.Code != http.StatusNotFound {
		t.Errorf("PUT unknown = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPut, "/keystores/"+secret.KindInstance, `{"params":{}}`); w.Code != http.StatusForbidden {
		t.Errorf("PUT instance = %d %s", w.Code, w.Body.String())
	}

	w = do(r, http.MethodPost, "/keystores", `{"kind":"`+secrettest.UpdatableKind+`","name":"Upd","params":{"passphrase":"p"}}`)
	var upd secret.Keystore
	if err := json.Unmarshal(w.Body.Bytes(), &upd); w.Code != http.StatusCreated || err != nil {
		t.Fatalf("POST updatable = %d %s", w.Code, w.Body.String())
	}
	w = do(r, http.MethodPut, "/keystores/"+upd.Ref, `{"params":{"passphrase":"s3cr3t-new"}}`)
	if w.Code != http.StatusOK {
		t.Errorf("PUT = %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "s3cr3t-new") || strings.Contains(w.Body.String(), "keyring") {
		t.Errorf("the update answer leaks what the store keeps: %s", w.Body.String())
	}
	if w := do(r, http.MethodPut, "/keystores/"+upd.Ref, `{"params":{"passphrase":""}}`); w.Code != http.StatusBadRequest {
		t.Errorf("PUT refused by the store = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPut, "/keystores/"+upd.Ref, `not json`); w.Code != http.StatusBadRequest {
		t.Errorf("PUT malformed = %d %s", w.Code, w.Body.String())
	}

	// Another user, on the same storage, reaches none of it.
	other := keystoreRouter(m, &happydns.User{Id: happydns.Identifier{0x02}})
	if w := do(other, http.MethodPut, "/keystores/"+upd.Ref, `{"params":{"passphrase":"x"}}`); w.Code != http.StatusNotFound {
		t.Errorf("PUT by another user = %d %s", w.Code, w.Body.String())
	}
	if w := do(other, http.MethodDelete, "/keystores/"+created.Ref, ""); w.Code != http.StatusNotFound {
		t.Errorf("DELETE by another user = %d %s", w.Code, w.Body.String())
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

}

func TestKeystoreUnlockRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithInteractiveStores(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	kc := NewKeystoreController(usecase.NewKeystoreUsecase(m, nil))
	user := &happydns.User{Id: happydns.Identifier{0x01}}

	r := gin.New()
	g := r.Group("", func(c *gin.Context) {
		c.Set("LoggedUser", user)
		c.Request = c.Request.WithContext(secret.WithUnlockScope(c.Request.Context(), "session"))
	})
	g.GET("/keystores", kc.ListKeystores)
	g.POST("/keystores", kc.CreateKeystore)
	g.POST("/keystores/:ref/unlock", kc.UnlockKeystore)
	g.POST("/keystores/:ref/lock", kc.LockKeystore)

	w := do(r, http.MethodPost, "/keystores", `{"kind":"`+secrettest.PassphraseKind+`","name":"Mine","params":{"passphrase":"p"}}`)
	var created secret.Keystore
	if err := json.Unmarshal(w.Body.Bytes(), &created); w.Code != http.StatusCreated || err != nil {
		t.Fatalf("POST = %d %s", w.Code, w.Body.String())
	}

	if w := do(r, http.MethodPost, "/keystores/"+created.Ref+"/unlock", `{"material":{"passphrase":"wrong"}}`); w.Code != http.StatusForbidden {
		t.Errorf("unlock(wrong) = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/keystores/"+created.Ref+"/unlock", `{"material":{"passphrase":"p"}}`); w.Code != http.StatusNoContent {
		t.Errorf("unlock = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodGet, "/keystores", ""); !strings.Contains(w.Body.String(), `"locked":false`) {
		t.Errorf("GET after unlock = %s", w.Body.String())
	}

	// Another user, in their own session, can neither unlock nor lock it.
	og := r.Group("/as-other", func(c *gin.Context) {
		c.Set("LoggedUser", &happydns.User{Id: happydns.Identifier{0x02}})
		c.Request = c.Request.WithContext(secret.WithUnlockScope(c.Request.Context(), "other"))
	})
	og.POST("/keystores/:ref/unlock", kc.UnlockKeystore)
	og.POST("/keystores/:ref/lock", kc.LockKeystore)
	if w := do(r, http.MethodPost, "/as-other/keystores/"+created.Ref+"/unlock", `{"material":{"passphrase":"p"}}`); w.Code != http.StatusNotFound {
		t.Errorf("unlock by another user = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/as-other/keystores/"+created.Ref+"/lock", ""); w.Code != http.StatusNotFound {
		t.Errorf("lock by another user = %d %s", w.Code, w.Body.String())
	}

	if w := do(r, http.MethodPost, "/keystores/nope/unlock", `{"material":{}}`); w.Code != http.StatusNotFound {
		t.Errorf("unlock unknown = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/keystores/"+secret.KindInstance+"/unlock", `{"material":{}}`); w.Code != http.StatusBadRequest {
		t.Errorf("unlock a keystore not needing it = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/keystores/"+created.Ref+"/unlock", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("unlock without material = %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPost, "/keystores/"+created.Ref+"/lock", ""); w.Code != http.StatusNoContent {
		t.Errorf("lock = %d %s", w.Code, w.Body.String())
	}
	w = do(r, http.MethodGet, "/keystores", "")
	var list []usecase.KeystoreUsage
	json.Unmarshal(w.Body.Bytes(), &list)
	for _, ks := range list {
		if ks.Ref == created.Ref && (!ks.Locked || !ks.Interactive || len(ks.UnlockForm) != 1) {
			t.Errorf("after lock: %+v", ks)
		}
	}
}

// A keystore still locked while sealing a channel is the client's to unlock,
// not internal.
func TestSealErrorKeystoreLocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	sealError(c, happydns.KeystoreLockedError{KeystoreLock: happydns.KeystoreLock{Ref: "abc"}})
	if w.Code != http.StatusLocked {
		t.Errorf("locked keystore = %d, want 423", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	sealError(c, happydns.ErrKeystoreLocked)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bare ErrKeystoreLocked = %d, want 400", w.Code)
	}
}

// A logout closes the keystores unlocked during the session.
func TestLogoutLocksSessionKeystores(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithInteractiveStores(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	keystores := usecase.NewKeystoreUsecase(m, nil)
	kc := NewKeystoreController(keystores)
	lc := NewLoginController(nil, nil, nil).WithKeystores(keystores)

	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "user@example.com"}
	sess, err := sessionUC.NewService(db).CreateUserSession(user, "script")
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(sessions.Sessions(session.COOKIE_NAME, session.NewSessionStore(&happydns.Options{}, db, securecookie.GenerateRandomKey(32))))
	r.Use(middleware.UnlockScope(), func(c *gin.Context) { c.Set("LoggedUser", user) })
	r.POST("/keystores", kc.CreateKeystore)
	r.POST("/keystores/:ref/unlock", kc.UnlockKeystore)
	r.POST("/auth/logout", lc.Logout)

	call := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+sess.Token)
		r.ServeHTTP(w, req)
		return w
	}

	w := call(http.MethodPost, "/keystores", `{"kind":"`+secrettest.PassphraseKind+`","name":"Mine","params":{"passphrase":"p"}}`)
	var created secret.Keystore
	if err := json.Unmarshal(w.Body.Bytes(), &created); w.Code != http.StatusCreated || err != nil {
		t.Fatalf("POST = %d %s", w.Code, w.Body.String())
	}
	if w := call(http.MethodPost, "/keystores/"+created.Ref+"/unlock", `{"material":{"passphrase":"p"}}`); w.Code != http.StatusNoContent {
		t.Fatalf("unlock = %d %s", w.Code, w.Body.String())
	}

	scoped := secret.WithUnlockScope(context.Background(), sess.Id)
	locked := func() bool {
		list, err := keystores.List(scoped, user.Id)
		if err != nil {
			t.Fatal(err)
		}
		for _, ks := range list {
			if ks.Ref == created.Ref {
				return ks.Locked
			}
		}
		t.Fatalf("keystore %s not listed", created.Ref)
		return false
	}
	if locked() {
		t.Fatal("the keystore is not unlocked under the session")
	}
	if w := call(http.MethodPost, "/auth/logout", ""); w.Code != http.StatusNoContent {
		t.Fatalf("logout = %d %s", w.Code, w.Body.String())
	}
	if !locked() {
		t.Error("a keystore stays unlocked after logout")
	}
}
