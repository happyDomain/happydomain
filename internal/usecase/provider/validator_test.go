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

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// spyProviderBody records whether it was ever instantiated. Whether the
// endpoint check runs before that happens is the security property under test:
// several backends dial while being constructed, and the credentials are
// already gone by then.
type spyProviderBody struct {
	ApiUrl string `json:"apiurl,omitempty" happydomain:"label=API Server Endpoint,endpoint"`

	instantiated *bool
}

func (s *spyProviderBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	*s.instantiated = true
	return nil, errors.New("instantiation not implemented in this test")
}

func TestValidatorChecksEndpointBeforeInstantiating(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		apiURL      string
		wantErr     bool
		wantReached bool
	}{
		{
			name:    "loopback is refused, and nothing is dialed",
			apiURL:  "http://127.0.0.1:8081",
			wantErr: true,
		},
		{
			name:    "the metadata endpoint is refused",
			apiURL:  "http://169.254.169.254/latest/meta-data/",
			wantErr: true,
		},
		{
			name:    "a LAN address is refused by default",
			apiURL:  "https://192.168.1.1",
			wantErr: true,
		},
		{
			name:        "an allowed loopback reaches the provider",
			allowed:     []string{"127.0.0.1"},
			apiURL:      "http://127.0.0.1:8081",
			wantErr:     true, // the spy always fails to instantiate
			wantReached: true,
		},
		{
			name:        "a public address reaches the provider",
			apiURL:      "https://1.1.1.1",
			wantErr:     true,
			wantReached: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := netguard.New("test", "-test", tt.allowed)
			if err != nil {
				t.Fatalf("netguard.New(%v) => %v", tt.allowed, err)
			}

			var reached bool
			p := &happydns.Provider{
				Provider: &spyProviderBody{ApiUrl: tt.apiURL, instantiated: &reached},
			}

			err = provider.NewValidator(guard).Validate(t.Context(), p)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if reached != tt.wantReached {
				t.Errorf("provider instantiated = %v, want %v", reached, tt.wantReached)
			}
		})
	}
}

// ddnsLikeBody mirrors providers.DDNSServer: leaving Server empty means
// 127.0.0.1 to the backend, which the tag has to spell out.
type ddnsLikeBody struct {
	Server string `json:"server,omitempty" happydomain:"label=Server,endpoint=127.0.0.1"`

	instantiated *bool
}

func (s *ddnsLikeBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	*s.instantiated = true
	return nil, errors.New("instantiation not implemented in this test")
}

func TestValidatorAppliesTheEndpointDefault(t *testing.T) {
	guard, err := netguard.New("test", "-test", nil)
	if err != nil {
		t.Fatalf("netguard.New() => %v", err)
	}

	t.Run("an empty field carrying a loopback default is refused", func(t *testing.T) {
		var reached bool
		p := &happydns.Provider{Provider: &ddnsLikeBody{instantiated: &reached}}

		if err := provider.NewValidator(guard).Validate(t.Context(), p); err == nil {
			t.Fatal("Validate() = nil: an empty field that means 127.0.0.1 must be checked, not skipped")
		}
		if reached {
			t.Error("the provider was instantiated despite its default pointing at loopback")
		}
	})

	t.Run("an empty field with no default is skipped", func(t *testing.T) {
		var reached bool
		p := &happydns.Provider{Provider: &spyProviderBody{instantiated: &reached}}

		if err := provider.NewValidator(guard).Validate(t.Context(), p); err == nil {
			t.Fatal("Validate() = nil, want the spy's instantiation error")
		}
		if !reached {
			t.Error("an unfilled endpoint with no default must not block the provider")
		}
	})
}
