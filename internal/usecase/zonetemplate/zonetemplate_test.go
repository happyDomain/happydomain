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

package zonetemplate_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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

// records edits the records of a zone, as the zone use cases do.
type records struct {
	add *zoneUC.AddRecordUsecase
	del *zoneUC.DeleteRecordUsecase
}

func (r records) AddRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return r.add.Add(zone, origin, record)
}

func (r records) DeleteRecord(zone *happydns.Zone, origin string, record happydns.Record) error {
	return r.del.Delete(zone, origin, record)
}

type fixture struct {
	store storage.Storage
	uc    *zonetemplate.Service
}

func newFixture(t *testing.T, maxPerUser int) *fixture {
	t.Helper()
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	list := serviceUC.NewListRecordsUsecase()
	f := &fixture{store: store}
	f.uc = zonetemplate.NewService(
		store,
		serviceUC.NewValidateServiceUsecase(),
		records{add: zoneUC.NewAddRecordUsecase(list), del: zoneUC.NewDeleteRecordUsecase(list, serviceUC.NewSearchRecordUsecase(list))},
		maxPerUser,
	)
	return f
}

var (
	alice = &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.org"}
	bob   = &happydns.User{Id: happydns.Identifier{0x02}, Email: "bob@example.org"}
)

const sourceZone = `$ORIGIN example.com.
$TTL 3600
@     IN SOA ns1.provider.net. hostmaster.example.com. 1 7200 3600 1209600 3600
@     IN NS  ns1.provider.net.
@     IN MX  10 mail.example.com.
@     IN TXT "v=spf1 include:_spf.example.com ~all"
www   IN CNAME example.com.
mail  IN A   192.0.2.1
`

// newZone analyzes a zone file into a zone stored under origin.
func (f *fixture) newZone(t *testing.T, origin, text string) (*happydns.Domain, *happydns.Zone) {
	t.Helper()
	zp := dns.NewZoneParser(strings.NewReader(text), origin, "")
	var rrs []happydns.Record
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		rrs = append(rrs, rr)
	}
	if err := zp.Err(); err != nil {
		t.Fatal(err)
	}

	services, ttl, err := serviceanalyzer.AnalyzeZone(origin, rrs)
	if err != nil {
		t.Fatal(err)
	}
	zone := &happydns.Zone{ZoneMeta: happydns.ZoneMeta{DefaultTTL: ttl}, Services: services}
	if err := f.store.CreateZone(zone); err != nil {
		t.Fatal(err)
	}
	return &happydns.Domain{Id: happydns.Identifier{0x10}, Owner: alice.Id, DomainName: origin, ZoneHistory: []happydns.Identifier{zone.Id}}, zone
}

// recordsOf lists the records of a zone, as text, sorted.
func recordsOf(t *testing.T, origin string, zone *happydns.Zone) []string {
	t.Helper()
	rrs, err := zoneUC.NewListRecordsUsecase(serviceUC.NewListRecordsUsecase()).List(&happydns.Domain{DomainName: origin}, zone)
	if err != nil {
		t.Fatal(err)
	}
	var ret []string
	for _, rr := range rrs {
		hdr := rr.Header()
		ret = append(ret, strings.TrimPrefix(rr.String(), hdr.String()))
		ret[len(ret)-1] = hdr.Name + " " + dns.TypeToString[hdr.Rrtype] + " " + ret[len(ret)-1]
	}
	slices.Sort(ret)
	return ret
}

func (f *fixture) template(t *testing.T, user *happydns.User, name string) *happydns.ZoneTemplate {
	t.Helper()
	tpl, err := f.uc.ImportZoneTemplate(user, happydns.ZoneTemplateInput{Name: name}, "", strings.NewReader(sourceZone))
	if err != nil {
		t.Fatal(err)
	}
	return tpl
}

