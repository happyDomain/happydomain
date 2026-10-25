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
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/internal/serviceanalyzer"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/internal/usecase/zonetemplate"
	"git.happydns.org/happyDomain/model"
	_ "git.happydns.org/happyDomain/services"
	_ "git.happydns.org/happyDomain/services/abstract"
)

// storedDomains finds the domains of the storage; nothing else is used by the
// routes of the templates.
type storedDomains struct {
	happydns.DomainUsecase
	store storage.Storage
}

func (d storedDomains) GetUserDomain(user *happydns.User, id happydns.Identifier) (*happydns.Domain, error) {
	domain, err := d.store.GetDomain(id)
	if err != nil {
		return nil, err
	}
	if !domain.Owner.Equals(user.Id) {
		return nil, happydns.ErrDomainNotFound
	}
	return domain, nil
}

// editableZones runs the actions on the zone given.
type editableZones struct{}

func (editableZones) ActionOnEditableZone(_ *happydns.User, _ *happydns.Domain, zone *happydns.Zone, act func(*happydns.Zone) error) (*happydns.Zone, error) {
	return zone, act(zone)
}

type templateAPI struct {
	t      *testing.T
	router *gin.Engine
	store  storage.Storage
	user   *happydns.User
}

func newTemplateAPI(t *testing.T, maxPerUser int) *templateAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	services := serviceUC.NewServiceUsecases()
	zones := zoneUC.NewZoneUsecases(store, services)
	uc := zonetemplate.NewService(store, services.ValidateServiceUC, zones, editableZones{}, store, maxPerUser)

	api := &templateAPI{t: t, router: gin.New(), store: store, user: &happydns.User{Id: happydns.Identifier{0x01}}}
	auth := api.router.Group("/", func(c *gin.Context) {
		c.Set("LoggedUser", api.user)
	})
	DeclareZoneTemplateRoutes(auth, uc, storedDomains{store: store}, zones)
	return api
}

func (api *templateAPI) call(method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	api.router.ServeHTTP(w, req)
	return w
}

