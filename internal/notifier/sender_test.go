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
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/model"
)

func publicOnlyGuard(t *testing.T) *netguard.Guard {
	t.Helper()
	g, err := netguard.New("outbound", "-outbound-allowed-target", nil)
	if err != nil {
		t.Fatalf("building guard: %v", err)
	}
	return g
}

// The adapter, not the config, owns the URL shape rules: a sender that declares
// a destination gets it checked whether or not its Validate looks at it.
func TestTypedAdapterDecodeChecksDestinationShape(t *testing.T) {
	adapter := Adapt(NewWebhookSender("https://happydomain.example", publicOnlyGuard(t)), publicOnlyGuard(t))

	for _, raw := range []string{
		`{"url":"ftp://example.com/hook"}`,
		`{"url":"https://"}`,
		`{"url":"://nope"}`,
	} {
		if _, err := adapter.DecodeConfig(json.RawMessage(raw)); err == nil {
			t.Errorf("DecodeConfig(%s) accepted a URL it should have refused", raw)
		}
	}

	if _, err := adapter.DecodeConfig(json.RawMessage(`{"url":"https://example.com/hook"}`)); err != nil {
		t.Errorf("DecodeConfig refused a well-formed URL: %v", err)
	}
}

func TestTypedAdapterCheckConfigAppliesAddressPolicy(t *testing.T) {
	guard := publicOnlyGuard(t)
	adapter := Adapt(NewWebhookSender("https://happydomain.example", guard), guard)

	cfg, err := adapter.DecodeConfig(json.RawMessage(`{"url":"http://127.0.0.1:9000/hook"}`))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}

	err = adapter.CheckConfig(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckConfig accepted a loopback destination")
	}
	if !strings.Contains(err.Error(), "The webhook URL") {
		t.Errorf("refusal does not name the offending field: %v", err)
	}
}

// A transport that dials nothing the user chose needs no guard, and says so by
// returning no destination.
func TestTypedAdapterCheckConfigWithoutDestinations(t *testing.T) {
	adapter := Adapt(&EmailSender{}, nil)

	cfg, err := adapter.DecodeConfig(json.RawMessage(`{"address":"user@example.com"}`))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if err := adapter.CheckConfig(context.Background(), cfg); err != nil {
		t.Errorf("CheckConfig: %v", err)
	}
}

// Registering a dialing sender without a policy must fail closed rather than
// let the channel through unchecked.
func TestTypedAdapterCheckConfigFailsClosedWithoutGuard(t *testing.T) {
	adapter := Adapt(&WebhookSender{}, nil)

	cfg, err := adapter.DecodeConfig(json.RawMessage(`{"url":"https://example.com/hook"}`))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if err := adapter.CheckConfig(context.Background(), cfg); err == nil {
		t.Error("CheckConfig accepted a destination with no guard in hand")
	}
}

