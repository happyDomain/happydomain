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

import (
	"errors"

	"git.happydns.org/happyDomain/config"
	"git.happydns.org/happyDomain/model"
)

// GenRecallID
type GenRecallID func() string

// CustomSettingsForm are functions to declare when we want to display a custom user experience when asking for Source's settings.
type CustomSettingsForm interface {
	// DisplaySettingsForm generates the CustomForm corresponding to the asked target state.
	DisplaySettingsForm(int32, *config.Options, *happydns.Session, GenRecallID) (*CustomForm, map[string]interface{}, error)
}

var (
	// DoneForm is the error raised when there is no more step to display, and edition is OK.
	DoneForm = errors.New("Done")

	// CancelForm is the error raised when there is no more step to display and should redirect to the previous page.
	CancelForm = errors.New("Cancel")
)