func (api *templateAPI) json(method, path string, body any, want int, into any) {
	api.t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		api.t.Fatal(err)
	}
	w := api.call(method, path, "application/json", raw)
	if w.Code != want {
		api.t.Fatalf("%s %s: status %d (%s), want %d", method, path, w.Code, w.Body.String(), want)
	}
	if into != nil {
		// Decoded afresh: json.Unmarshal adds to the maps already there.
		v := reflect.ValueOf(into).Elem()
		v.Set(reflect.Zero(v.Type()))
		if err := json.Unmarshal(w.Body.Bytes(), into); err != nil {
			api.t.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func (api *templateAPI) importZone(name, origin, zone string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("name", name)
	_ = mw.WriteField("origin", origin)
	fw, _ := mw.CreateFormFile("zone", "zone.txt")
	_, _ = fw.Write([]byte(zone))
	_ = mw.Close()
	return api.call(http.MethodPost, "/templates/import", mw.FormDataContentType(), body.Bytes())
}

// zone stores the zone of target.org. the records give.
func (api *templateAPI) zone(records string) *happydns.Zone {
	api.t.Helper()
	zp := dns.NewZoneParser(strings.NewReader(records), "target.org.", "")
	var rrs []happydns.Record
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrs = append(rrs, rr)
	}
	if err := zp.Err(); err != nil {
		api.t.Fatal(err)
	}
	services, ttl, err := serviceanalyzer.AnalyzeZone("target.org.", rrs)
	if err != nil {
		api.t.Fatal(err)
	}

	zone := &happydns.Zone{ZoneMeta: happydns.ZoneMeta{DefaultTTL: ttl}, Services: services}
	if err := api.store.CreateZone(zone); err != nil {
		api.t.Fatal(err)
	}
	return zone
}

// domain stores the domain target.org. of owner, with its zones, the newest
// first.
func (api *templateAPI) domain(owner happydns.Identifier, zones ...*happydns.Zone) *happydns.Domain {
	api.t.Helper()
	domain := &happydns.Domain{Owner: owner, DomainName: "target.org."}
	for _, zone := range zones {
		domain.ZoneHistory = append(domain.ZoneHistory, zone.Id)
	}
	if err := api.store.CreateDomain(domain); err != nil {
		api.t.Fatal(err)
	}
	return domain
}

const templateZone = `$TTL 3600
@     IN MX  10 mail
www   IN A   192.0.2.1
mail  IN A   192.0.2.2
`

func TestZoneTemplateRoutes(t *testing.T) {
	api := newTemplateAPI(t, 0)

	w := api.importZone("Mail", "example.com", templateZone)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: status %d (%s)", w.Code, w.Body.String())
	}
	var tpl happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &tpl); err != nil {
		t.Fatal(err)
	}
	id := tpl.Id.String()
	if len(tpl.Services["www"]) != 1 {
		t.Fatalf("imported template = %+v", tpl)
	}

	var list []happydns.ZoneTemplateMessage
	api.json(http.MethodGet, "/templates", nil, http.StatusOK, &list)
	if len(list) != 1 {
		t.Errorf("GET /templates = %d templates, want 1", len(list))
	}

	api.json(http.MethodPut, "/templates/"+id, happydns.ZoneTemplateInput{Name: "Mail servers"}, http.StatusOK, &tpl)
	if tpl.Name != "Mail servers" {
		t.Errorf("PUT /templates/%s: name %q", id, tpl.Name)
	}

	api.json(http.MethodPost, "/templates/"+id+"/records", []string{"blog 3600 IN A 192.0.2.3"}, http.StatusOK, &tpl)
	if len(tpl.Services["blog"]) != 1 {
		t.Fatalf("POST records: services %v, want blog", tpl.Services)
	}
	api.json(http.MethodPatch, "/templates/"+id+"/records", happydns.UpdateRecordForm{OldRR: "blog 3600 IN A 192.0.2.3", NewRR: "blog 3600 IN A 192.0.2.4"}, http.StatusOK, &tpl)
	api.json(http.MethodPost, "/templates/"+id+"/records/delete", []string{"blog 3600 IN A 192.0.2.4"}, http.StatusOK, &tpl)
	if _, ok := tpl.Services["blog"]; ok {
		t.Errorf("POST records/delete: services %v, want blog gone", tpl.Services)
	}

	www := tpl.Services["www"][0]
	www.UserComment = "the website"
	api.json(http.MethodPut, "/templates/"+id+"/www/services/"+www.Id.String(), www, http.StatusOK, &tpl)
	if tpl.Services["www"][0].UserComment != "the website" {
		t.Errorf("PUT service: %+v", tpl.Services["www"][0].ServiceMeta)
	}
	www.Id = nil
	api.json(http.MethodPost, "/templates/"+id+"/web/services", www, http.StatusOK, &tpl)
	if len(tpl.Services["web"]) != 1 {
		t.Fatalf("POST service: services %v, want web", tpl.Services)
	}
	api.json(http.MethodDelete, "/templates/"+id+"/web/services/"+tpl.Services["web"][0].Id.String(), nil, http.StatusOK, &tpl)

	// Applied to a zone of a domain.
	zone := &happydns.Zone{ZoneMeta: happydns.ZoneMeta{DefaultTTL: 3600}, Services: map[happydns.Subdomain][]*happydns.Service{}}
	if err := api.store.CreateZone(zone); err != nil {
		t.Fatal(err)
	}
	domain := &happydns.Domain{Owner: api.user.Id, DomainName: "target.org.", ZoneHistory: []happydns.Identifier{zone.Id}}
	if err := api.store.CreateDomain(domain); err != nil {
		t.Fatal(err)
	}
	var applied happydns.ZoneMessage
	api.json(http.MethodPost, "/domains/"+domain.Id.String()+"/zone/"+zone.Id.String()+"/apply_template", happydns.ApplyZoneTemplateForm{TemplateId: tpl.Id, Mode: happydns.ZoneTemplateMerge}, http.StatusOK, &applied)
	if len(applied.Services["www"]) != 1 || len(applied.Services["mail"]) != 1 {
		t.Errorf("apply_template: services %v, want www and mail", applied.Services)
	}

	// Saved again as a template, from the domain.
	var saved happydns.ZoneTemplateMessage
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateCreationForm{ZoneTemplateInput: happydns.ZoneTemplateInput{Name: "Target"}, DomainId: &domain.Id}, http.StatusCreated, &saved)
	if len(saved.Services["mail"]) != 1 {
		t.Errorf("POST /templates from a domain = %+v", saved)
	}

	api.json(http.MethodDelete, "/templates/"+id, nil, http.StatusNoContent, nil)
	api.json(http.MethodGet, "/templates/"+id, nil, http.StatusNotFound, nil)
}

