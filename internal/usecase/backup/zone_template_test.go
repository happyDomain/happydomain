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

package backup_test

import (
	"context"
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	"git.happydns.org/happyDomain/model"
)

// The zone templates come back under the same identifier, with their
// services and their owner, global ones included.
func TestBackupRestoreZoneTemplates(t *testing.T) {
	src, user := seed(t)

	mine := &happydns.ZoneTemplate{
		ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &user.Id, Name: "mine", DefaultTTL: 3600},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"www": {{ServiceMeta: happydns.ServiceMeta{Type: "svcs.Alias", Domain: "www"}, Service: nil}},
		},
	}
	global := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{Name: "global"}}
	for _, tpl := range []*happydns.ZoneTemplate{mine, global} {
		if err := src.CreateZoneTemplate(tpl); err != nil {
			t.Fatal(err)
		}
	}

	uc := backup.NewUsecase(src, plaintextSecrets(t))
	if dump := uc.BackupUser(user); len(dump.ZoneTemplates) != 1 || dump.ZoneTemplates[0].Name != "mine" {
		t.Errorf("user export holds %+v, want only the template of the user", dump.ZoneTemplates)
	}

	dump := uc.Backup()
	if len(dump.Errors) > 0 {
		t.Fatalf("backup errors: %v", dump.Errors)
	}
	if len(dump.ZoneTemplates) != 2 {
		t.Fatalf("backup holds %d templates, want 2", len(dump.ZoneTemplates))
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	var restored happydns.Backup
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := backup.NewUsecase(dst, plaintextSecrets(t)).Restore(context.Background(), &restored); err != nil {
			t.Fatalf("Restore: %v", err)
		}
	}

	templates, err := dst.ListZoneTemplatesByUser(user.Id)
	if err != nil || len(templates) != 1 || !templates[0].Id.Equals(mine.Id) || len(templates[0].Services["www"]) != 1 {
		t.Errorf("ListZoneTemplatesByUser = %v, %v; want the template with its service, once", templates, err)
	}
	if globals, err := dst.ListGlobalZoneTemplates(); err != nil || len(globals) != 1 || !globals[0].Id.Equals(global.Id) {
		t.Errorf("ListGlobalZoneTemplates = %v, %v; want the global template, once", globals, err)
	}
}
