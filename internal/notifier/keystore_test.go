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

package notifier

import (
	"context"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

// A channel is used in the background: its secrets never go to a keystore
// that needs its owner, chosen or default.
func TestChannelKeystoreIsUnattended(t *testing.T) {
	ctx := context.Background()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{
		Policy:      secret.PolicyInstance,
		InstanceKey: key,
		Safes:       safes,
		Defaults:    secrettest.NewDefaults(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry(m)
	guard := publicOnlyGuard(t)
	r.Register(Adapt(NewWebhookSender("https://happydomain.example", guard), guard))

	owner := happydns.Identifier{0x01}
	personal, err := m.CreateKeystore(ctx, owner, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetDefaultKeystore(ctx, owner, personal.Ref); err != nil {
		t.Fatal(err)
	}

	ch := webhookChannel(`{"url":"https://example.com/hook","secret":"s3cr3t"}`)
	if err := r.SealChannelConfig(ctx, ch, personal.Ref); !errors.Is(err, secret.ErrKeystoreNotUnattended) {
		t.Errorf("SealChannelConfig(personal) = %v, want ErrKeystoreNotUnattended", err)
	}

	if err := r.SealChannelConfig(ctx, ch, ""); err != nil {
		t.Fatal(err)
	}
	ids, err := r.ChannelSafes(ch)
	if err != nil || len(ids) != 1 {
		t.Fatalf("ChannelSafes = %v, %v; want one safe", ids, err)
	}
	if ids[0].Equals(personal.SafeId) {
		t.Error("a channel went to the default keystore, which needs its owner")
	}
	instanceSafe, err := safes.GetSafeByOwner(owner, secret.KindInstance)
	if err != nil || !ids[0].Equals(instanceSafe.Id) {
		t.Errorf("channel sealed in %v, want the instance safe (%v)", ids, err)
	}

	if _, err := r.OpenChannelConfig(ctx, ch); err != nil {
		t.Errorf("OpenChannelConfig = %v", err)
	}
}
