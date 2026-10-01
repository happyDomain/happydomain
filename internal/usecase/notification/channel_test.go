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
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

// channelServiceFixture returns a ChannelService handling webhooks, sealing
// under the instance policy, and the storage under it.
func channelServiceFixture(t *testing.T) (*notifUC.ChannelService, storage.Storage) {
	t.Helper()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	return notifUC.NewChannelService(db, channelRegistry(t, instanceManager(t, db))), db
}

// channelRegistry handles webhooks, allowing the documentation address so
// that the destination check does not depend on a resolver.
func channelRegistry(t *testing.T, m *secret.Manager) *notifPkg.Registry {
	t.Helper()
	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"192.0.2.10"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry(m)
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
	svc := notifUC.NewChannelService(failingChannelStore{db}, channelRegistry(t, instanceManager(t, db)))

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

// The secret is sealed before the channel is stored.
func TestCreateChannelSealsSecrets(t *testing.T) {
	svc, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	ch := &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","secret":"s3cr3t"}`),
	}
	if err := svc.CreateChannel(context.Background(), user, ch); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	stored, err := db.GetChannel(ch.Id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored.Config), "s3cr3t") {
		t.Errorf("stored config holds the secret in clear: %s", stored.Config)
	}
}

// A client never has a sealed value to send: accepting one would let a user
// paste a token taken from elsewhere.
func TestCreateChannelRefusesSealedValue(t *testing.T) {
	svc, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	first := &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","secret":"s3cr3t"}`),
	}
	if err := svc.CreateChannel(context.Background(), user, first); err != nil {
		t.Fatal(err)
	}
	stored, err := db.GetChannel(first.Id)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.CreateChannel(context.Background(), user, &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: stored.Config,
	})

	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("CreateChannel = %v, want a ValidationError", err)
	}
	if chs, _ := db.ListChannelsByUser(user.Id); len(chs) != 1 {
		t.Errorf("%d channels stored, want only the first one", len(chs))
	}
}

// On creation, nothing is stored for the placeholder to stand for: a client
// sending it, copying the form of another channel say, gets it refused as a
// bad request rather than a channel silently left unsigned.
func TestCreateChannelRefusesRedactedPlaceholder(t *testing.T) {
	svc, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	err := svc.CreateChannel(context.Background(), user, &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","secret":"` + happydns.RedactedSecret + `"}`),
	})

	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("CreateChannel = %v, want a ValidationError", err)
	}
	if chs, _ := db.ListChannelsByUser(user.Id); len(chs) != 0 {
		t.Errorf("%d channels stored, want none", len(chs))
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
	svc := notifUC.NewChannelService(store, channelRegistry(t, instanceManager(t, db)))

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
	if !strings.Contains(string(stored.Config), "192.0.2.10/new") || strings.Contains(string(stored.Config), "written-meanwhile") {
		t.Errorf("stored = %s, want the new URL and the secret carried forward, sealed", stored.Config)
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
	svc := notifUC.NewChannelService(db, channelRegistry(t, instanceManager(t, db)))

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

// A stored secret that no longer opens, its safe gone, does not make every
// later update a server error: the user is told to enter it again, and doing
// so repairs the channel.
func TestUpdateChannelWithAStoredSecretThatNoLongerOpens(t *testing.T) {
	ctx := context.Background()
	svc, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	ch := &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","secret":"lost"}`),
	}
	if err := svc.CreateChannel(ctx, user, ch); err != nil {
		t.Fatal(err)
	}
	safe := secrettest.DeleteSafeOf(t, db, user.Id, secret.KindInstance)

	_, err := svc.UpdateChannel(ctx, user, ch.Id, func(c *happydns.NotificationChannel) error {
		c.Name = "renamed"
		return nil
	})
	wantSecretUserError(t, err, 400, "enter it again", safe.Id)

	if _, err := svc.UpdateChannel(ctx, user, ch.Id, setConfig(`{"url":"https://192.0.2.10/h","secret":"entered-again"}`)); err != nil {
		t.Fatalf("UpdateChannel entering it again = %v", err)
	}
}

// wantSecretUserError checks that err tells the user what to do about a
// stored secret that does not open, with status, without naming safe.
func wantSecretUserError(t *testing.T, err error, status int, hint string, safe happydns.Identifier) {
	t.Helper()
	var he happydns.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("error = %v (%T), want a happydns.HTTPError", err, err)
	}
	if he.HTTPStatus() != status {
		t.Errorf("status = %d, want %d (%v)", he.HTTPStatus(), status, err)
	}
	msg := he.ToErrorResponse().Message
	if !strings.Contains(msg, hint) {
		t.Errorf("message %q does not say %q", msg, hint)
	}
	if strings.Contains(msg, safe.String()) {
		t.Errorf("the message sent back to the user names the safe: %q", msg)
	}
}

// A safe this instance cannot open, its key missing from the keyset, is the
// administrator's to repair: the user is told so, not asked for the secret,
// and the stored channel is left as it was.
func TestUpdateChannelWithASafeThisInstanceCannotOpen(t *testing.T) {
	ctx := context.Background()
	before, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	ch := &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","secret":"kept"}`),
	}
	if err := before.CreateChannel(ctx, user, ch); err != nil {
		t.Fatal(err)
	}
	stored, err := db.GetChannel(ch.Id)
	if err != nil {
		t.Fatal(err)
	}
	safe, err := db.GetSafeByOwner(user.Id, secret.KindInstance)
	if err != nil {
		t.Fatal(err)
	}

	// Restarted with another keyset.
	after := notifUC.NewChannelService(db, channelRegistry(t, instanceManager(t, db)))
	_, err = after.UpdateChannel(ctx, user, ch.Id, func(c *happydns.NotificationChannel) error {
		c.Name = "renamed"
		return nil
	})
	wantSecretUserError(t, err, 503, "administrator", safe.Id)

	now, err := db.GetChannel(ch.Id)
	if err != nil {
		t.Fatal(err)
	}
	if now.Name != stored.Name || string(now.Config) != string(stored.Config) {
		t.Error("a failed update changed the stored channel")
	}
}

// A header sent with the placeholder but nothing stored under its name,
// renamed say, has no value to keep: the user is asked for it rather than
// the header silently dropped.
func TestUpdateChannelRefusesAPlaceholderWithNothingStored(t *testing.T) {
	ctx := context.Background()
	svc, db := channelServiceFixture(t)
	user := &happydns.User{Id: existingUser(t, db)}

	ch := &happydns.NotificationChannel{
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://192.0.2.10/h","headers":{"Authorization":"Bearer t0k3n"}}`),
	}
	if err := svc.CreateChannel(ctx, user, ch); err != nil {
		t.Fatal(err)
	}

	_, err := svc.UpdateChannel(ctx, user, ch.Id, setConfig(`{"url":"https://192.0.2.10/h","headers":{"X-Token":"`+happydns.RedactedSecret+`"}}`))
	if !errors.As(err, new(happydns.ValidationError)) {
		t.Fatalf("UpdateChannel = %v, want a ValidationError", err)
	}
	if stored, _ := db.GetChannel(ch.Id); !strings.Contains(string(stored.Config), `"Authorization":"hds:1:`) {
		t.Errorf("stored = %s, want it unchanged", stored.Config)
	}
}