func TestImportLeavesTheOriginOut(t *testing.T) {
	f := newFixture(t, 0)
	tpl := f.template(t, alice, "Mail")

	if !tpl.UserId.Equals(alice.Id) || tpl.DefaultTTL != 3600 {
		t.Errorf("imported template = %+v, want alice as owner", tpl.ZoneTemplateMeta)
	}
	for subdomain, svcs := range tpl.Services {
		for _, s := range svcs {
			if strings.HasPrefix(s.Type, "abstract.") && strings.HasSuffix(s.Type, "Origin") {
				t.Errorf("imported template holds the origin %s under %q", s.Type, subdomain)
			}
		}
	}
	if len(tpl.Services["www"]) != 1 || len(tpl.Services["mail"]) != 1 {
		t.Errorf("imported template services = %v, want www and mail", tpl.Services)
	}
}

func TestImportRejectsBrokenZoneFile(t *testing.T) {
	f := newFixture(t, 0)
	_, err := f.uc.ImportZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "broken"}, "", strings.NewReader("www IN A not-an-ip\n"))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("ImportZoneTemplate of a broken file = %v, want a validation error", err)
	}
}

func TestCreateFromZone(t *testing.T) {
	f := newFixture(t, 0)
	domain, zone := f.newZone(t, "example.com.", sourceZone)

	tpl, err := f.uc.CreateZoneTemplateFromZone(alice, happydns.ZoneTemplateInput{Name: "  Mine  "}, domain, zone)
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "Mine" {
		t.Errorf("template = %+v, want the name trimmed", tpl.ZoneTemplateMeta)
	}
	if len(tpl.Services[""]) != 2 {
		t.Errorf("template apex = %v, want the MX and the SPF without the origin", tpl.Services[""])
	}

	// The template is a copy: changing it leaves the zone as it was.
	tpl.Services["www"][0].UserComment = "changed"
	if zone.Services["www"][0].UserComment != "" {
		t.Error("the template shares its services with the zone")
	}
}

func TestCreateNeedsAName(t *testing.T) {
	f := newFixture(t, 0)
	if _, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: " "}); err == nil {
		t.Error("CreateZoneTemplate without name succeeded")
	}
	tpl, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "empty"})
	if err != nil || len(tpl.Services) != 0 {
		t.Errorf("CreateZoneTemplate = %v, %v; want an empty template", tpl, err)
	}
}

func TestQuota(t *testing.T) {
	f := newFixture(t, 1)
	if _, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "one"}); err != nil {
		t.Fatal(err)
	}
	var ferr happydns.ForbiddenError
	if _, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "two"}); !errors.As(err, &ferr) {
		t.Errorf("CreateZoneTemplate over the quota = %v, want a forbidden error", err)
	}
	// Global templates are not counted.
	if _, err := f.uc.CreateZoneTemplate(nil, happydns.ZoneTemplateInput{Name: "global"}); err != nil {
		t.Errorf("CreateZoneTemplate of a global template = %v", err)
	}
}

func TestListEmpty(t *testing.T) {
	f := newFixture(t, 0)
	// A nil slice would be answered as null, breaking the clients.
	for _, user := range []*happydns.User{nil, alice} {
		if list, err := f.uc.ListZoneTemplates(user); err != nil || list == nil {
			t.Errorf("ListZoneTemplates(%v) = %#v, %v; want an empty list", user, list, err)
		}
	}
}

func TestListSortsByName(t *testing.T) {
	f := newFixture(t, 0)
	for _, name := range []string{"beta", "Alpha", "gamma"} {
		if _, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: name}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := f.uc.ListZoneTemplates(alice)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tpl := range list {
		names = append(names, tpl.Name)
	}
	if want := []string{"Alpha", "beta", "gamma"}; !slices.Equal(names, want) {
		t.Errorf("ListZoneTemplates = %v, want %v, whatever the case", names, want)
	}
}

