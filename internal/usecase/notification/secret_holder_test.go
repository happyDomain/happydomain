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

	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	notifUC "git.happydns.org/happyDomain/internal/usecase/notification"
	"git.happydns.org/happyDomain/model"
)

func TestChannelSecretsResealAndInspect(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})

	// Stored before sealing existed.
	legacy := &happydns.NotificationChannel{
		UserId: happydns.Identifier{0x01},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"legacy-secret"}`),
	}
	noSecret := &happydns.NotificationChannel{
		UserId: happydns.Identifier{0x01},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/other"}`),
	}
	for _, ch := range []*happydns.NotificationChannel{legacy, noSecret} {
		id, err := happydns.NewRandomIdentifier()
		if err != nil {
			t.Fatal(err)
		}
		ch.Id = id
		if err := db.CreateChannel(ch); err != nil {
			t.Fatal(err)
		}
	}

	holder := notifUC.NewChannelSecrets(db, channelRegistry(t, instance))

	counts, err := holder.InspectSecrets(ctx)
	if err != nil || counts.Clear != 1 {
		t.Fatalf("InspectSecrets = %+v, %v; want 1 clear", counts, err)
	}

	report, err := holder.ResealSecrets(ctx)
	if err != nil || report.Processed != 2 || report.Changed != 1 || report.Failed != 0 {
		t.Fatalf("ResealSecrets = %+v, %v", report, err)
	}
	stored, _ := db.GetChannel(legacy.Id)
	if strings.Contains(string(stored.Config), "legacy-secret") || !strings.Contains(string(stored.Config), `"secret":"hds:1:`) {
		t.Errorf("stored = %s, want the secret sealed", stored.Config)
	}

	counts, _ = holder.InspectSecrets(ctx)
	if counts.Clear != 0 || counts.Sealed[secret.KindInstance] != 1 {
		t.Errorf("InspectSecrets after = %+v", counts)
	}

	// And back.
	report, err = notifUC.NewChannelSecrets(db, channelRegistry(t, plaintext)).ResealSecrets(ctx)
	if err != nil || report.Changed != 1 {
		t.Fatalf("ResealSecrets(plaintext) = %+v, %v", report, err)
	}
	stored, _ = db.GetChannel(legacy.Id)
	if !strings.Contains(string(stored.Config), `"secret":"legacy-secret"`) {
		t.Errorf("stored = %s, want the secret in clear again", stored.Config)
	}
}

// withRawStore returns a storage and the key-value store under it, to write
// records the storage would refuse.
func withRawStore(t *testing.T) (storage.Storage, storage.KVStorage) {
	t.Helper()
	raw, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(raw)
	if err != nil {
		t.Fatal(err)
	}
	return db, raw
}

func instanceManager(t *testing.T, db storage.Storage) *secret.Manager {
	t.Helper()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Owners: db})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// One record that does not decode, or a channel of a deleted user, keeps
// neither the others from being resealed, nor the status from being told.
func TestChannelSecretsGoPastWhatTheyCannotReseal(t *testing.T) {
	ctx := context.Background()
	db, raw := withRawStore(t)

	corrupt, _ := happydns.NewRandomIdentifier()
	if err := raw.Put("notifch|"+corrupt.String(), "not a channel"); err != nil {
		t.Fatal(err)
	}
	good := legacyChannel(t, db, existingUser(t, db))
	orphan := legacyChannel(t, db, happydns.Identifier{0x09})

	holder := notifUC.NewChannelSecrets(db, channelRegistry(t, instanceManager(t, db)))

	counts, err := holder.InspectSecrets(ctx)
	if err != nil || counts.Clear != 2 || counts.Undecodable != 1 {
		t.Errorf("InspectSecrets = %+v, %v; want 2 clear, 1 undecodable", counts, err)
	}

	report, err := holder.ResealSecrets(ctx)
	if err != nil || report.Changed != 1 || report.Skipped != 1 || report.Failed != 1 {
		t.Errorf("ResealSecrets = %+v, %v; want 1 resealed, the orphan skipped, 1 failure", report, err)
	}
	if stored, _ := db.GetChannel(good.Id); strings.Contains(string(stored.Config), "legacy-secret") {
		t.Errorf("stored = %s, want the secret sealed", stored.Config)
	}
	if stored, _ := db.GetChannel(orphan.Id); !strings.Contains(string(stored.Config), "legacy-secret") {
		t.Errorf("orphan = %s, want it left as it was", stored.Config)
	}
}

// What another writer does while a channel is resealed wins: a user's
// update is kept, a deleted channel stays deleted.
func TestChannelResealLosesToConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	db, _ := inmemory.Instantiate()
	owner := existingUser(t, db)
	registry := channelRegistry(t, instanceManager(t, db))

	updated := legacyChannel(t, db, owner)
	store := &racingChannels{Storage: db, during: func() {
		ch, _ := db.GetChannel(updated.Id)
		ch.Config = json.RawMessage(`{"url":"https://example.com/changed"}`)
		overwriteChannel(t, db, ch)
	}}
	report, err := notifUC.NewChannelSecrets(store, registry).ResealSecrets(ctx)
	if err != nil || report.Skipped != 1 || report.Changed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want the channel skipped", report, err)
	}
	if stored, _ := db.GetChannel(updated.Id); string(stored.Config) != `{"url":"https://example.com/changed"}` {
		t.Errorf("stored = %s, want the user's update kept", stored.Config)
	}

	deleted := legacyChannel(t, db, owner)
	if err := db.DeleteChannel(updated.Id); err != nil {
		t.Fatal(err)
	}
	store = &racingChannels{Storage: db, during: func() {
		if err := db.DeleteChannel(deleted.Id); err != nil {
			t.Fatal(err)
		}
	}}
	report, err = notifUC.NewChannelSecrets(store, registry).ResealSecrets(ctx)
	if err != nil || report.Changed != 0 || report.Failed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want nothing written", report, err)
	}
	if _, err := db.GetChannel(deleted.Id); !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
		t.Errorf("GetChannel = %v, want the deleted channel to stay deleted", err)
	}
}

// A channel of a type no sender handles, removed or renamed since, may hold
// sealed secrets nobody can look at: it is reported, not left out, so that
// the status never reads as nothing sealed while it might be.
func TestChannelSecretsReportUnknownTypes(t *testing.T) {
	ctx := context.Background()
	db, _ := inmemory.Instantiate()

	legacyChannel(t, db, existingUser(t, db))
	unknown := &happydns.NotificationChannel{
		Id:     newTestIdentifier(t),
		UserId: existingUser(t, db),
		Type:   "carrier-pigeon",
		Config: json.RawMessage(`{"secret":"hds:1:AQ:c2VhbGVk"}`),
	}
	if err := db.CreateChannel(unknown); err != nil {
		t.Fatal(err)
	}

	holder := notifUC.NewChannelSecrets(db, channelRegistry(t, instanceManager(t, db)))

	counts, err := holder.InspectSecrets(ctx)
	if err != nil || counts.Clear != 1 || counts.Undecodable != 1 {
		t.Errorf("InspectSecrets = %+v, %v; want 1 clear, 1 undecodable", counts, err)
	}
	if !strings.Contains(strings.Join(counts.Problems, " "), unknown.Id.String()) {
		t.Errorf("problems = %q, want the unknown channel named", counts.Problems)
	}

	report, err := holder.ResealSecrets(ctx)
	if err != nil || report.Changed != 1 || report.Failed != 1 {
		t.Errorf("ResealSecrets = %+v, %v; want 1 resealed, 1 failure", report, err)
	}
}
