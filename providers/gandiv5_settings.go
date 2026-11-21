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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"fmt"
	"slices"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

const (
	gandiAuthPAT    = "Personal Access Token"
	gandiAuthAPIKey = "API key (legacy)"
)

// authMethod returns the chosen authentication method. Providers saved
// before the choice existed have none: they hold an API key, unless a token
// was set.
func (s *GandiAPI) authMethod() string {
	if s.AuthMethod != "" {
		return s.AuthMethod
	}
	if !s.APIKey.IsEmpty() {
		return gandiAuthAPIKey
	}
	return gandiAuthPAT
}

// gandiCredentialFields lists the form fields used by the given
// authentication method.
func gandiCredentialFields(method string) []string {
	if method == gandiAuthAPIKey {
		return []string{"api_key", "sharing_id"}
	}
	return []string{"token"}
}

func (s *GandiAPI) DisplaySettingsForm(state int32, _ happydns.GenRecallID, _ happydns.FormUsecase) (*happydns.CustomForm, map[string]any, error) {
	switch state {
	case 0:
		form := forms.GenDefaultSettingsForm(nil)
		form.Fields = []*happydns.Field{
			{
				Id:          "auth_method",
				Type:        "string",
				Label:       "Authentication method",
				Default:     s.authMethod(),
				Choices:     []string{gandiAuthPAT, gandiAuthAPIKey},
				Required:    true,
				Description: "Gandi is replacing API keys with Personal Access Tokens. Choose 'API key (legacy)' only if you still have an API key.",
			},
		}
		form.NextButtonText = "common.next"
		form.NextEditButtonText = "common.next"
		return form, nil, nil

	case 1:
		method := s.authMethod()
		keep := gandiCredentialFields(method)

		form := forms.GenDefaultSettingsForm(s)
		form.Fields = slices.DeleteFunc(form.Fields, func(f *happydns.Field) bool {
			return !slices.Contains(keep, f.Id)
		})
		form.NextButtonState = 2
		form.PreviousButtonText = "common.previous"
		form.PreviousEditButtonText = "common.previous"
		form.PreviousButtonState = 0
		return form, map[string]any{"auth_method": method}, nil

	case 2:
		if err := forms.ValidateStructValues(s); err != nil {
			return nil, nil, err
		}

		// Keep only the credentials of the chosen method, so that
		// switching method does not leave a stale secret behind.
		switch s.authMethod() {
		case gandiAuthAPIKey:
			if s.APIKey.IsEmpty() {
				return nil, nil, fmt.Errorf("field %q is required", "API Key")
			}
			s.Token = happydns.Secret{}
		default:
			if s.Token.IsEmpty() {
				return nil, nil, fmt.Errorf("field %q is required", "Personal Access Token")
			}
			s.APIKey = happydns.Secret{}
			s.SharingID = ""
		}
		s.AuthMethod = s.authMethod()

		return nil, nil, happydns.DoneForm

	default:
		return nil, nil, happydns.CancelForm
	}
}
