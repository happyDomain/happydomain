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
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// The storage keeps the identifier a channel carries, so the controller has to
// replace whatever the client sent: a client must not choose where its channel
// is stored, be it an identifier already taken or a free one.
func TestCreateChannelIgnoresClientId(t *testing.T) {
	nc, db := newTestNotificationController(t)

	takenId := newTestIdentifier(t)
	other := &happydns.NotificationChannel{
		Id:     takenId,
		UserId: happydns.Identifier{0x02},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/other"}`),
	}
	if err := db.CreateChannel(other); err != nil {
		t.Fatal(err)
	}
	freeId := newTestIdentifier(t)

	for name, clientId := range map[string]happydns.Identifier{"taken": takenId, "free": freeId} {
		t.Run(name, func(t *testing.T) {
			rawId, _ := json.Marshal(clientId)
			body := fmt.Sprintf(`{"id":%s,"type":%q,"config":{"url":"https://192.0.2.10/h"}}`, rawId, notifPkg.ChannelTypeWebhook)

			w, c := channelRequest(t, http.MethodPost, body, &happydns.User{Id: happydns.Identifier{0x01}}, nil)
			nc.CreateChannel(c)
			if w.Code != http.StatusCreated {
				t.Fatalf("CreateChannel = %d %s", w.Code, w.Body.String())
			}

			var created happydns.NotificationChannel
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			if created.Id.Equals(clientId) {
				t.Error("the channel was stored under the identifier the client sent")
			}
			if _, err := db.GetChannel(created.Id); err != nil {
				t.Errorf("GetChannel(created) = %v", err)
			}

			stored, err := db.GetChannel(takenId)
			if err != nil || !stored.UserId.Equals(other.UserId) {
				t.Errorf("the channel already stored under the taken identifier was altered: %v", err)
			}
		})
	}
}

type failingCreateChannelStore struct {
	notifUC.NotificationChannelStorage
}

func (failingCreateChannelStore) CreateChannel(*happydns.NotificationChannel) error {
	return fmt.Errorf("store down, key notifch|secret-internal")
}

// A channel the user got wrong is answered 400 with what is wrong; a storage
// failure 500 without its details.
func TestCreateChannelErrorStatus(t *testing.T) {
	registry := newWebhookRegistry(t)
	_, db := newTestNotificationController(t)

	for name, tc := range map[string]struct {
		store      notifUC.NotificationChannelStorage
		body       string
		wantStatus int
		wantInBody string
		notInBody  string
	}{
		"invalid config": {db, `{"type":"webhook","config":{"url":""}}`, http.StatusBadRequest, "URL", ""},
		"unknown type":   {db, `{"type":"carrier-pigeon","config":{}}`, http.StatusBadRequest, "carrier-pigeon", ""},
		"storage failure": {
			failingCreateChannelStore{}, `{"type":"webhook","config":{"url":"https://192.0.2.10/h"}}`,
			http.StatusInternalServerError, "internal server error", "secret-internal",
		},
	} {
		t.Run(name, func(t *testing.T) {
			nc := NewNotificationController(nil, registry, tc.store, db, db)

			w, c := channelRequest(t, http.MethodPost, tc.body, &happydns.User{Id: happydns.Identifier{0x01}}, nil)
			nc.CreateChannel(c)
			if w.Code != tc.wantStatus {
				t.Fatalf("CreateChannel = %d %s, want %d", w.Code, w.Body.String(), tc.wantStatus)
			}
			if !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Errorf("body %s does not mention %q", w.Body.String(), tc.wantInBody)
			}
			if tc.notInBody != "" && strings.Contains(w.Body.String(), tc.notInBody) {
				t.Errorf("body %s leaks %q", w.Body.String(), tc.notInBody)
			}
		})
	}
}
