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

package route

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/internal/usecase/zonetemplate"
	happydns "git.happydns.org/happyDomain/model"
)

func newAdminTemplateRouter(t *testing.T) (*gin.Engine, storage.Storage) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	services := serviceUC.NewServiceUsecases()
	uc := zonetemplate.NewService(store, services.ValidateServiceUC, zoneUC.NewZoneUsecases(store, services), nil, store, 0)

	router := gin.New()
	declareZoneTemplateRoutes(router.Group("/api"), Dependencies{ZoneTemplate: uc})
	return router, store
}

func serve(router *gin.Engine, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// The administrators manage the global templates: those they create belong
// to no one.
func TestAdminZoneTemplatesAreGlobal(t *testing.T) {
	router, store := newAdminTemplateRouter(t)

	w := serve(router, http.MethodPost, "/api/templates", "application/json", []byte(`{"name":"Google Workspace"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/templates: status %d (%s), want 201", w.Code, w.Body.String())
	}

	var created happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if !created.IsGlobal() {
		t.Errorf("template created by an administrator = %+v, want a global one", created.ZoneTemplateMeta)
	}

	if w := serve(router, http.MethodGet, "/api/templates/"+created.Id.String(), "", nil); w.Code != http.StatusOK {
		t.Errorf("GET /api/templates/%s: status %d, want 200", created.Id.String(), w.Code)
	}

	user := happydns.Identifier{0x01}
	if err := store.CreateZoneTemplate(&happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &user, Name: "theirs"}}); err != nil {
		t.Fatal(err)
	}
	w = serve(router, http.MethodGet, "/api/templates", "", nil)
	var list []happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Errorf("GET /api/templates = %s, %v; want the global template only", w.Body.String(), err)
	}
}

// The administrators change the global templates, and cannot reach those of
// the users.
func TestAdminZoneTemplatesEdition(t *testing.T) {
	router, store := newAdminTemplateRouter(t)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("name", "Mail")
	fw, _ := mw.CreateFormFile("zone", "zone.txt")
	_, _ = fw.Write([]byte("$TTL 3600\n@ IN MX 10 mail\nmail IN A 192.0.2.2\n"))
	_ = mw.Close()
	w := serve(router, http.MethodPost, "/api/templates/import", mw.FormDataContentType(), body.Bytes())
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/templates/import: status %d (%s), want 201", w.Code, w.Body.String())
	}
	var imported happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if !imported.IsGlobal() || len(imported.Services["mail"]) != 1 {
		t.Errorf("template imported by an administrator = %+v, want a global one with mail", imported)
	}
	global := "/api/templates/" + imported.Id.String()

	if w := serve(router, http.MethodPost, global+"/records", "application/json", []byte(`["blog 3600 IN A 192.0.2.3"]`)); w.Code != http.StatusOK {
		t.Errorf("POST %s/records: status %d (%s), want 200", global, w.Code, w.Body.String())
	}
	if w := serve(router, http.MethodPatch, global+"/records", "application/json", []byte(`{"oldrr":"blog 3600 IN A 192.0.2.3","newrr":"blog 3600 IN A 192.0.2.4"}`)); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "192.0.2.4") {
		t.Errorf("PATCH %s/records: status %d (%s), want 200 and the new record", global, w.Code, w.Body.String())
	}
	if w := serve(router, http.MethodPost, global+"/records/delete", "application/json", []byte(`["blog 3600 IN A 192.0.2.4"]`)); w.Code != http.StatusOK || strings.Contains(w.Body.String(), "192.0.2.4") {
		t.Errorf("POST %s/records/delete: status %d (%s), want 200 and the record gone", global, w.Code, w.Body.String())
	}

	mail, err := json.Marshal(imported.Services["mail"][0])
	if err != nil {
		t.Fatal(err)
	}
	w = serve(router, http.MethodPost, global+"/web/services", "application/json", mail)
	var added happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &added); w.Code != http.StatusOK || err != nil || len(added.Services["web"]) != 1 {
		t.Fatalf("POST %s/web/services: status %d (%s), want 200 and the service added", global, w.Code, w.Body.String())
	}
	service := global + "/web/services/" + added.Services["web"][0].Id.String()
	if w := serve(router, http.MethodPut, service, "application/json", mail); w.Code != http.StatusOK {
		t.Errorf("PUT %s: status %d (%s), want 200", service, w.Code, w.Body.String())
	}
	if w := serve(router, http.MethodDelete, service, "", nil); w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"web"`) {
		t.Errorf("DELETE %s: status %d (%s), want 200 and the service gone", service, w.Code, w.Body.String())
	}

	if w := serve(router, http.MethodPut, global, "application/json", []byte(`{"name":"Mail servers"}`)); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Mail servers") {
		t.Errorf("PUT %s: status %d (%s), want 200 and the new name", global, w.Code, w.Body.String())
	}

	domain := happydns.Identifier{0x10}
	if w := serve(router, http.MethodPost, "/api/templates", "application/json", []byte(`{"name":"From a zone","domain_id":"`+domain.String()+`"}`)); w.Code != http.StatusBadRequest {
		t.Errorf("POST /api/templates from a zone: status %d (%s), want 400", w.Code, w.Body.String())
	}

	user := happydns.Identifier{0x01}
	theirs := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &user, Name: "theirs"}}
	if err := store.CreateZoneTemplate(theirs); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		if w := serve(router, method, "/api/templates/"+theirs.Id.String(), "application/json", []byte(`{"name":"Taken"}`)); w.Code != http.StatusNotFound {
			t.Errorf("%s of the template of a user: status %d (%s), want 404", method, w.Code, w.Body.String())
		}
	}

	if w := serve(router, http.MethodDelete, global, "", nil); w.Code != http.StatusNoContent {
		t.Errorf("DELETE %s: status %d (%s), want 204", global, w.Code, w.Body.String())
	}
	if _, err := store.GetZoneTemplate(imported.Id); err == nil {
		t.Error("the global template is still there after its deletion")
	}
}
