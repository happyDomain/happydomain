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

package user

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type UpdateUser struct {
	store UserStorage
}

func NewUpdateUser(store UserStorage) *UpdateUser {
	return &UpdateUser{store: store}
}

func (uc *UpdateUser) Update(id happydns.Identifier, updateFn func(*happydns.User)) error {
	user, err := uc.store.GetUser(id)
	if err != nil {
		return err
	}

	updateFn(user)

	if !user.Id.Equals(id) {
		return happydns.ValidationError{Msg: "you cannot change the user identifier"}
	}

	if err := uc.store.CreateOrUpdateUser(user); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to update user: %w", err),
			UserMessage: "Sorry, we are currently unable to update your user. Please retry later.",
		}
	}

	return nil
}

func (uc *UpdateUser) UpdateSettings(user *happydns.User, newSettings happydns.UserSettings) error {
	user.Settings = newSettings
	return uc.store.CreateOrUpdateUser(user)
}
