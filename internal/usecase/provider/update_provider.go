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

type UpdateProviderUsecase struct {
	checker     ProviderValidator
	getProvider *GetProviderUsecase
	store       ProviderStorage
}

func NewUpdateProviderUsecase(store ProviderStorage, getProvider *GetProviderUsecase, checker ProviderValidator) *UpdateProviderUsecase {
	return &UpdateProviderUsecase{
		checker:     checker,
		getProvider: getProvider,
		store:       store,
	}
}

func (uc *UpdateProviderUsecase) Update(providerid happydns.Identifier, user *happydns.User, upd func(*happydns.Provider)) error {
	provider, err := uc.getProvider.Get(user, providerid)
	if err != nil {
		return err
	}

	upd(provider)

	if !provider.Id.Equals(providerid) {
		return happydns.ValidationError{Msg: "you cannot change the provider identifier"}
	}

	err = uc.checker.Validate(provider)
	if err != nil {
		return happydns.ValidationError{Msg: fmt.Sprintf("unable to validate provider attributes: %s", err.Error())}
	}

	err = uc.store.UpdateProvider(provider)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateProvider in UpdateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to update your provider. Please retry later.",
		}
	}

	return nil
}

func (uc *UpdateProviderUsecase) FromMessage(providerid happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	newprovider, err := ParseProvider(p)
	if err != nil {
		return err
	}

	return uc.Update(providerid, user, func(provider *happydns.Provider) {
		*provider = *newprovider
	})
}
