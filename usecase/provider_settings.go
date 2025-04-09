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
	"net/http"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type providerSettingsUsecase struct {
	config *config.Options
	store  storage.Storage
}

func NewProviderSettingsUsecase(cfg *config.Options, store storage.Storage) happydns.ProviderSettingsUsecase {
	return &providerSettingsUsecase{
		config: cfg,
		store:  store,
	}
}

func (psu *providerSettingsUsecase) NextProviderSettingsState(pbody happydns.ProviderBody, user *happydns.User, state *happydns.ProviderSettingsState) (*happydns.Provider, *happydns.ProviderSettingsResponse, error) {
	fu := NewFormUsecase(psu.config)

	form, p, err := forms.DoSettingState(fu, &state.FormState, pbody, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != happydns.DoneForm {
			return nil, nil, happydns.InternalError{
				Err:        err,
				HTTPStatus: http.StatusBadRequest,
			}
		} else if psu.config.DisableProviders {
			return nil, nil, happydns.InternalError{
				Err:        fmt.Errorf("Cannot change provider settings as DisableProviders parameter is set."),
				HTTPStatus: http.StatusForbidden,
			}
		}

		p, err := pbody.InstantiateProvider()
		if err != nil {
			return nil, nil, happydns.InternalError{
				Err:        fmt.Errorf("unable to instantiate provider: %w", err),
				HTTPStatus: http.StatusBadRequest,
			}
		}

		if sr, ok := p.(happydns.ZoneLister); ok {
			if _, err = sr.ListZones(); err != nil {
				return nil, nil, happydns.InternalError{
					Err:        fmt.Errorf("unable to list provider's zones: %w", err),
					HTTPStatus: http.StatusBadRequest,
				}
			}
		}

		if state.Id == nil {
			// Create a new Provider
			p, err := psu.store.CreateProvider(user, pbody, state.Name)
			if err != nil {
				return nil, nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to CreateProvider: %w", err),
					UserMessage: happydns.TryAgainErr,
				}
			}

			return p, nil, nil
		} else {
			// Update an existing Provider
			p, err := psu.store.GetProvider(user, *state.Id)
			if err != nil {
				return nil, nil, happydns.InternalError{
					Err:        fmt.Errorf("unable to retrieve the original provider: %w", err),
					HTTPStatus: http.StatusNotFound,
				}
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
