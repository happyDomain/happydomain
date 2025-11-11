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

func (pss *ProviderSettingsState) FormState() (fs FormState) {
	fs.UnderscoreId = pss.UnderscoreId
	fs.Comment = pss.Comment
	fs.Recall = pss.Recall
	fs.State = pss.State

	return
}

// GenRecallID
type GenRecallID func() string

// CustomSettingsForm are functions to declare when we want to display a custom user experience when asking for Source's settings.
type CustomSettingsForm interface {
	// DisplaySettingsForm generates the CustomForm corresponding to the asked target state.
	DisplaySettingsForm(int32, GenRecallID, FormUsecase) (*CustomForm, map[string]interface{}, error)
}

type FormUsecase interface {
	GetBaseURL() string
}