func TestZoneTemplateRoutesRefusals(t *testing.T) {
	api := newTemplateAPI(t, 0)

	if w := api.importZone("Broken", "", "www IN A not-an-ip\n"); w.Code != http.StatusBadRequest {
		t.Errorf("import of a broken zone file: status %d, want 400", w.Code)
	}
	if w := api.call(http.MethodGet, "/templates/not-an-id", "", nil); w.Code != http.StatusBadRequest {
		t.Errorf("GET of a malformed identifier: status %d, want 400", w.Code)
	}

	// A global template cannot be changed by a user.
	global := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{Name: "Global"}, Services: map[happydns.Subdomain][]*happydns.Service{}}
	if err := api.store.CreateZoneTemplate(global); err != nil {
		t.Fatal(err)
	}
	api.json(http.MethodGet, "/templates/"+global.Id.String(), nil, http.StatusOK, nil)
	api.json(http.MethodPut, "/templates/"+global.Id.String(), happydns.ZoneTemplateInput{Name: "Mine"}, http.StatusForbidden, nil)
	api.json(http.MethodDelete, "/templates/"+global.Id.String(), nil, http.StatusForbidden, nil)
	api.json(http.MethodPost, "/templates/"+global.Id.String()+"/records", []string{"blog 3600 IN A 192.0.2.3"}, http.StatusForbidden, nil)
	api.json(http.MethodDelete, "/templates/"+global.Id.String()+"/www/services/"+(&happydns.Identifier{0x42}).String(), nil, http.StatusForbidden, nil)

	// Nor can the template of another user be seen.
	other := happydns.Identifier{0x02}
	theirs := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &other, Name: "Theirs"}, Services: map[happydns.Subdomain][]*happydns.Service{}}
	if err := api.store.CreateZoneTemplate(theirs); err != nil {
		t.Fatal(err)
	}
	api.json(http.MethodGet, "/templates/"+theirs.Id.String(), nil, http.StatusNotFound, nil)

	var list []happydns.ZoneTemplateMessage
	api.json(http.MethodGet, "/templates", nil, http.StatusOK, &list)
	if len(list) != 1 || !strings.EqualFold(list[0].Name, "Global") {
		t.Errorf("GET /templates = %+v, want the global template only", list)
	}
}

func TestZoneTemplateRoutesFromAZone(t *testing.T) {
	api := newTemplateAPI(t, 0)

	older := api.zone("$TTL 3600\nwww IN A 192.0.2.1\n")
	domain := api.domain(api.user.Id, api.zone("$TTL 3600\n"), older)
	var saved happydns.ZoneTemplateMessage
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateCreationForm{ZoneTemplateInput: happydns.ZoneTemplateInput{Name: "Older"}, DomainId: &domain.Id, ZoneId: &older.Id}, http.StatusCreated, &saved)
	if len(saved.Services["www"]) != 1 {
		t.Errorf("POST /templates from the zone %s = %+v, want the services of that zone", older.Id, saved)
	}

	empty := api.domain(api.user.Id)
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateCreationForm{ZoneTemplateInput: happydns.ZoneTemplateInput{Name: "None"}, DomainId: &empty.Id}, http.StatusBadRequest, nil)

	theirs := api.domain(happydns.Identifier{0x02}, api.zone("$TTL 3600\nwww IN A 192.0.2.1\n"))
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateCreationForm{ZoneTemplateInput: happydns.ZoneTemplateInput{Name: "Theirs"}, DomainId: &theirs.Id}, http.StatusNotFound, nil)
}

// The template of another user is not found, whatever is asked of it.
func TestZoneTemplateRoutesOfAnotherUser(t *testing.T) {
	api := newTemplateAPI(t, 0)
	other := happydns.Identifier{0x02}
	theirs := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &other, Name: "Theirs"}, Services: map[happydns.Subdomain][]*happydns.Service{}}
	if err := api.store.CreateZoneTemplate(theirs); err != nil {
		t.Fatal(err)
	}
	id := theirs.Id.String()

	api.json(http.MethodPut, "/templates/"+id, happydns.ZoneTemplateInput{Name: "Mine"}, http.StatusNotFound, nil)
	api.json(http.MethodDelete, "/templates/"+id, nil, http.StatusNotFound, nil)

	zone := api.zone("$TTL 3600\n")
	domain := api.domain(api.user.Id, zone)
	api.json(http.MethodPost, "/domains/"+domain.Id.String()+"/zone/"+zone.Id.String()+"/apply_template", happydns.ApplyZoneTemplateForm{TemplateId: theirs.Id, Mode: happydns.ZoneTemplateMerge}, http.StatusNotFound, nil)

	if got, err := api.store.GetZoneTemplate(theirs.Id); err != nil || got.Name != "Theirs" {
		t.Errorf("their template = %v, %v; want it unchanged", got, err)
	}
}

