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

// ChangePasswordUsecase handles the logic for changing a user's password.
type ChangePasswordUsecase struct {
	store                    AuthUserStorage
	checkPasswordConstraints *CheckPasswordConstraintsUsecase
}

// NewChangePasswordUsecase creates a new instance of ChangePasswordUsecase.
func NewChangePasswordUsecase(store AuthUserStorage, checkPasswordConstraints *CheckPasswordConstraintsUsecase) *ChangePasswordUsecase {
	return &ChangePasswordUsecase{
		store:                    store,
		checkPasswordConstraints: checkPasswordConstraints,
	}
}

// Change changes the password of the given user after verifying the current password
// and checking new password constraints (length, confirmation, etc.).
func (uc *ChangePasswordUsecase) Change(user *happydns.UserAuth, password string) error {
	// Validate the new password according to application constraints
	if err := uc.checkPasswordConstraints.Check(password); err != nil {
		return happydns.ValidationError{Msg: err.Error()}
	}

	// Apply the new password to the user
	if err := user.DefinePassword(password); err != nil {
		return fmt.Errorf("unable to change user password: %w", err)
	}

	// Persist the updated user information
	if err := uc.store.UpdateAuthUser(user); err != nil {
		return fmt.Errorf("unable to save new password: %w", err)
	}

	return nil
}

func (uc *ChangePasswordUsecase) CheckNewPassword(user *happydns.UserAuth, form happydns.ChangePasswordForm) error {
	if !user.CheckPassword(form.Current) {
		return happydns.ValidationError{Msg: "bad current password"}
	}

	return uc.CheckResetPassword(user, form)
}

func (uc *ChangePasswordUsecase) CheckResetPassword(user *happydns.UserAuth, form happydns.ChangePasswordForm) error {
	// Validate the new password according to application constraints
	if err := uc.checkPasswordConstraints.Check(form.Password); err != nil {
		return happydns.ValidationError{Msg: err.Error()}
	}

	// Confirm the new password matches its confirmation
	if form.Password != form.PasswordConfirm {
		return happydns.ValidationError{Msg: "the new password and its confirmation are different."}
	}

	return nil
}
