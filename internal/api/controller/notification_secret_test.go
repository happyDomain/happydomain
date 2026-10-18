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
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/model"
)

func channelTestController(t *testing.T) (*NotificationController, storage.Storage) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	if err != nil {
		t.Fatal(err)
	}

	// Allow the documentation address, so that the destination check does
	// not depend on a resolver.
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry(secrets)
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))

	return NewNotificationController(nil, registry, db, db, db), db
}

func TestChannelSecretLifecycle(t *testing.T) {
	nc, db := channelTestController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "u@example.com"}

	w, c := channelRequest(t, http.MethodPost, `{"type":"webhook","name":"hook","enabled":true,"config":{"url":"https://192.0.2.10/hook","secret":"s3cr3t"}}`, user, nil)
	nc.CreateChannel(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateChannel = %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "s3cr3t") || !strings.Contains(w.Body.String(), `"hasSecret":true`) {
		t.Errorf("response = %s, want the secret withheld", w.Body.String())
	}

	var created happydns.NotificationChannel
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	stored, err := db.GetChannel(created.Id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored.Config), "s3cr3t") || !strings.Contains(string(stored.Config), `"secret":"hds:1:`) {
		t.Fatalf("stored = %s, want the secret sealed", stored.Config)
	}
	sealed := stored.Config

	// Rename without touching the secret: the sealed value is kept.
	w, c = channelRequest(t, http.MethodPut, `{"name":"renamed","config":{"url":"https://192.0.2.10/hook"}}`, user, stored)
	nc.UpdateChannel(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChannel = %d %s", w.Code, w.Body.String())
	}
	stored, _ = db.GetChannel(created.Id)
	if stored.Name != "renamed" || string(stored.Config) != string(sealed) {
		t.Errorf("stored = %s %s, want the name changed and the config as it was", stored.Name, stored.Config)
	}

	// Omitting the config altogether keeps it too.
	w, c = channelRequest(t, http.MethodPut, `{"name":"again"}`, user, stored)
	nc.UpdateChannel(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChannel without config = %d %s", w.Code, w.Body.String())
	}

	// A sealed value from the client is refused.
	w, c = channelRequest(t, http.MethodPut, `{"config":{"url":"https://192.0.2.10/hook","secret":"hds:1:AQ:c2VhbGVk"}}`, user, stored)
	nc.UpdateChannel(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateChannel(sealed) = %d, want 400", w.Code)
	}

	w, c = channelRequest(t, http.MethodPost, `{"type":"webhook","config":{"url":"https://192.0.2.10/hook","secret":"hds:1:AQ:c2VhbGVk"}}`, user, nil)
	nc.CreateChannel(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateChannel(sealed) = %d, want 400", w.Code)
	}
}

// The placeholder stands for a stored value: sent where none is stored, it is
// the client's mistake, answered as such.
func TestCreateChannelRefusesPlaceholderSecret(t *testing.T) {
	nc, db := channelTestController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "u@example.com"}

	w, c := channelRequest(t, http.MethodPost, `{"type":"webhook","name":"hook","enabled":true,"config":{"url":"https://192.0.2.10/hook","secret":"`+happydns.RedactedSecret+`"}}`, user, nil)
	nc.CreateChannel(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("CreateChannel = %d %s, want 400", w.Code, w.Body.String())
	}

	chs, err := db.ListChannelsByUser(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(chs) != 0 {
		t.Errorf("%d channels stored holding the placeholder", len(chs))
	}
}
