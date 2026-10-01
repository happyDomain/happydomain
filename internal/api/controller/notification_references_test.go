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
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/model"
)

// refController returns a controller over an in-memory storage, and that
// storage. Channel references need no sender: the registry is left out.
func refController(t *testing.T) (*NotificationController, storage.Storage) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	return NewNotificationController(nil, nil, db, db, db), db
}

// refChannel stores a webhook channel of user and returns it.
func refChannel(t *testing.T, db storage.Storage, user *happydns.User) *happydns.NotificationChannel {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	ch := &happydns.NotificationChannel{Id: id, UserId: user.Id, Type: "webhook", Config: json.RawMessage(`{"url":"https://192.0.2.10/hook"}`)}
	if err := db.CreateChannel(ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

// refPreference stores a preference of user sending to channels.
func refPreference(t *testing.T, db storage.Storage, user *happydns.User, channels ...happydns.Identifier) *happydns.NotificationPreference {
	t.Helper()
	pref := &happydns.NotificationPreference{UserId: user.Id, ChannelIds: channels, Enabled: true}
	if err := db.CreatePreference(pref); err != nil {
		t.Fatal(err)
	}
	return pref
}

// refRequest returns a request of user with body, the channel or preference
// in context as the middleware would have put it.
func refRequest(method, body string, user *happydns.User, ctxKey string, ctxValue any) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", user)
	if ctxValue != nil {
		c.Set(ctxKey, ctxValue)
	}
	return w, c
}

// A preference listing channels sends only to those: deleting the only one it
// lists would leave it sending nothing, silently.
func TestDeleteChannelUsedByPreference(t *testing.T) {
	nc, db := refController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	ch := refChannel(t, db, user)
	refPreference(t, db, user, ch.Id)

	w, c := refRequest(http.MethodDelete, "", user, "notification_channel", ch)
	nc.DeleteChannel(c)
	c.Writer.WriteHeaderNow()
	if w.Code != http.StatusConflict {
		t.Fatalf("DeleteChannel = %d %s, want 409", w.Code, w.Body.String())
	}
	if _, err := db.GetChannel(ch.Id); err != nil {
		t.Errorf("channel after a refused deletion: %v", err)
	}
}

// Preferences sending to all channels, or to other channels, or belonging to
// another user, do not hold the channel.
func TestDeleteChannelUnused(t *testing.T) {
	nc, db := refController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	other := &happydns.User{Id: happydns.Identifier{0x02}}
	ch := refChannel(t, db, user)
	kept := refChannel(t, db, user)
	refPreference(t, db, user)
	refPreference(t, db, user, kept.Id)
	refPreference(t, db, other, ch.Id)

	w, c := refRequest(http.MethodDelete, "", user, "notification_channel", ch)
	nc.DeleteChannel(c)
	c.Writer.WriteHeaderNow()
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteChannel = %d %s, want 204", w.Code, w.Body.String())
	}
	if _, err := db.GetChannel(ch.Id); !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
		t.Errorf("GetChannel after deletion = %v, want ErrNotificationChannelNotFound", err)
	}
}

// refBody encodes a preference body listing channels, along with quietStart.
func refBody(t *testing.T, quietStart int, channels ...happydns.Identifier) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"channelIds": channels, "quietStart": quietStart, "enabled": true})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A preference may only list channels of its user: an unknown one would
// never match, and the preference would send nothing.
func TestCreatePreferenceChannels(t *testing.T) {
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	other := &happydns.User{Id: happydns.Identifier{0x02}}

	for name, tc := range map[string]struct {
		channel func(*testing.T, storage.Storage) happydns.Identifier
		want    int
	}{
		"own channel": {func(t *testing.T, db storage.Storage) happydns.Identifier { return refChannel(t, db, user).Id }, http.StatusCreated},
		"unknown":     {func(*testing.T, storage.Storage) happydns.Identifier { return happydns.Identifier{0x0f} }, http.StatusBadRequest},
		"of another":  {func(t *testing.T, db storage.Storage) happydns.Identifier { return refChannel(t, db, other).Id }, http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			nc, db := refController(t)
			w, c := refRequest(http.MethodPost, refBody(t, 3, tc.channel(t, db)), user, "", nil)
			nc.CreatePreference(c)
			if w.Code != tc.want {
				t.Fatalf("CreatePreference = %d %s, want %d", w.Code, w.Body.String(), tc.want)
			}
			prefs, err := db.ListPreferencesByUser(user.Id)
			if err != nil {
				t.Fatal(err)
			}
			if stored := len(prefs) == 1; stored != (tc.want == http.StatusCreated) {
				t.Errorf("%d preferences stored after a %d", len(prefs), w.Code)
			}
		})
	}
}

func TestUpdatePreferenceAddingChannel(t *testing.T) {
	nc, db := refController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	ch := refChannel(t, db, user)
	pref := refPreference(t, db, user, ch.Id)

	w, c := refRequest(http.MethodPut, refBody(t, 3, ch.Id, happydns.Identifier{0x0f}), user, "notification_preference", pref)
	nc.UpdatePreference(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("UpdatePreference adding an unknown channel = %d %s, want 400", w.Code, w.Body.String())
	}
	if stored, _ := db.GetPreference(pref.Id); len(stored.ChannelIds) != 1 || stored.QuietStart != nil {
		t.Errorf("stored = %+v, want it unchanged", stored)
	}

	added := refChannel(t, db, user)
	w, c = refRequest(http.MethodPut, refBody(t, 3, ch.Id, added.Id), user, "notification_preference", pref)
	nc.UpdatePreference(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePreference adding an own channel = %d %s, want 200", w.Code, w.Body.String())
	}
}

// The interface sends back the channels it was given, those it does not show
// included: a reference left from before stays editable.
func TestUpdatePreferenceKeepingStaleChannel(t *testing.T) {
	nc, db := refController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}}
	stale := happydns.Identifier{0x0f}
	pref := refPreference(t, db, user, stale)

	w, c := refRequest(http.MethodPut, refBody(t, 3, stale), user, "notification_preference", pref)
	nc.UpdatePreference(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePreference keeping a stale channel = %d %s, want 200", w.Code, w.Body.String())
	}
	if stored, _ := db.GetPreference(pref.Id); stored.QuietStart == nil || *stored.QuietStart != 3 {
		t.Errorf("stored = %+v, want the update applied", stored)
	}
}
