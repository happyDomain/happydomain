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
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

func TestCreateChannelKeepsPresetId(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()

	ch := &happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook", Config: json.RawMessage(`{}`)}
	if err := s.CreateChannel(ch); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if !ch.Id.Equals(id) {
		t.Errorf("CreateChannel replaced the preset id")
	}
	if _, err := s.GetChannel(id); err != nil {
		t.Errorf("GetChannel(preset id): %v", err)
	}

	gen := &happydns.NotificationChannel{UserId: owner, Type: "webhook"}
	if err := s.CreateChannel(gen); err != nil || gen.Id.IsEmpty() {
		t.Errorf("CreateChannel without an id = %v, id %v; want one generated", err, gen.Id)
	}
}

// Secrets are sealed bound to the id: taking one already used must neither
// overwrite that channel nor index it for the new owner.
func TestCreateChannelWithUsedIdFails(t *testing.T) {
	s := newStorage(t)
	owner, _ := happydns.NewRandomIdentifier()
	other, _ := happydns.NewRandomIdentifier()
	id, _ := happydns.NewRandomIdentifier()

	first := &happydns.NotificationChannel{Id: id, UserId: owner, Type: "webhook", Name: "first", Config: json.RawMessage(`{}`)}
	if err := s.CreateChannel(first); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	second := &happydns.NotificationChannel{Id: append(happydns.Identifier(nil), id...), UserId: other, Type: "webhook", Name: "second", Config: json.RawMessage(`{}`)}
	if err := s.CreateChannel(second); err == nil {
		t.Fatal("CreateChannel with an id already used succeeded, want an error")
	}

	got, err := s.GetChannel(id)
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}
	if !got.UserId.Equals(owner) || got.Name != "first" {
		t.Error("the first channel was overwritten")
	}
	if chs, _ := s.ListChannelsByUser(other); len(chs) != 0 {
		t.Error("a failed creation left a user index entry")
	}
}
