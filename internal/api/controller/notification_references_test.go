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
