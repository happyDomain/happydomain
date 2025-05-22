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

package authuser

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// DeleteAuthUserUsecase represents the use case for deleting an authenticated user and their sessions.
type DeleteAuthUserUsecase struct {
	store             AuthUserStorage
	closeUserSessions happydns.SessionCloserUsecase
}

// NewDeleteAuthUserUsecase creates a new instance of DeleteAuthUserUsecase.
func NewDeleteAuthUserUsecase(store AuthUserStorage, closeUserSessions happydns.SessionCloserUsecase) *DeleteAuthUserUsecase {
	return &DeleteAuthUserUsecase{
		store:             store,
		closeUserSessions: closeUserSessions,
	}
}

// Do deletes an authenticated user from the system, ensuring their sessions are also removed.
// It first verifies the current password, then removes the user and their associated sessions from the storage.
func (uc *DeleteAuthUserUsecase) Delete(user *happydns.UserAuth, password string) error {
	// Step 1: Verify the current password.
	if !user.CheckPassword(password) {
		return fmt.Errorf("invalid current password")
	}

	// Step 2: Delete the user's sessions.
	if err := uc.closeUserSessions.CloseAll(user); err != nil {
		return fmt.Errorf("unable to delete user sessions: %w", err)
	}

	// Step 3: Delete the user from the storage.
	if err := uc.store.DeleteAuthUser(user); err != nil {
		return fmt.Errorf("unable to delete user: %w", err)
	}

	return nil
}
