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

package notification_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// countingServer counts the requests it receives.
func countingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		hits.Add(1)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func webhookChannel(t *testing.T, user *happydns.User, url string) *happydns.NotificationChannel {
	t.Helper()
	cfg, err := json.Marshal(notifPkg.WebhookConfig{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	return &happydns.NotificationChannel{Id: happydns.Identifier{0x02}, UserId: user.Id, Type: notifPkg.ChannelTypeWebhook, Enabled: true, Config: cfg}
}

// A webhook stored before its type was disabled must not be dialed anymore.
func TestNotifyUserSkipsADisabledChannelType(t *testing.T) {
	srv, hits := countingServer(t)

	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.com"}
	_, records := notifyUserWithTypes(t, []happydns.NotificationChannelType{notifPkg.ChannelTypeEmail}, user, eventPayload(), func(db notifUC.NotificationChannelStorage, _ notifUC.NotificationPreferenceStorage, user *happydns.User) {
		if err := db.CreateChannel(webhookChannel(t, user, srv.URL)); err != nil {
			t.Fatal(err)
		}
	})

	if n := hits.Load(); n != 0 {
		t.Errorf("webhook received %d requests, want none: its type is disabled", n)
	}
	for _, rec := range records {
		if rec.ChannelType == notifPkg.ChannelTypeWebhook && rec.Success {
			t.Errorf("record = %+v, want the webhook send marked as failed", rec)
		}
	}
}

// The test button sends right away: it must refuse a disabled type.
func TestTesterRefusesADisabledChannelType(t *testing.T) {
	srv, hits := countingServer(t)

	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, Safes: secrettest.NewSafes()})
	if err != nil {
		t.Fatal(err)
	}
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry(secrets)
	registry.Register(notifPkg.Adapt(notifPkg.NewEmailSender(&recordingMailer{}, "https://happydomain.example"), nil))
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
	registry.Restrict([]happydns.NotificationChannelType{notifPkg.ChannelTypeEmail})

	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.com"}
	err = notifUC.NewTester(registry).Send(context.Background(), webhookChannel(t, user, srv.URL), user)
	if !errors.Is(err, notifPkg.ErrChannelTypeDisabled) {
		t.Errorf("Send() = %v, want ErrChannelTypeDisabled", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("webhook received %d requests, want none: its type is disabled", n)
	}
}
