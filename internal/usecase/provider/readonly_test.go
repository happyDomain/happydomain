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

package provider_test

import (
	"errors"
	"testing"

	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

// transferOnlyMessage is a DDNS provider message whose settings allow updates
// or not, and whose meta claims ReadOnly as given, to check the server ignores
// that claim.
func transferOnlyMessage(transferOnly bool, claimedReadOnly bool) *happydns.ProviderMessage {
	body := `{"server":"127.0.0.1","algorithm":"hmac-sha256"}`
	if transferOnly {
		body = `{"server":"127.0.0.1","algorithm":"hmac-sha256","transfer_only":true}`
	}

	return &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type:     "DDNSServer",
			ReadOnly: claimedReadOnly,
		},
		Provider: []byte(body),
	}
}

func Test_CreateProvider_SetsReadOnlyFromSettings(t *testing.T) {
	providerService, db := newTestService(t)
	user := createTestUser(t, db, "readonly@example.com")

	for _, transferOnly := range []bool{false, true} {
		p, err := providerService.CreateProvider(ctx, user, transferOnlyMessage(transferOnly, !transferOnly))
		if err != nil {
			t.Fatalf("CreateProvider(transfer only: %t) = %v", transferOnly, err)
		}

		meta, err := providerService.GetUserProviderMeta(ctx, user, p.Id)
		if err != nil {
			t.Fatalf("GetUserProviderMeta() = %v", err)
		}
		if meta.ReadOnly != transferOnly {
			t.Errorf("stored ReadOnly = %t, want %t, whatever the client claimed", meta.ReadOnly, transferOnly)
		}
	}
}

func Test_UpdateProvider_SetsReadOnlyFromSettings(t *testing.T) {
	providerService, db := newTestService(t)
	user := createTestUser(t, db, "readonly-update@example.com")

	created, err := providerService.CreateProvider(ctx, user, transferOnlyMessage(true, true))
	if err != nil {
		t.Fatalf("CreateProvider() = %v", err)
	}

	// Back to read-write, with a key: the client still claims read-only.
	msg := createTestProviderMessage(t, "DDNSServer", "")
	msg.ReadOnly = true
	if _, err := providerService.UpdateProviderFromMessage(ctx, created.Id, user, msg); err != nil {
		t.Fatalf("UpdateProviderFromMessage() = %v", err)
	}

	meta, err := providerService.GetUserProviderMeta(ctx, user, created.Id)
	if err != nil {
		t.Fatalf("GetUserProviderMeta() = %v", err)
	}
	if meta.ReadOnly {
		t.Error("stored ReadOnly = true, want false once the settings allow updates")
	}
}

func Test_UpdateProvider_SetsReadOnlyWhenSwitchedToTransferOnly(t *testing.T) {
	providerService, db := newTestService(t)
	user := createTestUser(t, db, "readonly-switch@example.com")

	created, err := providerService.CreateProvider(ctx, user, transferOnlyMessage(false, false))
	if err != nil {
		t.Fatalf("CreateProvider() = %v", err)
	}
	if _, err := providerService.UpdateProviderFromMessage(ctx, created.Id, user, transferOnlyMessage(true, false)); err != nil {
		t.Fatalf("UpdateProviderFromMessage() = %v", err)
	}

	meta, err := providerService.GetUserProviderMeta(ctx, user, created.Id)
	if err != nil {
		t.Fatalf("GetUserProviderMeta() = %v", err)
	}
	if !meta.ReadOnly {
		t.Error("stored ReadOnly = false, want true once the settings forbid updates")
	}
}

func Test_CreateDomainOnProvider_RefusedWhenReadOnly(t *testing.T) {
	providerService, _ := newTestService(t)

	p := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "DDNSServer"},
		Provider:     &providers.DDNSServer{TransferOnly: true},
	}

	err := providerService.CreateDomainOnProvider(ctx, p, "example.com.")
	var forbidden happydns.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Errorf("CreateDomainOnProvider() = %v, want a ForbiddenError", err)
	}
}

func Test_ParseProvider_SetsReadOnlyFromSettings(t *testing.T) {
	for _, transferOnly := range []bool{false, true} {
		p, err := provider.ParseProvider(transferOnlyMessage(transferOnly, !transferOnly))
		if err != nil {
			t.Fatalf("ParseProvider(transfer only: %t) = %v", transferOnly, err)
		}
		if p.ReadOnly != transferOnly {
			t.Errorf("ReadOnly = %t, want %t, whatever the message claimed", p.ReadOnly, transferOnly)
		}
	}
}
