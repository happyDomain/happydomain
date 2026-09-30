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
	"encoding/json"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/model"
)

// The channels of a deleted user go with them: left behind, their secrets
// would stay stored, and the reseal would skip them on every run.
func TestTidyRemovesOrphanNotificationChannels(t *testing.T) {
	for name, tidy := range map[string]func(happydns.TidyUpUseCase) error{
		"TidyNotificationChannels": func(tu happydns.TidyUpUseCase) error { return tu.TidyNotificationChannels(true) },
		"TidyAll":                  func(tu happydns.TidyUpUseCase) error { return tu.TidyAll(true) },
	} {
		t.Run(name, func(t *testing.T) {
			db, err := inmemory.Instantiate()
			if err != nil {
				t.Fatal(err)
			}

			alive := &happydns.User{Id: newSafeId(t), Email: "alive@example.com"}
			if err := db.CreateOrUpdateUser(alive); err != nil {
				t.Fatal(err)
			}
			gone := newSafeId(t)

			newChannel := func(owner happydns.Identifier) *happydns.NotificationChannel {
				ch := &happydns.NotificationChannel{
					Id:     newSafeId(t),
					UserId: owner,
					Type:   "webhook",
					Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"s"}`),
				}
				if err := db.CreateChannel(ch); err != nil {
					t.Fatal(err)
				}
				return ch
			}
			kept := newChannel(alive.Id)
			orphan := newChannel(gone)

			if err := tidy(usecase.NewTidyUpUsecase(db)); err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			if _, err := db.GetChannel(kept.Id); err != nil {
				t.Errorf("the channel of an existing user was removed: %v", err)
			}
			if _, err := db.GetChannel(orphan.Id); !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
				t.Errorf("the orphan channel is still there: %v", err)
			}
			if chs, err := db.ListChannelsByUser(gone); err != nil || len(chs) != 0 {
				t.Errorf("ListChannelsByUser(gone) = %v, %v; want nothing", chs, err)
			}
		})
	}
}
