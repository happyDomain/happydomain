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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/model"
)

// Binding the body onto a copy of the stored channel must not overwrite the
// stored config, which the merge reads afterwards: a shorter config used to
// leave the tail of the old one behind, and the update failed.
func TestUpdateChannelWithShorterConfig(t *testing.T) {
	nc, db := newTestNotificationController(t)

	user := &happydns.User{Id: happydns.Identifier{0x01}}
	existingId := newTestIdentifier(t)
	existing := &happydns.NotificationChannel{
		Id:     existingId,
		UserId: user.Id,
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/a/rather/long/path/to/the/hook","secret":"kept"}`),
	}
	if err := db.CreateChannel(existing); err != nil {
		t.Fatal(err)
	}

	w, c := channelRequest(t, http.MethodPut, `{"config":{"url":"https://192.0.2.10/h"}}`, user, existing)
	nc.UpdateChannel(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChannel = %d %s", w.Code, w.Body.String())
	}

	stored, err := db.GetChannel(existing.Id)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(stored.Config, &cfg); err != nil {
		t.Fatalf("stored config %s: %v", stored.Config, err)
	}
	if cfg["url"] != "https://192.0.2.10/h" || cfg["secret"] != "kept" {
		t.Errorf("stored config = %s, want the new URL and the secret kept", stored.Config)
	}
}

func newChannelRegistry() *notifPkg.Registry { return notifPkg.NewRegistry() }

// newWebhookRegistry returns a registry handling webhooks.
func newWebhookRegistry(t *testing.T) *notifPkg.Registry {
	t.Helper()
	return registerWebhook(t, newChannelRegistry())
}

// registerWebhook makes registry handle webhooks, allowing the documentation
// address so that the destination check does not depend on a resolver.
func registerWebhook(t *testing.T, registry *notifPkg.Registry) *notifPkg.Registry {
	t.Helper()
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
	return registry
}

// newTestNotificationController returns a controller handling webhooks over
// an in-memory storage, and that storage.
func newTestNotificationController(t *testing.T) (*NotificationController, storage.Storage) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	return NewNotificationController(nil, newWebhookRegistry(t), db, db, db), db
}

func channelRequest(t *testing.T, method string, body string, user *happydns.User, existing *happydns.NotificationChannel) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, "/notifications/channels", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", user)
	if existing != nil {
		c.Set("notification_channel", existing)
	}
	return w, c
}

func newTestIdentifier(t *testing.T) happydns.Identifier {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// The body is bound onto a clone: the stored preference must not see it.
func TestUpdatePreferenceLeavesExistingAlone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	nc := NewNotificationController(nil, newChannelRegistry(), db, db, db)

	user := &happydns.User{Id: happydns.Identifier{0x01}}
	start := 22
	existing := &happydns.NotificationPreference{
		UserId:     user.Id,
		ChannelIds: []happydns.Identifier{{0x05}},
		QuietStart: &start,
	}
	if err := db.CreatePreference(existing); err != nil {
		t.Fatal(err)
	}

	// A channel of the user: preferences only accept those.
	chId, err := json.Marshal(refChannel(t, db, user).Id)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewBufferString(`{"quietStart":5,"channelIds":[`+string(chId)+`]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("LoggedUser", user)
	c.Set("notification_preference", existing)

	nc.UpdatePreference(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdatePreference = %d %s", w.Code, w.Body.String())
	}

	if *existing.QuietStart != 22 || existing.ChannelIds[0][0] != 0x05 {
		t.Errorf("existing changed by the request: quietStart=%d channelIds=%v", *existing.QuietStart, existing.ChannelIds)
	}
}
