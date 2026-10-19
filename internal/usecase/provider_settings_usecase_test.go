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

package usecase_test

import (
	"context"
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// SettingsSecretProvider holds a happydns.Secret, as every provider will.
type SettingsSecretProvider struct {
	ApiKey happydns.Secret `json:"apikey" happydomain:"required"`
}

func (*SettingsSecretProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, nil
}

func init() {
	providerReg.RegisterProvider(func() happydns.ProviderBody { return &SettingsSecretProvider{} }, happydns.ProviderInfos{Name: "settings secret test"})
}

type acceptAll struct{}

func (acceptAll) Validate(context.Context, *happydns.Provider) error { return nil }

// The controller presets the body, but a client sending "Provider": null
// clears it: that is a bad request, not an internal error.
func TestProviderSettingsRefusesNullBody(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}

	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "settings@example.com"}
	uc := usecase.NewProviderSettingsUsecase(&happydns.Options{}, providerUC.NewService(db, acceptAll{}, nil, secrets))

	state := &happydns.ProviderSettingsState{
		FormState:    happydns.FormState{Name: "wizard", State: 1},
		ProviderBody: nil,
	}

	_, _, err = uc.NextProviderSettingsState(context.Background(), state, "SettingsSecretProvider", user)
	if _, ok := err.(happydns.ValidationError); !ok {
		t.Fatalf("NextProviderSettingsState = %v (%T), want a ValidationError", err, err)
	}

	providers, err := db.ListProviders(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 0 {
		t.Errorf("%d providers stored from a null body", len(providers))
	}
}

func TestProviderSettingsCreatesProviderWithSecret(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}

	user := &happydns.User{Id: happydns.Identifier{0x01}, Email: "settings@example.com"}
	uc := usecase.NewProviderSettingsUsecase(&happydns.Options{}, providerUC.NewService(db, acceptAll{}, nil, secrets))

	// What the controller decodes from the wizard's last step.
	state := &happydns.ProviderSettingsState{
		FormState:    happydns.FormState{Name: "wizard", State: 1},
		ProviderBody: &SettingsSecretProvider{ApiKey: happydns.NewSecret("typed-in-wizard")},
	}

	p, _, err := uc.NextProviderSettingsState(context.Background(), state, "SettingsSecretProvider", user)
	if err != nil {
		t.Fatalf("NextProviderSettingsState: %v", err)
	}
	if p == nil {
		t.Fatal("no provider created")
	}

	stored, err := db.GetProvider(p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.Provider) != `{"apikey":"typed-in-wizard"}` {
		t.Errorf("stored body = %s", stored.Provider)
	}
}
