// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"encoding/json"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// The notification channels and preferences of a deleted user go with them:
// left behind, a channel keeps its configuration, a webhook secret say,
// stored for good.
func TestTidyRemovesOrphanNotifications(t *testing.T) {
	for name, tidy := range map[string]func(happydns.TidyUpUseCase) error{
		"TidyNotifications": func(tu happydns.TidyUpUseCase) error {
			return errors.Join(
				tu.TidyNotificationChannels(true),
				tu.TidyNotificationPreferences(true),
				tu.TidyNotificationStates(true),
				tu.TidyNotificationRecords(true),
			)
		},
		"TidyAll": func(tu happydns.TidyUpUseCase) error { return tu.TidyAll(true) },
	} {
		t.Run(name, func(t *testing.T) {
			db, err := inmemory.Instantiate()
			if err != nil {
				t.Fatal(err)
			}

			alive := &happydns.User{Id: newTestId(t), Email: "alive@example.com"}
			if err := db.CreateOrUpdateUser(alive); err != nil {
				t.Fatal(err)
			}
			gone := newTestId(t)

			newChannel := func(owner happydns.Identifier) *happydns.NotificationChannel {
				ch := &happydns.NotificationChannel{
					UserId: owner,
					Type:   "webhook",
					Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"s"}`),
				}
				if err := db.CreateChannel(ch); err != nil {
					t.Fatal(err)
				}
				return ch
			}
			newPreference := func(owner happydns.Identifier, ch *happydns.NotificationChannel) *happydns.NotificationPreference {
				pref := &happydns.NotificationPreference{UserId: owner, ChannelIds: []happydns.Identifier{ch.Id}, Enabled: true}
				if err := db.CreatePreference(pref); err != nil {
					t.Fatal(err)
				}
				return pref
			}
			kept := newChannel(alive.Id)
			keptPref := newPreference(alive.Id, kept)
			orphan := newChannel(gone)
			orphanPref := newPreference(gone, orphan)

			target := happydns.CheckTarget{DomainId: "d1"}
			for _, owner := range []happydns.Identifier{alive.Id, gone} {
				if err := db.PutState(&happydns.NotificationState{CheckerID: "ping", Target: target, UserId: owner, Acknowledged: true}); err != nil {
					t.Fatal(err)
				}
				if err := db.CreateRecord(&happydns.NotificationRecord{UserId: owner, CheckerID: "ping", Target: target}); err != nil {
					t.Fatal(err)
				}
			}

			if err := tidy(usecase.NewTidyUpUsecase(db)); err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			if _, err := db.GetChannel(kept.Id); err != nil {
				t.Errorf("the channel of an existing user was removed: %v", err)
			}
			if _, err := db.GetPreference(keptPref.Id); err != nil {
				t.Errorf("the preference of an existing user was removed: %v", err)
			}
			if _, err := db.GetChannel(orphan.Id); !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
				t.Errorf("the orphan channel is still there: %v", err)
			}
			if _, err := db.GetPreference(orphanPref.Id); !errors.Is(err, happydns.ErrNotificationPreferenceNotFound) {
				t.Errorf("the orphan preference is still there: %v", err)
			}
			if chs, err := db.ListChannelsByUser(gone); err != nil || len(chs) != 0 {
				t.Errorf("ListChannelsByUser(gone) = %v, %v; want nothing", chs, err)
			}
			if prefs, err := db.ListPreferencesByUser(gone); err != nil || len(prefs) != 0 {
				t.Errorf("ListPreferencesByUser(gone) = %v, %v; want nothing", prefs, err)
			}
			if _, err := db.GetState("ping", target, alive.Id); err != nil {
				t.Errorf("the state of an existing user was removed: %v", err)
			}
			if _, err := db.GetState("ping", target, gone); !errors.Is(err, happydns.ErrNotificationStateNotFound) {
				t.Errorf("the orphan state is still there: %v", err)
			}
			if recs, err := db.ListRecordsByUser(alive.Id, 0); err != nil || len(recs) != 1 {
				t.Errorf("ListRecordsByUser(alive) = %v, %v; want its record", recs, err)
			}
			if recs, err := db.ListRecordsByUser(gone, 0); err != nil || len(recs) != 0 {
				t.Errorf("ListRecordsByUser(gone) = %v, %v; want nothing", recs, err)
			}
			if iter, err := db.ListAllRecords(); err != nil {
				t.Error(err)
			} else {
				n := 0
				for iter.Next() {
					n++
				}
				iter.Close()
				if n != 1 {
					t.Errorf("%d records left in the storage, want 1", n)
				}
			}
		})
	}
}

func newTestId(t *testing.T) happydns.Identifier {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
