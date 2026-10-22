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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

type sentMail struct {
	to, subject string
}

type recordingMailer struct {
	mu   sync.Mutex
	sent []sentMail
}

func (m *recordingMailer) SendMail(to *mail.Address, subject, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMail{to.Address, subject})
	return nil
}

func eventPayload() *notifPkg.NotificationPayload {
	return &notifPkg.NotificationPayload{
		DomainName: "example.com",
		Event: &notifPkg.NotificationEvent{
			Kind:  "domain_watch",
			Title: "example.com is available",
			Body:  "Register it now.",
		},
	}
}

// notifyUser runs NotifyUser against a fresh storage prepared by setup, and
// returns the mails sent and the records written.
func notifyUser(t *testing.T, setup func(db notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User)) ([]sentMail, []*happydns.NotificationRecord) {
	t.Helper()
	return notifyUserOf(t, &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.com"}, eventPayload(), setup)
}

// notifyUserOf is notifyUser, sending payload to user.
func notifyUserOf(t *testing.T, user *happydns.User, payload *notifPkg.NotificationPayload, setup func(db notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User)) ([]sentMail, []*happydns.NotificationRecord) {
	t.Helper()

	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, Safes: secrettest.NewSafes()})
	if err != nil {
		t.Fatal(err)
	}

	mailer := &recordingMailer{}
	registry := notifPkg.NewRegistry(secrets)
	registry.Register(notifPkg.Adapt(notifPkg.NewEmailSender(mailer, "https://happydomain.example"), nil))
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))

	if setup != nil {
		setup(db, db, user)
	}

	pool := notifUC.NewPool(registry, db)
	d := notifUC.NewDispatcher(db, db, db, db, notifUC.NewResolver(db, db), pool, nil, nil, notifUC.NewStateLocker())
	d.Start()
	d.NotifyUser(user, payload)
	d.Stop()

	records, err := db.ListRecordsByUser(user.Id, 10)
	if err != nil {
		t.Fatal(err)
	}
	return mailer.sent, records
}

func TestNotifyUserFallsBackToTheAccountEmail(t *testing.T) {
	sent, records := notifyUser(t, nil)

	if len(sent) != 1 || sent[0].to != "alice@example.com" || sent[0].subject != "[HappyDomain] example.com is available" {
		t.Fatalf("sent = %+v", sent)
	}
	if len(records) != 1 || !records[0].Success || records[0].CheckerID != "domain_watch" {
		t.Errorf("records = %+v", records)
	}
}

func TestNotifyUserUsesTheChannels(t *testing.T) {
	sent, _ := notifyUser(t, func(db notifUC.NotificationChannelStorage, _ notifUC.NotificationPreferenceStorage, user *happydns.User) {
		for _, ch := range []*happydns.NotificationChannel{
			{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: true, Config: []byte(`{"address":"alerts@example.com"}`)},
			{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: false, Config: []byte(`{"address":"old@example.com"}`)},
		} {
			if err := db.CreateChannel(ch); err != nil {
				t.Fatal(err)
			}
		}
	})

	if len(sent) != 1 || sent[0].to != "alerts@example.com" {
		t.Errorf("sent = %+v, want one mail to the enabled channel", sent)
	}
}

