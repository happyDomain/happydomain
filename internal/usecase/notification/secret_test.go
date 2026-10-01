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

package notification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	notifPkg "git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

// Both send paths, the dispatcher's pool and the tester, sign with the
// secret in clear although it is stored sealed.
func TestSendPathsOpenTheWebhookSecret(t *testing.T) {
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: secrettest.NewSafes()})
	if err != nil {
		t.Fatal(err)
	}

	guard, err := netguard.New("outbound", "-outbound-allowed-target", []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry(secrets)
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", guard), guard))

	var sigs []string
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		sigs = append(sigs, req.Header.Get("X-Happydomain-Signature"))
		bodies = append(bodies, b)
	}))
	defer srv.Close()

	ch := &happydns.NotificationChannel{
		Id:     happydns.Identifier{0x0c},
		UserId: happydns.Identifier{0x01},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"` + srv.URL + `","secret":"s3cr3t"}`),
	}
	if err := registry.SealChannelConfig(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ch.Config), "s3cr3t") {
		t.Fatalf("stored config holds the secret: %s", ch.Config)
	}

	user := &happydns.User{Id: ch.UserId, Email: "u@example.com"}

	if err := NewTester(registry).Send(ch, user); err != nil {
		t.Fatalf("Tester.Send: %v", err)
	}

	pool := NewPool(registry, nil)
	if err := pool.runSend(ch, testPayloadFor(user)); err != nil {
		t.Fatalf("Pool.runSend: %v", err)
	}

	if len(sigs) != 2 {
		t.Fatalf("%d requests received, want 2", len(sigs))
	}
	for i := range sigs {
		mac := hmac.New(sha256.New, []byte("s3cr3t"))
		mac.Write(bodies[i])
		if want := "sha256=" + hex.EncodeToString(mac.Sum(nil)); sigs[i] != want {
			t.Errorf("request %d signature = %q, want %q", i, sigs[i], want)
		}
	}
}

func testPayloadFor(user *happydns.User) *notifPkg.NotificationPayload {
	return &notifPkg.NotificationPayload{
		Recipient: notifPkg.Recipient{Email: user.Email},
		CheckerID: "test",
		NewStatus: happydns.StatusOK,
	}
}

// A stored secret that does not open is told for what it is on both send
// paths: the tester answers the user, and the pool records what the user
// sees in the history of the channel. Neither names the safe.
func TestSendPathsTellAStoredSecretThatDoesNotOpen(t *testing.T) {
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	store := secrettest.NewSafes()
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	registry := notifPkg.NewRegistry(secrets)
	registry.Register(notifPkg.Adapt(notifPkg.NewWebhookSender("https://happydomain.example", nil), nil))

	ch := &happydns.NotificationChannel{
		Id:     happydns.Identifier{0x0d},
		UserId: happydns.Identifier{0x01},
		Type:   notifPkg.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"lost"}`),
	}
	if err := registry.SealChannelConfig(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	safe := secrettest.DeleteSafeOf(t, store, ch.UserId, secret.KindInstance)
	user := &happydns.User{Id: ch.UserId, Email: "u@example.com"}

	err = NewTester(registry).Send(ch, user)
	var he happydns.HTTPError
	if !errors.As(err, &he) || he.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("Tester.Send = %v, want a 400 happydns.HTTPError", err)
	}

	err = NewPool(registry, nil).runSend(ch, testPayloadFor(user))
	if err == nil || !strings.Contains(err.Error(), "enter it again") {
		t.Errorf("Pool.runSend = %v, want the user told to enter it again", err)
	}
	for name, err := range map[string]error{"tester": NewTester(registry).Send(ch, user), "pool": err} {
		if err != nil && strings.Contains(err.Error(), safe.Id.String()) {
			t.Errorf("%s error names the safe: %v", name, err)
		}
	}
}
