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
	"errors"
	"fmt"
	"slices"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/model"
)

const (
	gandiAuthPAT    = "Personal Access Token"
	gandiAuthAPIKey = "API key (legacy)"
)

// gandiCredentialFields lists the form fields used by the given
// authentication method.
func gandiCredentialFields(method string) []string {
	if method == gandiAuthAPIKey {
		return []string{"api_key", "sharing_id"}
	}
	return []string{"token"}
}

// hasCredentials tells whether a token or an API key is set, that is
// whether an existing provider is being edited.
func (s *GandiAPI) hasCredentials() bool {
	return !s.Token.IsEmpty() || !s.APIKey.IsEmpty()
}

// gandiEditForm shows every credential field at once, to edit an existing
// provider.
func (s *GandiAPI) gandiEditForm() *happydns.CustomForm {
	form := forms.GenDefaultSettingsForm(s)
	form.Fields = slices.DeleteFunc(form.Fields, func(f *happydns.Field) bool {
		return f.Id == "auth_method"
	})
	return form
}

// gandiMethodForm asks which authentication method is used, as the first
// step of a provider creation.
func (s *GandiAPI) gandiMethodForm() *happydns.CustomForm {
	form := forms.GenDefaultSettingsForm(nil)
	form.Fields = []*happydns.Field{
		{
			Id:          "auth_method",
			Type:        "string",
			Label:       "Authentication method",
			Default:     gandiAuthPAT,
			Choices:     []string{gandiAuthPAT, gandiAuthAPIKey},
			Required:    true,
			Description: "Gandi is replacing API keys with Personal Access Tokens. Choose 'API key (legacy)' only if you still have an API key.",
		},
	}
	form.NextButtonText = "common.next"
	return form
}

// gandiCredentialsForm shows the credentials of the chosen method, as the
// second step of a provider creation.
func (s *GandiAPI) gandiCredentialsForm() (*happydns.CustomForm, map[string]any) {
	keep := gandiCredentialFields(s.AuthMethod)

	form := forms.GenDefaultSettingsForm(s)
	form.Fields = slices.DeleteFunc(form.Fields, func(f *happydns.Field) bool {
		return !slices.Contains(keep, f.Id)
	})
	form.NextButtonState = 2
	form.PreviousButtonText = "common.previous"
	form.PreviousButtonState = 0
	return form, map[string]any{"auth_method": s.AuthMethod}
}

// DisplaySettingsForm drives two paths, told apart by whether credentials
// are already set (an existing provider holds a token or an API key):
//
//   - creation: state 0 asks the authentication method, state 1 shows the
//     credentials of that method, state 2 validates them;
//   - edition: state 0 shows every credential field in a single form,
//     state 1 validates it.
//
// Both paths end with the same validation, below the switch.
func (s *GandiAPI) DisplaySettingsForm(state int32, _ happydns.GenRecallID, _ happydns.FormUsecase) (*happydns.CustomForm, map[string]any, error) {
	switch state {
	case 0:
		// Edition: an existing provider comes with its credentials.
		if s.hasCredentials() {
			return s.gandiEditForm(), nil, nil
		}
		// Creation: nothing is set yet, start with the method.
		return s.gandiMethodForm(), nil, nil

	case 1:
		// Creation: the method has just been chosen (AuthMethod is set)
		// and no credential is filled yet, show the ones of that method.
		// This is also reached when an edition empties every credential:
		// the user then enters new ones, as on creation.
		if !s.hasCredentials() {
			form, values := s.gandiCredentialsForm()
			return form, values, nil
		}
		// Edition: the edit form was submitted. Every field is kept as
		// is, but only one method may be filled in.
		if !s.Token.IsEmpty() && !s.APIKey.IsEmpty() {
			return nil, nil, errors.New("fill in either the Personal Access Token or the API Key, not both")
		}

	case 2:
		// Creation: the credentials form was submitted, the credential
		// of the chosen method must be filled in.
		if s.AuthMethod == gandiAuthAPIKey && s.APIKey.IsEmpty() {
			return nil, nil, fmt.Errorf("field %q is required", "API Key")
		}
		if s.AuthMethod != gandiAuthAPIKey && s.Token.IsEmpty() {
			return nil, nil, fmt.Errorf("field %q is required", "Personal Access Token")
		}

	default:
		return nil, nil, happydns.CancelForm
	}

	// End of both paths: edition (state 1) and creation (state 2).
	if err := forms.ValidateStructValues(s); err != nil {
		return nil, nil, err
	}

	// AuthMethod only carries the choice from state 0 to state 2 of the
	// creation: it is not stored.
	s.AuthMethod = ""

	return nil, nil, happydns.DoneForm
}
