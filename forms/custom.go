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

package forms // import "git.happydns.org/happyDomain/forms"

import ()

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
