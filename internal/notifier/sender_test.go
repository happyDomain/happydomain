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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/netguard"
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
