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

package database_test

import (
	"encoding/json"
	"errors"
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func newTemplate(owner *happydns.Identifier, name string) *happydns.ZoneTemplate {
	return &happydns.ZoneTemplate{
		ZoneTemplateMeta: happydns.ZoneTemplateMeta{
			UserId:     owner,
			Name:       name,
			DefaultTTL: 3600,
		},
		Services: map[happydns.Subdomain][]*happydns.Service{},
	}
}

func TestCreateZoneTemplate(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	tpl := newTemplate(&owner, "Mail")
	tpl.Description = "Our mail setup"
	if err := s.CreateZoneTemplate(tpl); err != nil {
		t.Fatal(err)
	}
	if tpl.Id.IsEmpty() {
		t.Fatal("CreateZoneTemplate gave the template no identifier")
	}

	got, err := s.GetZoneTemplate(tpl.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Mail" || got.Description != "Our mail setup" || got.DefaultTTL != 3600 {
		t.Errorf("GetZoneTemplate = %+v, want %+v", got.ZoneTemplateMeta, tpl.ZoneTemplateMeta)
	}
	if got.UserId == nil || !got.UserId.Equals(owner) {
		t.Errorf("GetZoneTemplate owner = %v, want %v", got.UserId, owner)
	}
}

func TestGetZoneTemplateNotFound(t *testing.T) {
	s := newStorage(t)
	if _, err := s.GetZoneTemplate(happydns.Identifier{0x42}); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		t.Errorf("GetZoneTemplate of an unknown template = %v, want ErrZoneTemplateNotFound", err)
	}
}

func TestListZoneTemplatesByOwner(t *testing.T) {
	s := newStorage(t)
	alice, bob := happydns.Identifier{0x01}, happydns.Identifier{0x02}
	for _, tpl := range []*happydns.ZoneTemplate{
		newTemplate(&alice, "a"),
		newTemplate(&alice, "b"),
		newTemplate(&bob, "c"),
		newTemplate(nil, "global"),
	} {
		if err := s.CreateZoneTemplate(tpl); err != nil {
			t.Fatal(err)
		}
	}

	if got, err := s.ListZoneTemplatesByUser(bob); err != nil || len(got) != 1 || got[0].Name != "c" {
		t.Errorf("ListZoneTemplatesByUser(bob) = %v, %v; want only the template of bob", got, err)
	}
	if got, err := s.ListZoneTemplatesByUser(alice); err != nil || len(got) != 2 {
		t.Errorf("ListZoneTemplatesByUser(alice) = %v, %v; want her 2 templates", got, err)
	}
	if got, err := s.ListZoneTemplatesByUser(happydns.Identifier{0x03}); err != nil || len(got) != 0 {
		t.Errorf("ListZoneTemplatesByUser of a user without template = %v, %v; want none", got, err)
	}
	if got, err := s.ListGlobalZoneTemplates(); err != nil || len(got) != 1 || got[0].Name != "global" || !got[0].IsGlobal() {
		t.Errorf("ListGlobalZoneTemplates = %v, %v; want the global template only", got, err)
	}

	iter, err := s.ListAllZoneTemplates()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 4 {
		t.Errorf("ListAllZoneTemplates gave %d templates, want 4", n)
	}
}

func TestUpdateZoneTemplate(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	tpl := newTemplate(&owner, "Web")
	if err := s.CreateZoneTemplate(tpl); err != nil {
		t.Fatal(err)
	}

	tpl.Name = "Website"
	if err := s.UpdateZoneTemplate(tpl); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListZoneTemplatesByUser(owner)
	if err != nil || len(got) != 1 || got[0].Name != "Website" {
		t.Errorf("ListZoneTemplatesByUser = %v, %v; want the updated template, once", got, err)
	}
}

func TestDeleteZoneTemplate(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	mine, global := newTemplate(&owner, "mine"), newTemplate(nil, "global")
	for _, tpl := range []*happydns.ZoneTemplate{mine, global} {
		if err := s.CreateZoneTemplate(tpl); err != nil {
			t.Fatal(err)
		}
	}

	for _, tpl := range []*happydns.ZoneTemplate{mine, global} {
		if err := s.DeleteZoneTemplate(tpl.Id); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetZoneTemplate(tpl.Id); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
			t.Errorf("GetZoneTemplate(%s) after deletion = %v, want ErrZoneTemplateNotFound", tpl.Name, err)
		}
	}
	if got, _ := s.ListZoneTemplatesByUser(owner); len(got) != 0 {
		t.Errorf("ListZoneTemplatesByUser after deletion = %v, want none", got)
	}
	if got, _ := s.ListGlobalZoneTemplates(); len(got) != 0 {
		t.Errorf("ListGlobalZoneTemplates after deletion = %v, want none", got)
	}
}

func TestDeleteZoneTemplateNotFound(t *testing.T) {
	s := newStorage(t)
	if err := s.DeleteZoneTemplate(happydns.Identifier{0x42}); !errors.Is(err, happydns.ErrZoneTemplateNotFound) {
		t.Errorf("DeleteZoneTemplate of an unknown template = %v, want ErrZoneTemplateNotFound", err)
	}
}

func TestRestoreZoneTemplate(t *testing.T) {
	s := newStorage(t)
	alice := happydns.Identifier{0x01}
	msg := &happydns.ZoneTemplateMessage{
		ZoneTemplateMeta: newTemplate(&alice, "restored").ZoneTemplateMeta,
	}
	msg.Id = happydns.Identifier{0x42}

	// Restoring twice the same backup leaves a single template.
	for range 2 {
		if err := s.RestoreZoneTemplate(msg); err != nil {
			t.Fatal(err)
		}
	}

	if got, err := s.GetZoneTemplate(msg.Id); err != nil || got.Name != "restored" {
		t.Errorf("GetZoneTemplate after restore = %v, %v; want the restored template", got, err)
	}
	if got, err := s.ListZoneTemplatesByUser(alice); err != nil || len(got) != 1 || !got[0].Id.Equals(msg.Id) {
		t.Errorf("ListZoneTemplatesByUser(alice) after restore = %v, %v; want the restored template, once", got, err)
	}
}

func TestRestoreZoneTemplateMovesIndex(t *testing.T) {
	s := newStorage(t)
	alice := happydns.Identifier{0x01}
	tpl := newTemplate(&alice, "shared")
	if err := s.CreateZoneTemplate(tpl); err != nil {
		t.Fatal(err)
	}

	// The backup holds it as a global one.
	raw, err := json.Marshal(tpl)
	if err != nil {
		t.Fatal(err)
	}
	var msg happydns.ZoneTemplateMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatal(err)
	}
	msg.UserId = nil
	if err := s.RestoreZoneTemplate(&msg); err != nil {
		t.Fatal(err)
	}

	if got, _ := s.ListZoneTemplatesByUser(alice); len(got) != 0 {
		t.Errorf("ListZoneTemplatesByUser(alice) after restore = %v, want none", got)
	}
	if got, _ := s.ListGlobalZoneTemplates(); len(got) != 1 || !got[0].Id.Equals(tpl.Id) {
		t.Errorf("ListGlobalZoneTemplates after restore = %v, want the restored template", got)
	}
}

func TestRestoreZoneTemplateNeedsIdentifier(t *testing.T) {
	s := newStorage(t)
	if err := s.RestoreZoneTemplate(&happydns.ZoneTemplateMessage{}); err == nil {
		t.Error("RestoreZoneTemplate without identifier succeeded, want an error")
	}
}
