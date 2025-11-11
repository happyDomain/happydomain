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

package usecase

import (
	"fmt"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

type providerSettingsUsecase struct {
	config          *happydns.Options
	providerService happydns.ProviderUsecase
	store           provider.ProviderStorage
}

func NewProviderSettingsUsecase(cfg *happydns.Options, ps happydns.ProviderUsecase, store provider.ProviderStorage) happydns.ProviderSettingsUsecase {
	return &providerSettingsUsecase{
		config:          cfg,
		providerService: ps,
		store:           store,
	}
}

func (psu *providerSettingsUsecase) NextProviderSettingsState(state *happydns.ProviderSettingsState, pType string, user *happydns.User) (*happydns.Provider, *happydns.ProviderSettingsResponse, error) {
	fu := NewFormUsecase(psu.config)

	fs := state.FormState()
	form, p, err := forms.DoSettingState(fu, &fs, state.Provider, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != happydns.DoneForm {
			return nil, nil, happydns.ValidationError{Msg: err.Error()}
		} else if psu.config.DisableProviders {
			return nil, nil, happydns.ForbiddenError{Msg: "cannot change provider settings as DisableProviders parameter is set."}
		}

		p, err := state.Provider.InstantiateProvider()
		if err != nil {
			return nil, nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to instantiate provider: %s", err.Error())}
		}

		if p.CanListZones() {
			if _, err = p.ListZones(); err != nil {
				return nil, nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to list provider's zones: %s", err.Error())}
			}
		}

		if state.UnderscoreId == nil {
			provider := &happydns.Provider{
				Provider:          state.Provider,
				Type:              pType,
				UnderscoreOwnerid: user.Id,
				Comment:           state.Comment,
			}
			// Create a new Provider
			err = psu.store.CreateProvider(provider)
			if err != nil {
				return nil, nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to CreateProvider: %w", err),
					UserMessage: happydns.TryAgainErr,
				}
			}

			return provider, nil, nil
		} else {
			// Update an existing Provider
			p, err := psu.providerService.GetUserProvider(user, state.UnderscoreId)
			if err != nil {
				return nil, nil, happydns.NotFoundError{Msg: fmt.Sprintf("unable to retrieve the original provider: %s", err.Error())}
			}

			newp := &happydns.Provider{
				UnderscoreId:      p.UnderscoreId,
				UnderscoreOwnerid: p.UnderscoreOwnerid,
				Type:              p.Type,
				Comment:           p.Comment,
				Provider:          state.Provider,
			}
			err = psu.store.UpdateProvider(newp)
			if err != nil {
				return nil, nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to UpdateProvider: %w", err),
					UserMessage: happydns.TryAgainErr,
				}
			}

			return newp, nil, nil
		}
	}

	return nil, &happydns.ProviderSettingsResponse{
		Form:   form,
		Values: p,
	}, nil
}
