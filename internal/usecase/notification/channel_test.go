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
	"strings"
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
	return notifUC.NewChannelService(db, channelRegistry(t)), db
}

// channelRegistry handles webhooks, allowing the documentation address so
// that the destination check does not depend on a resolver.
func channelRegistry(t *testing.T) *notifPkg.Registry {
	t.Helper()
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry()
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
	return registry
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
	svc := notifUC.NewChannelService(failingChannelStore{db}, channelRegistry(t))

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

// legacyChannel stores a webhook channel of owner, its secret in clear.
func legacyChannel(t *testing.T, db storage.Storage, owner happydns.Identifier) *happydns.NotificationChannel {
	t.Helper()
	id, _ := happydns.NewRandomIdentifier()
	ch := &happydns.NotificationChannel{
		Id:     id,
		UserId: owner,
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"legacy-secret"}`),
	}
	if err := db.CreateChannel(ch); err != nil {
		t.Fatal(err)
	}
	return ch
}

func existingUser(t *testing.T, db storage.Storage) happydns.Identifier {
	t.Helper()
	id, _ := happydns.NewRandomIdentifier()
	if err := db.CreateOrUpdateUser(&happydns.User{Id: id, Email: id.String() + "@example.com"}); err != nil {
		t.Fatal(err)
	}
	return id
}

// racingChannels runs during once, right after the next read of a channel
// and before whatever is written from that read: another writer landing in
// between, however the reader reads.
type racingChannels struct {
	storage.Storage
	during func()
}

func (s *racingChannels) run() {
	if s.during != nil {
		during := s.during
		s.during = nil
		during()
	}
}

func (s *racingChannels) GetChannel(id happydns.Identifier) (*happydns.NotificationChannel, error) {
	ch, err := s.Storage.GetChannel(id)
	s.run()
	return ch, err
}

func (s *racingChannels) ReplaceChannel(id happydns.Identifier, update func(*happydns.NotificationChannel) (*happydns.NotificationChannel, error)) error {
	return s.Storage.ReplaceChannel(id, func(ch *happydns.NotificationChannel) (*happydns.NotificationChannel, error) {
		s.run()
		return update(ch)
	})
}

// overwriteChannel stores ch over the channel of its identifier, whatever it
// holds, as another writer would.
func overwriteChannel(t *testing.T, db storage.Storage, ch *happydns.NotificationChannel) {
	t.Helper()
	if err := db.ReplaceChannel(ch.Id, func(*happydns.NotificationChannel) (*happydns.NotificationChannel, error) {
		return ch, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// setConfig returns an update replacing the config of a channel with cfg.
func setConfig(cfg string) func(*happydns.NotificationChannel) error {
	return func(ch *happydns.NotificationChannel) error {
		return json.Unmarshal([]byte(`{"config":`+cfg+`}`), ch)
	}
}

// An update carries the stored secret forward. Written over what a reseal
// stored in between, it would bring back a token of a safe the reseal just
// emptied, and that may be dropped next: the update is refused instead.
func TestUpdateChannelDoesNotOverwriteAConcurrentWrite(t *testing.T) {
	ctx := context.Background()
	db, _ := inmemory.Instantiate()
	owner := existingUser(t, db)
	ch := legacyChannel(t, db, owner)
	user := &happydns.User{Id: owner}

	meanwhile := ch.Clone()
	meanwhile.Config = json.RawMessage(`{"url":"https://192.0.2.10/meanwhile","secret":"written-meanwhile"}`)
	store := &racingChannels{Storage: db, during: func() {
		overwriteChannel(t, db, meanwhile)
	}}
	svc := notifUC.NewChannelService(store, channelRegistry(t))

	_, err := svc.UpdateChannel(ctx, user, ch.Id, setConfig(`{"url":"https://192.0.2.10/new"}`))
	var conflict happydns.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("UpdateChannel = %v, want a ConflictError", err)
	}
	if stored, _ := db.GetChannel(ch.Id); string(stored.Config) != string(meanwhile.Config) {
		t.Errorf("stored = %s, want what was written meanwhile", stored.Config)
	}

	// Retrying reads again, and carries forward what is stored now.
	updated, err := svc.UpdateChannel(ctx, user, ch.Id, setConfig(`{"url":"https://192.0.2.10/new"}`))
	if err != nil {
		t.Fatalf("retried UpdateChannel: %v", err)
	}
	stored, _ := db.GetChannel(ch.Id)
	if !strings.Contains(string(stored.Config), "192.0.2.10/new") || !strings.Contains(string(stored.Config), "written-meanwhile") {
		t.Errorf("stored = %s, want the new URL and the secret carried forward", stored.Config)
	}
	if !updated.Id.Equals(ch.Id) {
		t.Errorf("UpdateChannel returned channel %s, want %s", updated.Id, ch.Id)
	}
}

// Only the owner updates a channel, and what the update is applied to is
// what is stored when writing.
func TestUpdateChannelChecksWhatIsStored(t *testing.T) {
	ctx := context.Background()
	db, _ := inmemory.Instantiate()
	ch := legacyChannel(t, db, existingUser(t, db))
	svc := notifUC.NewChannelService(db, channelRegistry(t))

	stranger := &happydns.User{Id: existingUser(t, db)}
	if _, err := svc.UpdateChannel(ctx, stranger, ch.Id, setConfig(`{"url":"https://192.0.2.10/new"}`)); !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
		t.Errorf("UpdateChannel by another user = %v, want ErrNotificationChannelNotFound", err)
	}

	owner := &happydns.User{Id: ch.UserId}
	if _, err := svc.UpdateChannel(ctx, owner, ch.Id, func(c *happydns.NotificationChannel) error {
		c.Type = "carrier-pigeon"
		return nil
	}); !errors.As(err, new(happydns.ValidationError)) {
		t.Errorf("UpdateChannel changing the type = %v, want a ValidationError", err)
	}

	if _, err := svc.UpdateChannel(ctx, owner, ch.Id, func(c *happydns.NotificationChannel) error {
		c.Id = newTestIdentifier(t)
		c.UserId = stranger.Id
		return nil
	}); err != nil {
		t.Fatalf("UpdateChannel = %v", err)
	}
	if stored, err := db.GetChannel(ch.Id); err != nil || !stored.UserId.Equals(ch.UserId) {
		t.Errorf("stored = %+v, %v; want the identifier and owner kept", stored, err)
	}
}
