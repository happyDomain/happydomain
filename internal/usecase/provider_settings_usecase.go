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

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

type providerSettingsUsecase struct {
	config          *config.Options
	providerService happydns.ProviderUsecase
	store           provider.ProviderStorage
}

func NewProviderSettingsUsecase(cfg *config.Options, ps happydns.ProviderUsecase, store provider.ProviderStorage) happydns.ProviderSettingsUsecase {
	return &providerSettingsUsecase{
		config:          cfg,
		providerService: ps,
		store:           store,
	}
}

func (psu *providerSettingsUsecase) NextProviderSettingsState(state *happydns.ProviderSettingsState, pType string, user *happydns.User) (*happydns.Provider, *happydns.ProviderSettingsResponse, error) {
	fu := NewFormUsecase(psu.config)

	form, p, err := forms.DoSettingState(fu, &state.FormState, state.ProviderBody, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != happydns.DoneForm {
			return nil, nil, happydns.ValidationError{err.Error()}
		} else if psu.config.DisableProviders {
			return nil, nil, happydns.ForbiddenError{"cannot change provider settings as DisableProviders parameter is set."}
		}

		p, err := state.ProviderBody.InstantiateProvider()
		if err != nil {
			return nil, nil, happydns.ValidationError{fmt.Sprintf("unable to instantiate provider: %s", err.Error())}
		}

		if sr, ok := p.(happydns.ZoneLister); ok {
			if _, err = sr.ListZones(); err != nil {
				return nil, nil, happydns.ValidationError{fmt.Sprintf("unable to list provider's zones: %s", err.Error())}
			}
		}

		if state.Id == nil {
			provider := &happydns.Provider{
				Provider: state.ProviderBody,
				ProviderMeta: happydns.ProviderMeta{
					Type:    pType,
					Owner:   user.Id,
					Comment: state.Name,
				},
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
			p, err := psu.providerService.GetUserProvider(user, *state.Id)
			if err != nil {
				return nil, nil, happydns.NotFoundError{fmt.Sprintf("unable to retrieve the original provider: %s", err.Error())}
			}

			newp := &happydns.Provider{
				ProviderMeta: p.ProviderMeta,
				Provider:     state.ProviderBody,
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
