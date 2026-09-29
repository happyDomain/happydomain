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
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

func webhookRegistry(t *testing.T) *Registry {
	t.Helper()
	r := newTestRegistry(t)
	guard := publicOnlyGuard(t)
	r.Register(Adapt(NewWebhookSender("https://happydomain.example", guard), guard))
	return r
}

func webhookChannel(config string) *happydns.NotificationChannel {
	return &happydns.NotificationChannel{
		Id:     happydns.Identifier{0x0c},
		UserId: happydns.Identifier{0x01},
		Type:   ChannelTypeWebhook,
		Config: json.RawMessage(config),
	}
}

func decodeMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return m
}

// The API contract: the secret never goes back to the client, only whether
// one is set.
func TestWebhookRedactChannel(t *testing.T) {
	r := webhookRegistry(t)

	red, err := r.RedactChannel(webhookChannel(`{"url":"https://example.com/hook","secret":"s3cr3t"}`))
	if err != nil {
		t.Fatal(err)
	}
	m := decodeMap(t, red.Config)
	if _, ok := m["secret"]; ok || strings.Contains(string(red.Config), "s3cr3t") {
		t.Errorf("redacted config carries the secret: %s", red.Config)
	}
	if m["hasSecret"] != true {
		t.Errorf("hasSecret = %v, want true: %s", m["hasSecret"], red.Config)
	}

	red, err = r.RedactChannel(webhookChannel(`{"url":"https://example.com/hook"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m := decodeMap(t, red.Config); m["hasSecret"] != nil {
		t.Errorf("hasSecret set without a secret: %s", red.Config)
	}
}

// An empty secret on update keeps the stored one; a new one replaces it.
func TestWebhookMergeChannelForUpdate(t *testing.T) {
	r := webhookRegistry(t)
	existing := webhookChannel(`{"url":"https://example.com/hook","secret":"stored"}`)

	merged, err := r.MergeChannelForUpdate(existing, webhookChannel(`{"url":"https://example.com/other","hasSecret":true}`))
	if err != nil {
		t.Fatal(err)
	}
	m := decodeMap(t, merged)
	if m["secret"] != "stored" || m["url"] != "https://example.com/other" || m["hasSecret"] != nil {
		t.Errorf("merged = %s, want the stored secret kept, the new URL, no hasSecret", merged)
	}

	merged, err = r.MergeChannelForUpdate(existing, webhookChannel(`{"url":"https://example.com/hook","secret":"rotated"}`))
	if err != nil {
		t.Fatal(err)
	}
	if m := decodeMap(t, merged); m["secret"] != "rotated" {
		t.Errorf("merged = %s, want the new secret", merged)
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}
	return NewRegistry(m)
}

func instanceRegistry(t *testing.T, allowLoopback bool) (*Registry, *secrettest.Safes) {
	t.Helper()
	h, err := secret.GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes})
	if err != nil {
		t.Fatal(err)
	}

	var allowed []string
	if allowLoopback {
		allowed = []string{"127.0.0.1"}
	}
	guard, err := netguard.New("outbound", "-outbound-allowed-target", allowed)
	if err != nil {
		t.Fatal(err)
	}

	r := NewRegistry(m)
	r.Register(Adapt(NewWebhookSender("https://happydomain.example", guard), guard))
	return r, safes
}

func TestWebhookSealUnderInstancePolicy(t *testing.T) {
	r, _ := instanceRegistry(t, false)
	ch := webhookChannel(`{"url":"https://example.com/hook","secret":"s3cr3t"}`)

	if err := r.SealChannelConfig(context.Background(), ch); err != nil {
		t.Fatalf("SealChannelConfig: %v", err)
	}
	if strings.Contains(string(ch.Config), "s3cr3t") || !strings.Contains(string(ch.Config), `"secret":"hds:1:`) {
		t.Errorf("stored config = %s, want the secret sealed", ch.Config)
	}

	// Still redacted as before.
	red, err := r.RedactChannel(ch)
	if err != nil {
		t.Fatal(err)
	}
	if m := decodeMap(t, red.Config); m["secret"] != nil || m["hasSecret"] != true {
		t.Errorf("redacted = %s", red.Config)
	}

	// An update with no secret keeps the sealed one as it is.
	sealed := decodeMap(t, ch.Config)["secret"]
	merged, err := r.MergeChannelForUpdate(ch, webhookChannel(`{"url":"https://example.com/other"}`))
	if err != nil {
		t.Fatal(err)
	}
	updated := webhookChannel(string(merged))
	if err := r.SealChannelConfig(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	if got := decodeMap(t, updated.Config)["secret"]; got != sealed {
		t.Errorf("secret after update = %v, want the stored %v", got, sealed)
	}

	// A new one is sealed.
	merged, err = r.MergeChannelForUpdate(ch, webhookChannel(`{"url":"https://example.com/hook","secret":"rotated"}`))
	if err != nil {
		t.Fatal(err)
	}
	updated = webhookChannel(string(merged))
	if err := r.SealChannelConfig(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated.Config), "rotated") || decodeMap(t, updated.Config)["secret"] == sealed {
		t.Errorf("config after rotation = %s, want a new sealed secret", updated.Config)
	}
}

func TestWebhookRefusesSealedFromClient(t *testing.T) {
	r, _ := instanceRegistry(t, false)

	err := r.CheckIncomingChannel(webhookChannel(`{"url":"https://example.com/hook","secret":"hds:1:AQ:c2VhbGVk"}`))
	if !errors.Is(err, secret.ErrSealedFromClient) {
		t.Errorf("CheckIncomingChannel(sealed) = %v, want ErrSealedFromClient", err)
	}
	if err := r.CheckIncomingChannel(webhookChannel(`{"url":"https://example.com/hook","secret":"typed"}`)); err != nil {
		t.Errorf("CheckIncomingChannel(clear) = %v", err)
	}
}

// The notification reaches the webhook signed with the secret in clear, both
// for a sealed secret and for one stored before sealing existed.
func TestWebhookSendsWithOpenedSecret(t *testing.T) {
	r, _ := instanceRegistry(t, true)

	for name, prepare := range map[string]func(*happydns.NotificationChannel){
		"sealed": func(ch *happydns.NotificationChannel) {
			if err := r.SealChannelConfig(context.Background(), ch); err != nil {
				t.Fatal(err)
			}
		},
		"legacy plaintext": func(*happydns.NotificationChannel) {},
	} {
		t.Run(name, func(t *testing.T) {
			var gotSig string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				gotSig = req.Header.Get("X-Happydomain-Signature")
				gotBody, _ = io.ReadAll(req.Body)
			}))
			defer srv.Close()

			ch := webhookChannel(`{"url":"` + srv.URL + `","secret":"s3cr3t"}`)
			prepare(ch)

			cfg, err := r.OpenChannelConfig(context.Background(), ch)
			if err != nil {
				t.Fatalf("OpenChannelConfig: %v", err)
			}
			sender, _ := r.Get(ch.Type)
			if err := sender.SendTest(context.Background(), cfg, &happydns.User{Email: "u@example.com"}); err != nil {
				t.Fatalf("SendTest: %v", err)
			}

			mac := hmac.New(sha256.New, []byte("s3cr3t"))
			mac.Write(gotBody)
			if want := "sha256=" + hex.EncodeToString(mac.Sum(nil)); gotSig != want {
				t.Errorf("signature = %q, want %q", gotSig, want)
			}
		})
	}
}

