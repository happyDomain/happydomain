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

func TestListAllChannels(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.CreateChannel(&happydns.NotificationChannel{UserId: owner, Type: "webhook"}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllChannels()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllChannels listed %d channels, want 2", n)
	}
}

// A restore puts a channel back under its identifier, reachable from its
// user, whether it is missing or already there, even under another user.
func TestRestoreChannel(t *testing.T) {
	s := newStorage(t)
	id, _ := happydns.NewRandomIdentifier()
	first, second := happydns.Identifier{0x01}, happydns.Identifier{0x02}

	ch := &happydns.NotificationChannel{Id: id, UserId: first, Type: "webhook", Name: "a", Config: json.RawMessage(`{"url":"https://example.com"}`)}
	if err := s.RestoreChannel(ch); err != nil {
		t.Fatalf("RestoreChannel(missing): %v", err)
	}
	if got, err := s.ListChannelsByUser(first); err != nil || len(got) != 1 || !got[0].Id.Equals(id) || got[0].Name != "a" {
		t.Fatalf("ListChannelsByUser = %v, %v; want the restored channel", got, err)
	}

	moved := &happydns.NotificationChannel{Id: id, UserId: second, Type: "webhook", Name: "b"}
	if err := s.RestoreChannel(moved); err != nil {
		t.Fatalf("RestoreChannel(existing): %v", err)
	}
	if got, _ := s.ListChannelsByUser(first); len(got) != 0 {
		t.Errorf("the previous user still lists %d channels", len(got))
	}
	if got, _ := s.ListChannelsByUser(second); len(got) != 1 || got[0].Name != "b" {
		t.Errorf("ListChannelsByUser(new user) = %v, want the restored channel", got)
	}

	if err := s.RestoreChannel(&happydns.NotificationChannel{UserId: first}); err == nil {
		t.Error("RestoreChannel without an identifier succeeded")
	}
}

func TestListAllPreferences(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.CreatePreference(&happydns.NotificationPreference{UserId: owner}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllPreferences()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllPreferences listed %d preferences, want 2", n)
	}
}

func TestRestorePreference(t *testing.T) {
	s := newStorage(t)
	id, _ := happydns.NewRandomIdentifier()
	first, second := happydns.Identifier{0x01}, happydns.Identifier{0x02}

	if err := s.RestorePreference(&happydns.NotificationPreference{Id: id, UserId: first, Timezone: "UTC"}); err != nil {
		t.Fatalf("RestorePreference(missing): %v", err)
	}
	if got, err := s.ListPreferencesByUser(first); err != nil || len(got) != 1 || !got[0].Id.Equals(id) {
		t.Fatalf("ListPreferencesByUser = %v, %v; want the restored preference", got, err)
	}

	if err := s.RestorePreference(&happydns.NotificationPreference{Id: id, UserId: second, Timezone: "Europe/Paris"}); err != nil {
		t.Fatalf("RestorePreference(existing): %v", err)
	}
	if got, _ := s.ListPreferencesByUser(first); len(got) != 0 {
		t.Errorf("the previous user still lists %d preferences", len(got))
	}
	if got, _ := s.ListPreferencesByUser(second); len(got) != 1 || got[0].Timezone != "Europe/Paris" {
		t.Errorf("ListPreferencesByUser(new user) = %v, want the restored preference", got)
	}

	if err := s.RestorePreference(&happydns.NotificationPreference{UserId: first}); err == nil {
		t.Error("RestorePreference without an identifier succeeded")
	}
}

func TestListAllStates(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.PutState(&happydns.NotificationState{CheckerID: "ping", UserId: owner, LastStatus: 2}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllStates()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		if iter.Item().CheckerID != "ping" {
			t.Errorf("listed state = %+v, want the stored one", iter.Item())
		}
		n++
	}
	if n != 2 {
		t.Errorf("ListAllStates listed %d states, want 2", n)
	}
}

func TestListAllRecords(t *testing.T) {
	s := newStorage(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		if err := s.CreateRecord(&happydns.NotificationRecord{UserId: owner, CheckerID: "ping"}); err != nil {
			t.Fatal(err)
		}
	}

	iter, err := s.ListAllRecords()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	if n != 2 {
		t.Errorf("ListAllRecords listed %d records, want 2", n)
	}
}

func TestDeleteRecord(t *testing.T) {
	s := newStorage(t)
	owner := happydns.Identifier{0x01}
	keep := &happydns.NotificationRecord{UserId: owner, CheckerID: "keep"}
	drop := &happydns.NotificationRecord{UserId: owner, CheckerID: "drop"}
	for _, rec := range []*happydns.NotificationRecord{keep, drop} {
		if err := s.CreateRecord(rec); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.DeleteRecord(drop.Id); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if got, err := s.ListRecordsByUser(owner, 0); err != nil || len(got) != 1 || got[0].CheckerID != "keep" {
		t.Errorf("ListRecordsByUser = %v, %v; want only the kept record", got, err)
	}
	if err := s.DeleteRecord(drop.Id); err != nil {
		t.Errorf("DeleteRecord of a missing record: %v", err)
	}
}

func TestRestoreRecord(t *testing.T) {
	s := newStorage(t)
	id, _ := happydns.NewRandomIdentifier()
	first, second := happydns.Identifier{0x01}, happydns.Identifier{0x02}

	if err := s.RestoreRecord(&happydns.NotificationRecord{Id: id, UserId: first, CheckerID: "a"}); err != nil {
		t.Fatalf("RestoreRecord(missing): %v", err)
	}
	if got, err := s.ListRecordsByUser(first, 0); err != nil || len(got) != 1 || !got[0].Id.Equals(id) {
		t.Fatalf("ListRecordsByUser = %v, %v; want the restored record", got, err)
	}

	if err := s.RestoreRecord(&happydns.NotificationRecord{Id: id, UserId: second, CheckerID: "b"}); err != nil {
		t.Fatalf("RestoreRecord(existing): %v", err)
	}
	if got, _ := s.ListRecordsByUser(first, 0); len(got) != 0 {
		t.Errorf("the previous user still lists %d records", len(got))
	}
	if got, _ := s.ListRecordsByUser(second, 0); len(got) != 1 || got[0].CheckerID != "b" {
		t.Errorf("ListRecordsByUser(new user) = %v, want the restored record", got)
	}

	if err := s.RestoreRecord(&happydns.NotificationRecord{UserId: first}); err == nil {
		t.Error("RestoreRecord without an identifier succeeded")
	}
}
