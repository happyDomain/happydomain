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
	"context"
	"encoding/json"
	"fmt"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

type providerSettingsUsecase struct {
	config          *happydns.Options
	providerService happydns.ProviderUsecase
}

func NewProviderSettingsUsecase(cfg *happydns.Options, ps happydns.ProviderUsecase) happydns.ProviderSettingsUsecase {
	return &providerSettingsUsecase{
		config:          cfg,
		providerService: ps,
	}
}

func (psu *providerSettingsUsecase) NextProviderSettingsState(ctx context.Context, state *happydns.ProviderSettingsState, pType string, user *happydns.User) (*happydns.Provider, *happydns.ProviderSettingsResponse, error) {
	fu := NewFormUsecase(psu.config)

	form, p, err := forms.DoSettingState(fu, &state.FormState, state.ProviderBody, forms.GenDefaultSettingsForm)

	if err != nil {
		if err != happydns.DoneForm {
			return nil, nil, happydns.ValidationError{Msg: err.Error()}
		} else if psu.config.DisableProviders {
			return nil, nil, happydns.ForbiddenError{Msg: "cannot change provider settings as DisableProviders parameter is set."}
		}

		providerJSON, err := json.Marshal(state.ProviderBody)
		if err != nil {
			return nil, nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to marshal provider body: %w", err),
				UserMessage: happydns.TryAgainErr,
			}
		}

		msg := &happydns.ProviderMessage{
			ProviderMeta: happydns.ProviderMeta{
				Type:    pType,
				Comment: state.Name,
			},
			Provider: providerJSON,
		}

		if state.Id == nil {
			// Create a new Provider via the service layer
			provider, err := psu.providerService.CreateProvider(ctx, user, msg)
			if err != nil {
				return nil, nil, err
			}

			return provider, nil, nil
		} else {
			// Update an existing Provider via the service layer
			err := psu.providerService.UpdateProviderFromMessage(ctx, *state.Id, user, msg)
			if err != nil {
				return nil, nil, err
			}

			provider, err := psu.providerService.GetUserProvider(ctx, user, *state.Id)
			if err != nil {
				return nil, nil, err
			}

			return provider, nil, nil
		}
	}

	return nil, &happydns.ProviderSettingsResponse{
		Form:   form,
		Values: p,
	}, nil
}
