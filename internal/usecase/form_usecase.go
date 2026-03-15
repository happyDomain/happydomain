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
	"git.happydns.org/happyDomain/model"
)

// formUsecase implements happydns.FormUsecase, providing form-related helpers
// such as base URL generation used when building dynamic forms.
type formUsecase struct {
	config *happydns.Options
}

// NewFormUsecase returns a FormUsecase backed by the given application options.
func NewFormUsecase(cfg *happydns.Options) happydns.FormUsecase {
	return &formUsecase{
		config: cfg,
	}
}

// GetBaseURL returns the application's base URL, used when constructing
// absolute links inside forms (e.g. OAuth redirect URIs).
func (fu *formUsecase) GetBaseURL() string {
	return fu.config.GetBaseURL()
}