func TestAccess(t *testing.T) {
	f := newFixture(t, 0)
	mine := f.template(t, alice, "mine")
	global := f.template(t, nil, "global")

	if !global.IsGlobal() {
		t.Fatal("a template made by an administrator is not global")
	}

	list, err := f.uc.ListZoneTemplates(bob)
	if err != nil || len(list) != 1 || !list[0].Id.Equals(global.Id) {
		t.Errorf("ListZoneTemplates(bob) = %v, %v; want the global template only", list, err)
	}
	list, err = f.uc.ListZoneTemplates(alice)
	if err != nil || len(list) != 2 || !list[0].Id.Equals(mine.Id) {
		t.Errorf("ListZoneTemplates(alice) = %v, %v; want her template first, then the global one", list, err)
	}
	if list, _ := f.uc.ListZoneTemplates(nil); len(list) != 1 {
		t.Errorf("ListZoneTemplates(admin) = %v, want the global template only", list)
	}

	if _, err := f.uc.GetZoneTemplate(bob, mine.Id); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		t.Errorf("GetZoneTemplate of the template of another = %v, want not found", err)
	}
	if _, err := f.uc.GetZoneTemplate(bob, global.Id); err != nil {
		t.Errorf("GetZoneTemplate of a global template = %v", err)
	}

	var ferr happydns.ForbiddenError
	if _, err := f.uc.UpdateZoneTemplate(bob, global.Id, happydns.ZoneTemplateInput{Name: "taken"}); !errors.As(err, &ferr) {
		t.Errorf("UpdateZoneTemplate of a global template by a user = %v, want forbidden", err)
	}
	if err := f.uc.DeleteZoneTemplate(alice, global.Id); !errors.As(err, &ferr) {
		t.Errorf("DeleteZoneTemplate of a global template by a user = %v, want forbidden", err)
	}
	if err := f.uc.DeleteZoneTemplate(bob, mine.Id); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		t.Errorf("DeleteZoneTemplate of the template of another = %v, want not found", err)
	}
	if _, err := f.uc.UpdateZoneTemplate(nil, global.Id, happydns.ZoneTemplateInput{Name: "renamed"}); err != nil {
		t.Errorf("UpdateZoneTemplate of a global template by an administrator = %v", err)
	}
	if err := f.uc.DeleteZoneTemplate(alice, mine.Id); err != nil {
		t.Errorf("DeleteZoneTemplate of her template = %v", err)
	}
}

// A zone file is read for the origin it is written for. The names of the
// services are relative to it: they name the origin of the templates, while
// free texts are kept as they are.
func TestImportFromAnotherOrigin(t *testing.T) {
	f := newFixture(t, 0)
	tpl, err := f.uc.ImportZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "mail"}, "Example.NET", strings.NewReader(`$TTL 3600
@     IN MX  10 mail.example.net.
@     IN TXT "v=spf1 include:_spf.example.net ~all"
mail  IN A   192.0.2.1
`))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(recordsOf(t, happydns.ZoneTemplateOrigin, tpl.Zone()), "\n")
	if !strings.Contains(got, "example.com. MX 10 mail.example.com.") || !strings.Contains(got, "include:_spf.example.net") {
		t.Errorf("records of the template:\n%s\nwant the names under example.com., the SPF include as it was", got)
	}

	var verr happydns.ValidationError
	if _, err := f.uc.ImportZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "elsewhere"}, "example..com", strings.NewReader(sourceZone)); !errors.As(err, &verr) {
		t.Errorf("ImportZoneTemplate for an invalid origin = %v, want a validation error", err)
	}
}

func TestCheckInput(t *testing.T) {
	f := newFixture(t, 0)
	for _, input := range []happydns.ZoneTemplateInput{
		{Name: strings.Repeat("n", 101)},
		{Name: "described", Description: strings.Repeat("d", 1001)},
	} {
		var verr happydns.ValidationError
		if _, err := f.uc.CreateZoneTemplate(alice, input); !errors.As(err, &verr) {
			t.Errorf("CreateZoneTemplate(%.20q, %.20q) = %v, want a validation error", input.Name, input.Description, err)
		}
	}

	if _, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: strings.Repeat("n", 100), Description: strings.Repeat("d", 1000)}); err != nil {
		t.Fatalf("CreateZoneTemplate at the bounds = %v", err)
	}
}

