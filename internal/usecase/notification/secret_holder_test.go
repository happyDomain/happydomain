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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
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

	guard, _ := netguard.New("outbound", "-outbound-allowed-target", nil)
	registry := func(m *secret.Manager) *notifPkg.Registry {
		r := notifPkg.NewRegistry(m)
		r.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))
		return r
	}

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
		if err := db.CreateChannel(ch); err != nil {
			t.Fatal(err)
		}
	}

	holder := notifUC.NewChannelSecrets(db, registry(instance))

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
	report, err = notifUC.NewChannelSecrets(db, registry(plaintext)).ResealSecrets(ctx)
	if err != nil || report.Changed != 1 {
		t.Fatalf("ResealSecrets(plaintext) = %+v, %v", report, err)
	}
	stored, _ = db.GetChannel(legacy.Id)
	if !strings.Contains(string(stored.Config), `"secret":"legacy-secret"`) {
		t.Errorf("stored = %s, want the secret in clear again", stored.Config)
	}
}

// A channel that cannot be looked at, a record that does not decode or a type
// this version does not know, may hold sealed secrets: it is counted, and
// keeps neither the others from being resealed nor the status from being
// reported.
func TestChannelSecretsGoPastUndecodableChannels(t *testing.T) {
	ctx := context.Background()
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := kvdb.Put("notifch|corrupt", "not a channel"); err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	guard, _ := netguard.New("outbound", "-outbound-allowed-target", nil)
	registry := notifPkg.NewRegistry(instance)
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))

	legacy := &happydns.NotificationChannel{
		UserId: happydns.Identifier{0x01},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"legacy-secret"}`),
	}
	unknown := &happydns.NotificationChannel{
		UserId: happydns.Identifier{0x01},
		Type:   "removed-type",
		Config: json.RawMessage(`{"token":"hds:1:AQ:c2VhbGVk"}`),
	}
	for _, ch := range []*happydns.NotificationChannel{legacy, unknown} {
		if err := db.CreateChannel(ch); err != nil {
			t.Fatal(err)
		}
	}

	holder := notifUC.NewChannelSecrets(db, registry)

	counts, err := holder.InspectSecrets(ctx)
	if err != nil {
		t.Fatalf("InspectSecrets: %v", err)
	}
	if counts.Clear != 1 || counts.Undecodable != 2 {
		t.Errorf("InspectSecrets = %+v, want 1 clear and 2 undecodable", counts)
	}

	report, err := holder.ResealSecrets(ctx)
	if err != nil {
		t.Fatalf("ResealSecrets: %v", err)
	}
	if report.Processed != 3 || report.Changed != 1 || report.Failed != 2 {
		t.Errorf("ResealSecrets = %+v, want the readable channel resealed, 2 failures", report)
	}
	if stored, _ := db.GetChannel(legacy.Id); strings.Contains(string(stored.Config), "legacy-secret") {
		t.Errorf("stored = %s, want the secret sealed", stored.Config)
	}
}

// A channel is used in the background: moving it to a keystore that needs its
// owner is refused, and reported; one that opens without them takes it.
func TestChannelSecretsMoveOwnerSecrets(t *testing.T) {
	ctx := context.Background()
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	guard, _ := netguard.New("outbound", "-outbound-allowed-target", nil)
	registry := notifPkg.NewRegistry(m)
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))

	owner := happydns.Identifier{0x01}
	ch := &happydns.NotificationChannel{
		Id:     happydns.Identifier{0x0c},
		UserId: owner,
		Name:   "Hook",
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"s3cr3t"}`),
	}
	if err := registry.SealChannelConfig(ctx, ch, ""); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateChannel(ch); err != nil {
		t.Fatal(err)
	}

	holder := notifUC.NewChannelSecrets(db, registry)
	objects, err := holder.OwnerObjects(ctx, owner)
	if err != nil || len(objects) != 1 || objects[0].Label != "Hook" {
		t.Fatalf("OwnerObjects = %+v, %v", objects, err)
	}
	from := objects[0].Safes[0]

	personal, err := m.CreateKeystore(ctx, owner, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	report, err := holder.MoveOwnerSecrets(ctx, owner, from, personal.Ref)
	if err != nil || report.Processed != 1 || report.Failed != 1 || len(report.Errors) != 1 {
		t.Fatalf("MoveOwnerSecrets(personal) = %+v, %v; want the channel reported", report, err)
	}
	stored, _ := db.GetChannel(ch.Id)
	if ids, _ := registry.ChannelSafes(stored); len(ids) != 1 || !ids[0].Equals(from) {
		t.Errorf("the channel moved: %v", ids)
	}
}