func TestOpenChannelConfigFailsClosed(t *testing.T) {
	r, _ := instanceRegistry(t, false)
	ch := webhookChannel(`{"url":"https://example.com/hook","secret":"` + secret.FormatSealed(happydns.Identifier{0x42}, []byte("x")) + `"}`)

	if _, err := r.OpenChannelConfig(context.Background(), ch); err == nil {
		t.Error("a channel whose secret does not open was accepted for sending")
	}
}

// A sealed secret that was not opened must not be sent unsigned.
func TestWebhookSendRefusesUnopenedSecret(t *testing.T) {
	r, _ := instanceRegistry(t, true)
	ch := webhookChannel(`{"url":"https://example.com/hook","secret":"s3cr3t"}`)
	if err := r.SealChannelConfig(context.Background(), ch); err != nil {
		t.Fatal(err)
	}

	sender, _ := r.Get(ch.Type)
	cfg, err := sender.DecodeConfig(ch.Config)
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendTest(context.Background(), cfg, &happydns.User{Email: "u@example.com"}); err == nil {
		t.Error("a webhook was sent with a sealed secret it could not sign with")
	}
}

// Transports without secrets go through sealing unchanged.
func TestSealChannelWithoutSecret(t *testing.T) {
	r, _ := instanceRegistry(t, false)
	guard := publicOnlyGuard(t)
	r.Register(Adapt(NewUnifiedPushSender("https://happydomain.example", guard), guard))
	r.Register(Adapt(NewEmailSender(nil, "https://happydomain.example"), nil))

	for typ, config := range map[happydns.NotificationChannelType]string{
		"unifiedpush": `{"endpoint":"https://push.example.com/abc"}`,
		"email":       `{}`,
	} {
		ch := &happydns.NotificationChannel{Id: happydns.Identifier{0x0c}, UserId: happydns.Identifier{0x01}, Type: typ, Config: json.RawMessage(config)}
		if err := r.SealChannelConfig(context.Background(), ch); err != nil {
			t.Errorf("%s: SealChannelConfig: %v", typ, err)
			continue
		}
		if _, err := r.OpenChannelConfig(context.Background(), ch); err != nil {
			t.Errorf("%s: OpenChannelConfig: %v", typ, err)
		}
	}
}
