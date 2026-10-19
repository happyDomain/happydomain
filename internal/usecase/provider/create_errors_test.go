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
	"context"
	"encoding/json"
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// failingValidator refuses every provider, as a provider that cannot be
// reached or that rejects its credentials is.
type failingValidator struct{}

func (failingValidator) Validate(_ context.Context, _ *happydns.Provider) error {
	return errors.New("unable to reach the provider")
}

// A provider the user got wrong is their mistake to fix, not a failure of the
// server: the API has to answer 400, as it does on update.
func Test_CreateProvider_ClientErrorsAreValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		validator *failingValidator
		msg       *happydns.ProviderMessage
	}{
		{
			name:      "settings refused by the validator",
			validator: &failingValidator{},
			msg:       createTestProviderMessage(t, "DDNSServer", "Refused"),
		},
		{
			name: "an unknown type",
			msg: &happydns.ProviderMessage{
				ProviderMeta: happydns.ProviderMeta{Type: "NonExistentProvider"},
				Provider:     json.RawMessage(`{}`),
			},
		},
		{
			name: "settings that do not decode",
			msg: &happydns.ProviderMessage{
				ProviderMeta: happydns.ProviderMeta{Type: "DDNSServer"},
				Provider:     json.RawMessage(`{"server": 42}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerService, db := newTestService(t)
			if tt.validator != nil {
				providerService, db = newTestServiceWithValidator(t, tt.validator)
			}
			user := createTestUser(t, db, "invalid@example.com")

			_, err := providerService.CreateProvider(ctx, user, tt.msg)
			var validation happydns.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("CreateProvider() = %T %v, want a ValidationError", err, err)
			}

			providers, err := providerService.ListUserProviders(ctx, user)
			if err != nil {
				t.Fatalf("ListUserProviders() = %v", err)
			}
			if len(providers) != 0 {
				t.Errorf("%d providers stored, want none", len(providers))
			}
		})
	}
}
