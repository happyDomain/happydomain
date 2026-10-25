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

package usecase_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// The templates of a deleted user go with them; the global ones, which
// belong to no one, stay.
func TestTidyRemovesOrphanZoneTemplates(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	alive := &happydns.User{Id: newTestId(t), Email: "alive@example.com"}
	if err := db.CreateOrUpdateUser(alive); err != nil {
		t.Fatal(err)
	}

	gone := newTestId(t)
	kept := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &alive.Id, Name: "kept"}}
	orphan := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &gone, Name: "orphan"}}
	global := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{Name: "global"}}
	for _, tpl := range []*happydns.ZoneTemplate{kept, orphan, global} {
		if err := db.CreateZoneTemplate(tpl); err != nil {
			t.Fatal(err)
		}
	}

	if err := usecase.NewTidyUpUsecase(db).TidyAll(true); err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetZoneTemplate(kept.Id); err != nil {
		t.Errorf("the template of a user still there was removed: %v", err)
	}
	if _, err := db.GetZoneTemplate(global.Id); err != nil {
		t.Errorf("the global template was removed: %v", err)
	}
	if _, err := db.GetZoneTemplate(orphan.Id); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		t.Errorf("the template of a deleted user is still there: %v", err)
	}
	if list, _ := db.ListZoneTemplatesByUser(gone); len(list) != 0 {
		t.Errorf("the index of the deleted user still lists %d templates", len(list))
	}
	if list, _ := db.ListGlobalZoneTemplates(); len(list) != 1 {
		t.Errorf("the global index lists %d templates, want 1", len(list))
	}
}