func TestLastModified(t *testing.T) {
	f := newFixture(t, 0)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	f.uc.SetClock(func() time.Time { return now })

	tpl, err := f.uc.CreateZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "dated"})
	if err != nil || !tpl.LastModified.Equal(now) {
		t.Fatalf("CreateZoneTemplate = %v, %v; want it modified at %s", tpl, err, now)
	}

	now = now.Add(time.Hour)
	tpl, err = f.uc.AddZoneTemplateRecords(alice, tpl.Id, []string{"www 3600 IN A 192.0.2.7"})
	if err != nil || !tpl.LastModified.Equal(now) {
		t.Errorf("AddZoneTemplateRecords = %v, %v; want it modified at %s", tpl, err, now)
	}
}

func TestCreateFromZoneWithoutTTL(t *testing.T) {
	f := newFixture(t, 0)
	domain, zone := f.newZone(t, "example.com.", sourceZone)
	zone.DefaultTTL = 0

	tpl, err := f.uc.CreateZoneTemplateFromZone(alice, happydns.ZoneTemplateInput{Name: "no TTL"}, domain, zone)
	if err != nil || tpl.DefaultTTL != 3600 {
		t.Errorf("CreateZoneTemplateFromZone of a zone without TTL = %v, %v; want the TTL 3600", tpl, err)
	}
}

func TestImportRefusesInclude(t *testing.T) {
	f := newFixture(t, 0)
	included := filepath.Join(t.TempDir(), "included.zone")
	if err := os.WriteFile(included, []byte("www 3600 IN A 192.0.2.7\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := f.uc.ImportZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "included"}, "", strings.NewReader("$INCLUDE "+included+"\n"))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("ImportZoneTemplate with $INCLUDE = %v, want a validation error", err)
	}
}

func TestImportRefusesRecordsOutOfOrigin(t *testing.T) {
	f := newFixture(t, 0)
	_, err := f.uc.ImportZoneTemplate(alice, happydns.ZoneTemplateInput{Name: "elsewhere"}, "example.com", strings.NewReader("$ORIGIN other.org.\nwww 3600 IN A 192.0.2.1\n"))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("ImportZoneTemplate of a record out of the origin = %v, want a validation error", err)
	}
}

// The services and records of a template are changed by its owner, those of
// a global template by the administrators only.
func TestEditAccess(t *testing.T) {
	f := newFixture(t, 0)
	mine := f.template(t, alice, "mine")
	global := f.template(t, nil, "global")
	www := *mine.Services["www"][0]

	var ferr happydns.ForbiddenError
	for _, tc := range []struct {
		name  string
		user  *happydns.User
		id    happydns.Identifier
		check func(error) bool
		want  string
	}{
		{"the template of another", bob, mine.Id, func(err error) bool { return errors.Is(err, happydns.ErrZoneTemplateNotFound) }, "not found"},
		{"a global template by a user", alice, global.Id, func(err error) bool { return errors.As(err, &ferr) }, "forbidden"},
	} {
		for edit, err := range map[string]error{
			"AddZoneTemplateService":    second(f.uc.AddZoneTemplateService(tc.user, tc.id, "web", &www)),
			"UpdateZoneTemplateService": second(f.uc.UpdateZoneTemplateService(tc.user, tc.id, "www", www.Id, &www)),
			"DeleteZoneTemplateService": second(f.uc.DeleteZoneTemplateService(tc.user, tc.id, "www", www.Id)),
			"AddZoneTemplateRecords":    second(f.uc.AddZoneTemplateRecords(tc.user, tc.id, []string{"blog 3600 IN A 192.0.2.7"})),
			"DeleteZoneTemplateRecords": second(f.uc.DeleteZoneTemplateRecords(tc.user, tc.id, []string{"mail 3600 IN A 192.0.2.1"})),
			"UpdateZoneTemplateRecord":  second(f.uc.UpdateZoneTemplateRecord(tc.user, tc.id, happydns.UpdateRecordForm{OldRR: "mail 3600 IN A 192.0.2.1", NewRR: "mail 3600 IN A 192.0.2.2"})),
		} {
			if !tc.check(err) {
				t.Errorf("%s of %s = %v, want %s", edit, tc.name, err, tc.want)
			}
		}
	}

	if _, err := f.uc.AddZoneTemplateRecords(nil, global.Id, []string{"blog 3600 IN A 192.0.2.7"}); err != nil {
		t.Errorf("AddZoneTemplateRecords to a global template by an administrator = %v", err)
	}
	if _, err := f.uc.DeleteZoneTemplateService(nil, global.Id, "www", global.Services["www"][0].Id); err != nil {
		t.Errorf("DeleteZoneTemplateService of a global template by an administrator = %v", err)
	}
	for _, tpl := range []*happydns.ZoneTemplate{mine, global} {
		stored, err := f.uc.GetZoneTemplate(alice, tpl.Id)
		if err != nil {
			t.Fatal(err)
		}
		if tpl == mine && (len(stored.Services["www"]) != 1 || len(stored.Services["web"]) != 0) {
			t.Errorf("template of alice = %v, want it unchanged", stored.Services)
		}
		if tpl == global && (len(stored.Services["www"]) != 0 || len(stored.Services["blog"]) != 1) {
			t.Errorf("global template = %v, want the changes of the administrator only", stored.Services)
		}
	}
}