func TestNotifyUserRespectsDisabledChannels(t *testing.T) {
	sent, _ := notifyUser(t, func(db notifUC.NotificationChannelStorage, _ notifUC.NotificationPreferenceStorage, user *happydns.User) {
		if err := db.CreateChannel(&happydns.NotificationChannel{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: false}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 0 {
		t.Errorf("sent = %+v, want nothing: the user disabled their only channel", sent)
	}
}

func TestNotifyUserRespectsDisabledGlobalPreference(t *testing.T) {
	sent, _ := notifyUser(t, func(_ notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User) {
		if err := prefs.CreatePreference(&happydns.NotificationPreference{UserId: user.Id, Enabled: false}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 0 {
		t.Errorf("sent = %+v, want nothing: notifications are disabled", sent)
	}
}

func TestNotifyUserIgnoresTheSeverityThreshold(t *testing.T) {
	sent, _ := notifyUser(t, func(_ notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User) {
		if err := prefs.CreatePreference(&happydns.NotificationPreference{UserId: user.Id, Enabled: true, MinStatus: happydns.StatusCrit}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 1 {
		t.Errorf("sent = %+v, want one mail", sent)
	}
}

func TestNotifyUserIgnoresQuietHours(t *testing.T) {
	sent, _ := notifyUser(t, func(_ notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User) {
		// Quiet for the next two hours: whatever the hour the test runs.
		start := time.Now().UTC().Hour()
		end := (start + 2) % 24
		if err := prefs.CreatePreference(&happydns.NotificationPreference{UserId: user.Id, Enabled: true, QuietStart: &start, QuietEnd: &end}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 1 {
		t.Errorf("sent = %+v, want one mail despite the quiet hours", sent)
	}
}

func TestNotifyUserKeepsTheRecipientEmail(t *testing.T) {
	payload := eventPayload()
	payload.Recipient.Email = "bob@example.com"
	sent, _ := notifyUserOf(t, &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.com"}, payload, nil)

	if len(sent) != 1 || sent[0].to != "bob@example.com" {
		t.Errorf("sent = %+v, want one mail to the recipient of the payload", sent)
	}
}

func TestNotifyUserRecordsTheFailureToReachAUserWithoutEmail(t *testing.T) {
	sent, records := notifyUserOf(t, &happydns.User{Id: happydns.Identifier{0x01}}, eventPayload(), nil)

	if len(sent) != 0 {
		t.Errorf("sent = %+v, want nothing: the user has no email", sent)
	}
	if len(records) != 1 || records[0].Success {
		t.Errorf("records = %+v, want one failed send", records)
	}
}

func TestNotifyUserSendsTheEventToAWebhook(t *testing.T) {
	var mu sync.Mutex
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		mu.Lock()
		bodies = append(bodies, b)
		mu.Unlock()
	}))
	defer srv.Close()

	payload := eventPayload()
	payload.Event.Path = "/watchlist/"
	sent, _ := notifyUserOf(t, &happydns.User{Id: happydns.Identifier{0x01}, Email: "alice@example.com"}, payload, func(db notifUC.NotificationChannelStorage, _ notifUC.NotificationPreferenceStorage, user *happydns.User) {
		cfg, err := json.Marshal(notifPkg.WebhookConfig{URL: srv.URL})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.CreateChannel(&happydns.NotificationChannel{UserId: user.Id, Type: notifPkg.ChannelTypeWebhook, Enabled: true, Config: cfg}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 0 {
		t.Errorf("sent = %+v, want no mail: the user set up a webhook", sent)
	}
	if len(bodies) != 1 {
		t.Fatalf("webhook received %d requests, want 1", len(bodies))
	}
	var got map[string]any
	if err := json.Unmarshal(bodies[0], &got); err != nil {
		t.Fatal(err)
	}
	if got["event"] != "domain_watch" || got["title"] != "example.com is available" || got["message"] != "Register it now." || got["dashboardUrl"] != "https://happydomain.example/watchlist/" {
		t.Errorf("webhook payload = %v", got)
	}
}

func TestNotifyUserRestrictsToTheAllowedChannels(t *testing.T) {
	sent, _ := notifyUser(t, func(db notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User) {
		allowed := &happydns.NotificationChannel{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: true, Config: []byte(`{"address":"alerts@example.com"}`)}
		other := &happydns.NotificationChannel{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: true, Config: []byte(`{"address":"other@example.com"}`)}
		for _, ch := range []*happydns.NotificationChannel{allowed, other} {
			if err := db.CreateChannel(ch); err != nil {
				t.Fatal(err)
			}
		}
		if err := prefs.CreatePreference(&happydns.NotificationPreference{UserId: user.Id, Enabled: true, ChannelIds: []happydns.Identifier{allowed.Id}}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 1 || sent[0].to != "alerts@example.com" {
		t.Errorf("sent = %+v, want one mail to the allowed channel", sent)
	}
}

func TestNotifyUserDoesNotFallBackWhenNoChannelIsAllowed(t *testing.T) {
	sent, _ := notifyUser(t, func(db notifUC.NotificationChannelStorage, prefs notifUC.NotificationPreferenceStorage, user *happydns.User) {
		if err := db.CreateChannel(&happydns.NotificationChannel{UserId: user.Id, Type: notifPkg.ChannelTypeEmail, Enabled: true, Config: []byte(`{"address":"alerts@example.com"}`)}); err != nil {
			t.Fatal(err)
		}
		if err := prefs.CreatePreference(&happydns.NotificationPreference{UserId: user.Id, Enabled: true, ChannelIds: []happydns.Identifier{{0x42}}}); err != nil {
			t.Fatal(err)
		}
	})

	if len(sent) != 0 {
		t.Errorf("sent = %+v, want nothing: the user set up channels, none of them allowed", sent)
	}
}

type failingChannelStorage struct {
	notifUC.NotificationChannelStorage
}

func (failingChannelStorage) ListChannelsByUser(happydns.Identifier) ([]*happydns.NotificationChannel, error) {
	return nil, errors.New("storage unavailable")
}

func TestHasChannelsAssumesSomeOnStorageFailure(t *testing.T) {
	r := notifUC.NewResolver(failingChannelStorage{}, nil)
	if !r.HasChannels(&happydns.User{Id: happydns.Identifier{0x01}}) {
		t.Error("HasChannels() = false on a storage failure, want true: the account email would be used instead of the channels")
	}
}
