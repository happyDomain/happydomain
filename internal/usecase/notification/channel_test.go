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
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// channelServiceFixture returns a ChannelService handling webhooks, and the
// storage under it.
func channelServiceFixture(t *testing.T) (*notifUC.ChannelService, storage.Storage) {
	t.Helper()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry()
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
	return notifUC.NewChannelService(db, registry), db
}

func newTestIdentifier(t *testing.T) happydns.Identifier {
	t.Helper()
	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// The storage keeps the identifier a channel carries, so creating one has to
// replace whatever the submitted channel held: a client must not choose where
// its channel is stored, be it an identifier already taken or a free one, nor
// whose channel it is.
func TestCreateChannelChoosesIdentifierAndOwner(t *testing.T) {
	svc, db := channelServiceFixture(t)

	other := &happydns.NotificationChannel{
		Id:     newTestIdentifier(t),
		UserId: newTestIdentifier(t),
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/other"}`),
	}
	if err := db.CreateChannel(other); err != nil {
		t.Fatal(err)
	}

	user := &happydns.User{Id: newTestIdentifier(t)}

	for name, clientId := range map[string]happydns.Identifier{"taken": other.Id, "free": newTestIdentifier(t)} {
		t.Run(name, func(t *testing.T) {
			ch := &happydns.NotificationChannel{
				Id:     clientId,
				UserId: other.UserId,
				Type:   notifPkg.ChannelTypeWebhook,
				Config: json.RawMessage(`{"url":"https://192.0.2.10/h"}`),
			}
			if err := svc.CreateChannel(context.Background(), user, ch); err != nil {
				t.Fatalf("CreateChannel: %v", err)
			}

			if ch.Id.Equals(clientId) {
				t.Error("the channel was stored under the identifier the client sent")
			}
			stored, err := db.GetChannel(ch.Id)
			if err != nil {
				t.Fatalf("GetChannel(created) = %v", err)
			}
			if !stored.UserId.Equals(user.Id) {
				t.Errorf("stored channel belongs to %s, want %s", stored.UserId, user.Id)
			}

			kept, err := db.GetChannel(other.Id)
			if err != nil || !kept.UserId.Equals(other.UserId) {
				t.Errorf("the channel already stored under the taken identifier was altered: %v", err)
			}
		})
	}
}

// What the user submitted is at fault: the caller answers 400, and nothing is
// stored.
func TestCreateChannelRefusesInvalidChannel(t *testing.T) {
	for name, ch := range map[string]*happydns.NotificationChannel{
		"unknown type":   {Type: "carrier-pigeon", Config: json.RawMessage(`{}`)},
		"invalid config": {Type: notifPkg.ChannelTypeWebhook, Config: json.RawMessage(`{"url":""}`)},
	} {
		t.Run(name, func(t *testing.T) {
			svc, db := channelServiceFixture(t)
			user := &happydns.User{Id: newTestIdentifier(t)}

			err := svc.CreateChannel(context.Background(), user, ch)

			var verr happydns.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("CreateChannel = %v, want a ValidationError", err)
			}
			if chs, _ := db.ListChannelsByUser(user.Id); len(chs) != 0 {
				t.Errorf("%d channels stored, want none", len(chs))
			}
		})
	}
}

type failingChannelStore struct {
	notifUC.NotificationChannelStorage
}

var errStoreDown = errors.New("store down")

func (failingChannelStore) CreateChannel(*happydns.NotificationChannel) error {
	return errStoreDown
}

// A storage failure is not the user's fault: it must not read as a
// ValidationError, whose message would be sent back to the client.
func TestCreateChannelStorageFailureIsNotValidation(t *testing.T) {
	_, db := channelServiceFixture(t)
	guard, _ := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	registry := notifPkg.NewRegistry()
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
	svc := notifUC.NewChannelService(failingChannelStore{db}, registry)

	err := svc.CreateChannel(context.Background(), &happydns.User{Id: newTestIdentifier(t)}, &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h"}`),
	})

	if !errors.Is(err, errStoreDown) {
		t.Fatalf("CreateChannel = %v, want the storage error", err)
	}
	var verr happydns.ValidationError
	if errors.As(err, &verr) {
		t.Errorf("a storage failure reads as a ValidationError: %v", err)
	}
}