func second[T any](_ T, err error) error {
	return err
}

func TestEditRefusesInvalidServices(t *testing.T) {
	f := newFixture(t, 0)
	tpl := f.template(t, alice, "mail")
	mail := tpl.Services["mail"][0]

	// An abstract.Server without address gives no record.
	body, err := serviceanalyzer.FindService(mail.Type)
	if err != nil {
		t.Fatal(err)
	}
	empty := &happydns.Service{ServiceMeta: happydns.ServiceMeta{Type: mail.Type}, Service: body}
	var verr happydns.ValidationError
	if _, err := f.uc.AddZoneTemplateService(alice, tpl.Id, "blog", empty); !errors.As(err, &verr) {
		t.Errorf("AddZoneTemplateService of an invalid service = %v, want a validation error", err)
	}
	if _, err := f.uc.UpdateZoneTemplateService(alice, tpl.Id, "mail", mail.Id, empty); !errors.As(err, &verr) {
		t.Errorf("UpdateZoneTemplateService to an invalid service = %v, want a validation error", err)
	}
}

func TestEditRefusals(t *testing.T) {
	f := newFixture(t, 0)
	tpl := f.template(t, alice, "mail")
	_, zone := f.newZone(t, "example.com.", sourceZone)

	var origin *happydns.Service
	for _, s := range zone.Services[""] {
		if s.Type == "abstract.Origin" {
			origin = s
		}
	}
	if origin == nil {
		t.Fatalf("the zone has no origin: %v", zone.Services[""])
	}

	var verr happydns.ValidationError
	if _, err := f.uc.AddZoneTemplateService(alice, tpl.Id, "@", origin); !errors.As(err, &verr) {
		t.Errorf("AddZoneTemplateService of the origin = %v, want a validation error", err)
	}
	if _, err := f.uc.AddZoneTemplateService(alice, tpl.Id, "www", nil); !errors.As(err, &verr) {
		t.Errorf("AddZoneTemplateService without service = %v, want a validation error", err)
	}
	if _, err := f.uc.UpdateZoneTemplateService(alice, tpl.Id, "", tpl.Services[""][0].Id, origin); !errors.As(err, &verr) {
		t.Errorf("UpdateZoneTemplateService to the origin = %v, want a validation error", err)
	}

	unknown := happydns.Identifier{0x42}
	updated := *tpl.Services["mail"][0]
	var nerr happydns.NotFoundError
	if _, err := f.uc.UpdateZoneTemplateService(alice, tpl.Id, "mail", unknown, &updated); !errors.As(err, &nerr) {
		t.Errorf("UpdateZoneTemplateService of an unknown service = %v, want not found", err)
	}
	if _, err := f.uc.DeleteZoneTemplateService(alice, tpl.Id, "mail", unknown); !errors.As(err, &nerr) {
		t.Errorf("DeleteZoneTemplateService of an unknown service = %v, want not found", err)
	}
	if _, err := f.uc.DeleteZoneTemplateRecords(alice, tpl.Id, []string{"blog 3600 IN A 192.0.2.9"}); err == nil {
		t.Error("DeleteZoneTemplateRecords of a missing record succeeded")
	}

	stored, err := f.uc.GetZoneTemplate(alice, tpl.Id)
	if err != nil || len(stored.Services[""]) != 2 || len(stored.Services["mail"]) != 1 {
		t.Errorf("stored template = %v, %v; want it unchanged", stored, err)
	}
}