// Each type reads its own config: merging the stored config of one type into
// another would hand it the stored secrets, whether or not the new type is
// known.
func TestMergeChannelForUpdateRefusesTypeChange(t *testing.T) {
	r := newTestRegistry(t)
	r.Register(Adapt(NewWebhookSender("https://happydomain.example", publicOnlyGuard(t)), publicOnlyGuard(t)))
	r.Register(Adapt(NewEmailSender(nil, "https://happydomain.example"), nil))
	existing := &happydns.NotificationChannel{
		Type:   ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"stored"}`),
	}

	for _, typ := range []happydns.NotificationChannelType{ChannelTypeEmail, "unregistered"} {
		t.Run(string(typ), func(t *testing.T) {
			incoming := &happydns.NotificationChannel{
				Type:   typ,
				Config: json.RawMessage(`{"url":"https://example.com/hook"}`),
			}

			merged, err := r.MergeChannelForUpdate(existing, incoming)
			if !errors.Is(err, ErrChannelTypeChanged) {
				t.Errorf("MergeChannelForUpdate = %v, want ErrChannelTypeChanged", err)
			}
			if merged != nil {
				t.Errorf("merged = %s, want nothing", merged)
			}
		})
	}
}

func TestEventURL(t *testing.T) {
	for _, tc := range []struct {
		base, path, want string
	}{
		{"https://happydomain.example", "/watchlist/", "https://happydomain.example/watchlist/"},
		{"https://happydomain.example/", "/watchlist/", "https://happydomain.example/watchlist/"},
		{"https://happydomain.example/app", "watchlist", "https://happydomain.example/app/watchlist"},
		{"https://happydomain.example", "", "https://happydomain.example"},
		{"", "/watchlist/", ""},
		{"://bad", "/watchlist/", ""},
	} {
		if got := eventURL(tc.base, &NotificationEvent{Path: tc.path}); got != tc.want {
			t.Errorf("eventURL(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

// registryWithAllTypes registers the three transports, as the application does.
func registryWithAllTypes(t *testing.T) *Registry {
	t.Helper()
	guard := publicOnlyGuard(t)
	r := newTestRegistry(t)
	r.Register(Adapt(NewEmailSender(nil, "https://happydomain.example"), nil))
	r.Register(Adapt(NewWebhookSender("https://happydomain.example", guard), guard))
	r.Register(Adapt(NewUnifiedPushSender("https://happydomain.example", guard), guard))
	return r
}

func TestKnownChannelTypesAreTheRegisteredOnes(t *testing.T) {
	r := registryWithAllTypes(t)
	for _, typ := range KnownChannelTypes() {
		if _, ok := r.Get(typ); !ok {
			t.Errorf("KnownChannelTypes() lists %q, which no sender handles", typ)
		}
	}
	if got, want := len(KnownChannelTypes()), len(r.Types()); got != want {
		t.Errorf("KnownChannelTypes() lists %d types, the registry %d", got, want)
	}
}

func TestRegistryWithoutRestrictionEnablesEveryType(t *testing.T) {
	r := registryWithAllTypes(t)

	want := []happydns.NotificationChannelType{ChannelTypeEmail, ChannelTypeUnifiedPush, ChannelTypeWebhook}
	if got := r.Types(); !slices.Equal(got, want) {
		t.Errorf("Types() = %v, want %v", got, want)
	}
	for _, typ := range want {
		if !r.Enabled(typ) {
			t.Errorf("Enabled(%q) = false, want true", typ)
		}
	}
	if r.Enabled("unregistered") {
		t.Error(`Enabled("unregistered") = true, want false`)
	}
}

func TestRegistryRestrict(t *testing.T) {
	r := registryWithAllTypes(t)
	r.Restrict([]happydns.NotificationChannelType{ChannelTypeEmail})

	if got, want := r.Types(), []happydns.NotificationChannelType{ChannelTypeEmail}; !slices.Equal(got, want) {
		t.Errorf("Types() = %v, want %v", got, want)
	}
	if !r.Enabled(ChannelTypeEmail) {
		t.Error("Enabled(email) = false, want true")
	}
	if r.Enabled(ChannelTypeWebhook) || r.Enabled(ChannelTypeUnifiedPush) {
		t.Error("a type left out of Restrict is still enabled")
	}

	// A disabled sender stays registered, so the secrets of the channels
	// already stored keep being maintained.
	if _, ok := r.Get(ChannelTypeWebhook); !ok {
		t.Error("Restrict unregistered the webhook sender")
	}
}

func TestRegistryRestrictToNothing(t *testing.T) {
	r := registryWithAllTypes(t)
	r.Restrict([]happydns.NotificationChannelType{})

	if got := r.Types(); len(got) != 0 {
		t.Errorf("Types() = %v, want none", got)
	}
}

// Creating or updating a channel goes through CheckIncomingChannel: a disabled
// type must be refused there, before anything is dialed to check it.
func TestCheckIncomingChannelRefusesADisabledType(t *testing.T) {
	r := registryWithAllTypes(t)
	r.Restrict([]happydns.NotificationChannelType{ChannelTypeEmail})

	err := r.CheckIncomingChannel(&happydns.NotificationChannel{
		Type:   ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"https://example.com/hook"}`),
	})
	if !errors.Is(err, ErrChannelTypeDisabled) {
		t.Errorf("CheckIncomingChannel(webhook) = %v, want ErrChannelTypeDisabled", err)
	}

	err = r.CheckIncomingChannel(&happydns.NotificationChannel{
		Type:   ChannelTypeEmail,
		Config: json.RawMessage(`{"address":"user@example.com"}`),
	})
	if err != nil {
		t.Errorf("CheckIncomingChannel(email) = %v, want nil", err)
	}
}