func TestZoneTemplateRoutesQuota(t *testing.T) {
	api := newTemplateAPI(t, 1)
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateInput{Name: "One"}, http.StatusCreated, nil)
	api.json(http.MethodPost, "/templates", happydns.ZoneTemplateInput{Name: "Two"}, http.StatusForbidden, nil)
	if w := api.importZone("Three", "", templateZone); w.Code != http.StatusForbidden {
		t.Errorf("import over the quota: status %d, want 403", w.Code)
	}
}

func TestZoneTemplateRoutesBadRequests(t *testing.T) {
	api := newTemplateAPI(t, 0)
	w := api.importZone("Mail", "example.com", templateZone)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: status %d (%s)", w.Code, w.Body.String())
	}
	var tpl happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &tpl); err != nil {
		t.Fatal(err)
	}
	id := tpl.Id.String()

	zone := api.zone("$TTL 3600\n")
	domain := api.domain(api.user.Id, zone)
	api.json(http.MethodPost, "/domains/"+domain.Id.String()+"/zone/"+zone.Id.String()+"/apply_template", happydns.ApplyZoneTemplateForm{TemplateId: tpl.Id, Mode: "replace"}, http.StatusBadRequest, nil)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("name", "No file")
	_ = mw.Close()
	if w := api.call(http.MethodPost, "/templates/import", mw.FormDataContentType(), body.Bytes()); w.Code != http.StatusBadRequest {
		t.Errorf("import without zone file: status %d, want 400", w.Code)
	}

	api.json(http.MethodPut, "/templates/"+id+"/www/services/not-an-id", tpl.Services["www"][0], http.StatusBadRequest, nil)
	api.json(http.MethodDelete, "/templates/"+id+"/www/services/not-an-id", nil, http.StatusBadRequest, nil)

	for _, route := range [][2]string{
		{http.MethodPost, "/templates"},
		{http.MethodPut, "/templates/" + id},
		{http.MethodPost, "/templates/" + id + "/records"},
	} {
		if w := api.call(route[0], route[1], "application/json", []byte("{")); w.Code != http.StatusBadRequest {
			t.Errorf("%s %s with a broken body: status %d, want 400", route[0], route[1], w.Code)
		}
	}
}

// The global templates, read-only to the users, are applied to their zones.
func TestZoneTemplateRoutesApplyAGlobalTemplate(t *testing.T) {
	api := newTemplateAPI(t, 0)
	w := api.importZone("Global", "example.com", templateZone)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: status %d (%s)", w.Code, w.Body.String())
	}
	var global happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &global); err != nil {
		t.Fatal(err)
	}
	// Owned by no one, as those of the administrators.
	global.UserId = nil
	if err := api.store.RestoreZoneTemplate(&global); err != nil {
		t.Fatal(err)
	}

	zone := api.zone("$TTL 3600\n")
	domain := api.domain(api.user.Id, zone)
	var applied happydns.ZoneMessage
	api.json(http.MethodPost, "/domains/"+domain.Id.String()+"/zone/"+zone.Id.String()+"/apply_template", happydns.ApplyZoneTemplateForm{TemplateId: global.Id, Mode: happydns.ZoneTemplateOverwrite}, http.StatusOK, &applied)
	if len(applied.Services["www"]) != 1 || len(applied.Services["mail"]) != 1 {
		t.Errorf("apply_template of a global template: services %v, want www and mail", applied.Services)
	}
}

// "@" names the apex in the routes of the services.
func TestZoneTemplateRoutesApex(t *testing.T) {
	api := newTemplateAPI(t, 0)
	w := api.importZone("Mail", "example.com", templateZone)
	if w.Code != http.StatusCreated {
		t.Fatalf("import: status %d (%s)", w.Code, w.Body.String())
	}
	var tpl happydns.ZoneTemplateMessage
	if err := json.Unmarshal(w.Body.Bytes(), &tpl); err != nil {
		t.Fatal(err)
	}
	id := tpl.Id.String()

	mx := tpl.Services[""][0]
	mx.UserComment = "the mail servers"
	api.json(http.MethodPut, "/templates/"+id+"/@/services/"+mx.Id.String(), mx, http.StatusOK, &tpl)
	if len(tpl.Services[""]) != 1 || tpl.Services[""][0].UserComment != "the mail servers" {
		t.Errorf("PUT a service of @: apex %v, want the MX commented", tpl.Services[""])
	}
	if _, ok := tpl.Services["@"]; ok {
		t.Errorf("PUT a service of @: services %v, want no subdomain named @", tpl.Services)
	}

	api.json(http.MethodDelete, "/templates/"+id+"/@/services/"+mx.Id.String(), nil, http.StatusOK, &tpl)
	if len(tpl.Services[""]) != 0 {
		t.Errorf("DELETE a service of @: apex %v, want it empty", tpl.Services[""])
	}
}
