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

package database

import (
	"testing"

	"git.happydns.org/happyDomain/model"
)

// The indexes keep the entries of a live template, under its live user or
// among the global ones, and drop those whose user or template is gone.
func TestTidyZoneTemplateIndexes(t *testing.T) {
	s := newDiscoveryTestStore()

	alive := &happydns.User{Email: "alive@example.com"}
	if err := s.CreateOrUpdateUser(alive); err != nil {
		t.Fatal(err)
	}
	kept := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{UserId: &alive.Id, Name: "kept"}}
	global := &happydns.ZoneTemplate{ZoneTemplateMeta: happydns.ZoneTemplateMeta{Name: "global"}}
	for _, tpl := range []*happydns.ZoneTemplate{kept, global} {
		if err := s.CreateZoneTemplate(tpl); err != nil {
			t.Fatal(err)
		}
	}

	missingTemplate, _ := happydns.NewRandomIdentifier()
	goneUser, _ := happydns.NewRandomIdentifier()
	stale := []string{
		zonetplUserKey(alive.Id, missingTemplate),
		zonetplUserKey(goneUser, kept.Id),
		zonetplGlobalKey(missingTemplate),
		zonetplUserPrefix + "garbage",
		zonetplGlobalPrefix + "garbage",
	}
	for _, key := range stale {
		if err := s.db.Put(key, ""); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.TidyZoneTemplateIndexes(); err != nil {
		t.Fatalf("TidyZoneTemplateIndexes: %v", err)
	}

	for _, key := range stale {
		if ok, _ := s.db.Has(key); ok {
			t.Errorf("stale index entry %q is still there", key)
		}
	}
	if got, err := s.ListZoneTemplatesByUser(alive.Id); err != nil || len(got) != 1 || !got[0].Id.Equals(kept.Id) {
		t.Errorf("ListZoneTemplatesByUser = %v, %v; want the kept template", got, err)
	}
	if got, err := s.ListGlobalZoneTemplates(); err != nil || len(got) != 1 || !got[0].Id.Equals(global.Id) {
		t.Errorf("ListGlobalZoneTemplates = %v, %v; want the global template", got, err)
	}
}
