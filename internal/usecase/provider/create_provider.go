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
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type CreateProviderUsecase struct {
	checker ProviderValidator
	store   ProviderStorage
}

func NewCreateProviderUsecase(store ProviderStorage, checker ProviderValidator) *CreateProviderUsecase {
	return &CreateProviderUsecase{
		checker: checker,
		store:   store,
	}
}

func (uc *CreateProviderUsecase) Create(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse provider: %w", err)
	}

	if err := uc.checker.Validate(provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	provider.Owner = user.Id

	if err := uc.store.CreateProvider(provider); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("failed to save provider: %w", err),
			UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
		}
	}

	return provider, nil
}
