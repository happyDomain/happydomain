// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package happydns

import (
	"errors"
)

var (
	// DoneForm is the error raised when there is no more step to display, and edition is OK.
	DoneForm = errors.New("Done")

	// CancelForm is the error raised when there is no more step to display and should redirect to the previous page.
	CancelForm = errors.New("Cancel")
)

// CustomForm is used to create a form with several steps when creating or updating provider's settings.
type CustomForm struct {
	// BeforeText is the text presented before the fields.
	BeforeText string `json:"beforeText,omitempty"`

	// SideText is displayed in the sidebar, after any already existing text. When a sidebar is avaiable.
	SideText string `json:"sideText,omitempty"`

	// AfterText is the text presented after the fields and before the buttons
	AfterText string `json:"afterText,omitempty"`

	// Fields are the fields presented to the User.
	Fields []*Field `json:"fields"`

	// NextButtonText is the next button content.
	NextButtonText string `json:"nextButtonText,omitempty"`

	// NextEditButtonText is the next button content when updating the settings (if not set, NextButtonText is used instead).
	NextEditButtonText string `json:"nextEditButtonText,omitempty"`

	// PreviousButtonText is previous/cancel button content.
	PreviousButtonText string `json:"previousButtonText,omitempty"`

	// PreviousEditButtonText is the previous/cancel button content when updating the settings (if not set, NextButtonText is used instead).
	PreviousEditButtonText string `json:"previousEditButtonText,omitempty"`

	// NextButtonLink is the target of the next button, exclusive with NextButtonState.
	NextButtonLink string `json:"nextButtonLink,omitempty"`

	// NextButtonState is the step number asked when submiting the form.
	NextButtonState int32 `json:"nextButtonState,omitempty"`

	// PreviousButtonState is the step number to go when hitting the previous button.
	PreviousButtonState int32 `json:"previousButtonState,omitempty"`
}

// Field
type Field struct {
	// Id is the field identifier.
	Id string `json:"id"`

	// Type is the string representation of the field's type.
	Type string `json:"type"`

	// Label is the title given to the field, displayed as <label> tag on the interface.
	Label string `json:"label,omitempty"`

	// Placeholder is the placeholder attribute of the corresponding <input> tag.
	Placeholder string `json:"placeholder,omitempty"`

	// Default is the preselected value or the default value in case the field is not filled by the user.
	Default any `json:"default,omitempty"`

	// Choices holds the differents choices shown to the user in <select> tag.
	Choices []string `json:"choices,omitempty"`

	// Required indicates whether the field has to be filled or not.
	Required bool `json:"required,omitempty"`

	// Secret indicates if the field contains sensitive information such as API key, in order to hide
	// the field when not needed. When typing, it doesn't hide characters like in password input.
	Secret bool `json:"secret,omitempty"`

	// Hide indicates if the field should be hidden to the user.
	Hide bool `json:"hide,omitempty"`

	// Textarea indicates that a large field is expected.
	Textarea bool `json:"textarea,omitempty"`

	// Description stores an helpfull sentence describing the field.
	Description string `json:"description,omitempty"`
}

type FormState struct {
	// Id for an already existing element.
	Id *Identifier `json:"_id,omitempty" swaggertype:"string"`

	// User defined name of the element.
	Name string `json:"_comment"`

	// State is the desired form to shows next (starting at 0).
	State int32 `json:"state"`

	// Recall is the identifier for a saved FormState you want to retrieve.
	Recall string `json:"recall,omitempty"`
}

// GenRecallID
type GenRecallID func() string

// CustomSettingsForm are functions to declare when we want to display a custom user experience when asking for Source's settings.
type CustomSettingsForm interface {
	// DisplaySettingsForm generates the CustomForm corresponding to the asked target state.
	DisplaySettingsForm(int32, GenRecallID, FormUsecase) (*CustomForm, map[string]any, error)
}

type FormUsecase interface {
	GetBaseURL() string
}