func TestEditServices(t *testing.T) {
	f := newFixture(t, 0)
	tpl := f.template(t, alice, "mail")
	mail := tpl.Services["mail"][0]

	// Through a copy, as a service comes from the API.
	updated := *mail
	updated.UserComment = "the mail server"
	tpl, err := f.uc.UpdateZoneTemplateService(alice, tpl.Id, "mail", mail.Id, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := tpl.Services["mail"][0]; got.UserComment != "the mail server" || !got.Id.Equals(mail.Id) {
		t.Errorf("updated service = %+v, want the comment and the same identifier", got.ServiceMeta)
	}

	if _, err := f.uc.AddZoneTemplateService(alice, tpl.Id, "@", f.template(t, alice, "other").Services["www"][0]); err != nil {
		t.Errorf("AddZoneTemplateService = %v", err)
	}

	tpl, err = f.uc.DeleteZoneTemplateService(alice, tpl.Id, "mail", mail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tpl.Services["mail"]; ok {
		t.Errorf("services after deletion = %v, want mail gone", tpl.Services)
	}

	stored, err := f.uc.GetZoneTemplate(alice, tpl.Id)
	if err != nil || len(stored.Services[""]) != 3 {
		t.Errorf("stored template = %v, %v; want the alias added at the apex", stored, err)
	}
}

func TestEditRecords(t *testing.T) {
	f := newFixture(t, 0)
	tpl := f.template(t, alice, "web")

	tpl, err := f.uc.AddZoneTemplateRecords(alice, tpl.Id, []string{"blog 3600 IN A 192.0.2.7"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tpl.Services["blog"]) != 1 {
		t.Fatalf("services after adding a record = %v, want blog", tpl.Services)
	}

	tpl, err = f.uc.UpdateZoneTemplateRecord(alice, tpl.Id, happydns.UpdateRecordForm{OldRR: "blog 3600 IN A 192.0.2.7", NewRR: "blog 3600 IN A 192.0.2.8"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(recordsOf(t, happydns.ZoneTemplateOrigin, tpl.Zone()), "\n"); !strings.Contains(got, "192.0.2.8") || strings.Contains(got, "192.0.2.7") {
		t.Errorf("records after an update:\n%s", got)
	}

	if tpl, err = f.uc.DeleteZoneTemplateRecords(alice, tpl.Id, []string{"blog 3600 IN A 192.0.2.8"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := tpl.Services["blog"]; ok {
		t.Errorf("services after deleting the last record of blog = %v, want blog gone", tpl.Services)
	}
	if _, err := f.uc.AddZoneTemplateRecords(alice, tpl.Id, []string{"blog.elsewhere.org. 3600 IN A 192.0.2.7"}); err == nil {
		t.Error("AddZoneTemplateRecords out of the origin succeeded")
	}
	if _, err := f.uc.AddZoneTemplateRecords(alice, tpl.Id, []string{"@ 3600 IN NS ns.example.net."}); err == nil {
		t.Error("AddZoneTemplateRecords of a NS at the apex succeeded, a template cannot hold the origin")
	}
}
