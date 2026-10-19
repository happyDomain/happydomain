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
	"net/http/httptest"
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

// Header values are sealed and withheld like the signing secret. Echoed back
// with the placeholder, a header keeps its value; renamed, it has none to
// keep, and the update is refused rather than the header silently dropped.
func TestChannelHeadersAreSealed(t *testing.T) {
	nc, db := channelTestController(t)
	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "u@example.com"}

	w, c := channelRequest(t, http.MethodPost, `{"type":"webhook","name":"hook","enabled":true,"config":{"url":"https://192.0.2.10/hook","headers":{"Authorization":"Bearer t0k3n"}}}`, user, nil)
	nc.CreateChannel(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateChannel = %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "t0k3n") {
		t.Errorf("response = %s, want the header value withheld", w.Body.String())
	}

	var created happydns.NotificationChannel
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	stored, err := db.GetChannel(created.Id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored.Config), "t0k3n") || !strings.Contains(string(stored.Config), `"Authorization":"hds:1:`) {
		t.Fatalf("stored = %s, want the header sealed", stored.Config)
	}

	// Echoed back: kept.
	w, c = channelRequest(t, http.MethodPut, `{"name":"renamed","config":{"url":"https://192.0.2.10/hook","headers":{"Authorization":"`+happydns.RedactedSecret+`"}}}`, user, stored)
	nc.UpdateChannel(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChannel = %d %s", w.Code, w.Body.String())
	}
	kept, _ := db.GetChannel(created.Id)
	if !strings.Contains(string(kept.Config), `"Authorization":"hds:1:`) {
		t.Errorf("stored = %s, want the header kept", kept.Config)
	}

	// Renamed: nothing to keep.
	w, c = channelRequest(t, http.MethodPut, `{"config":{"url":"https://192.0.2.10/hook","headers":{"X-Token":"`+happydns.RedactedSecret+`"}}}`, user, kept)
	nc.UpdateChannel(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateChannel = %d %s, want 400", w.Code, w.Body.String())
	}
	if after, _ := db.GetChannel(created.Id); string(after.Config) != string(kept.Config) {
		t.Errorf("stored = %s, want it unchanged", after.Config)
	}
}

// The UnifiedPush endpoint is a capability URL: sealed, withheld, and kept
// when echoed back. A stored one that no longer opens is not the user's to
// fix, and the reason, naming a safe, stays in the logs.
func TestUnifiedPushEndpointIsSealed(t *testing.T) {
	nc, db := channelTestController(t)
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	nc.registry.Register(notifPkg.Adapt(notifPkg.NewUnifiedPushSender("https://happydomain.example", guard), guard))
	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "u@example.com"}

	w, c := channelRequest(t, http.MethodPost, `{"type":"unifiedpush","name":"phone","enabled":true,"config":{"endpoint":"https://192.0.2.10/push/c4p4b1l1ty"}}`, user, nil)
	nc.CreateChannel(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateChannel = %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "c4p4b1l1ty") {
		t.Errorf("response = %s, want the endpoint withheld", w.Body.String())
	}
	var created happydns.NotificationChannel
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	stored, _ := db.GetChannel(created.Id)
	if strings.Contains(string(stored.Config), "c4p4b1l1ty") {
		t.Fatalf("stored = %s, want the endpoint sealed", stored.Config)
	}

	// Echoed back: kept, and checked once opened.
	w, c = channelRequest(t, http.MethodPut, `{"name":"renamed","config":{"endpoint":"`+happydns.RedactedSecret+`"}}`, user, stored)
	nc.UpdateChannel(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateChannel = %d %s", w.Code, w.Body.String())
	}
	kept, _ := db.GetChannel(created.Id)
	if kept.Name != "renamed" || string(kept.Config) != string(stored.Config) {
		t.Errorf("stored = %s %s, want the endpoint kept", kept.Name, kept.Config)
	}

	// A stored endpoint that no longer opens.
	broken := *kept
	broken.Config = json.RawMessage(`{"endpoint":"hds:1:AQ:c2VhbGVk"}`)
	if err := db.UpdateChannel(&broken); err != nil {
		t.Fatal(err)
	}
	w, c = channelRequest(t, http.MethodPut, `{"name":"again","config":{"endpoint":"`+happydns.RedactedSecret+`"}}`, user, &broken)
	nc.UpdateChannel(c)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "safe") {
		t.Errorf("UpdateChannel = %d %s, want a 500 naming no safe", w.Code, w.Body.String())
	}
}

// What the owner entered in a secret field, refused as a reference or in
// place of one, is theirs to fix: answered 400 with the reason.
func TestSealErrorTellsReferenceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, sentinel := range []error{secret.ErrUnresolvedReference, secret.ErrReferencesOnly, secret.ErrReferenceAsSecret} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		sealError(c, fmt.Errorf("headers.X-Token: %w: no such file in the directory of the keystore", sentinel))
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), sentinel.Error()) {
			t.Errorf("sealError(%v) = %d %s, want 400 with the reason", sentinel, w.Code, w.Body.String())
		}
	}
}
