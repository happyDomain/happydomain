// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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

package provider

import (
	"context"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// ProviderValidator verifies that a provider configuration is functional before it is persisted.
type ProviderValidator interface {
	Validate(context.Context, *happydns.Provider) error
}

// DefaultProviderValidator instantiates the provider and, when zone listing is supported, performs a live check.
type DefaultProviderValidator struct {
	instantiator
}

func NewValidator(guard *netguard.Guard, secrets *secret.Manager) *DefaultProviderValidator {
	return &DefaultProviderValidator{instantiator{guard: guard, secrets: secrets}}
}

func (v *DefaultProviderValidator) Validate(ctx context.Context, p *happydns.Provider) error {
	instance, err := v.instantiate(ctx, p)
	if err != nil {
		return err
	}

	if instance.CanListZones() {
		_, err = instance.ListZones()
	}

	return err
}
