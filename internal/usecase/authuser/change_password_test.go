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

package authuser_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

func TestChangePasswordUsecase_Change(t *testing.T) {
	// Setup in-memory storage
	storage, _ := inmemory.NewInMemoryStorage()
	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("oldpassword")

	// Create a user in the storage
	err := storage.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Setup the usecase
	checkPasswordConstraints := authuser.NewCheckPasswordConstraintsUsecase()
	uc := authuser.NewChangePasswordUsecase(storage, checkPasswordConstraints)

	// Test changing password
	newPassword := "newPa$$w0rd"
	err = uc.Change(user, newPassword)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the password was changed
	updatedUser, err := storage.GetAuthUser(user.Id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updatedUser.CheckPassword(newPassword) {
		t.Error("Expected password to be updated")
	}
}

func TestChangePasswordUsecase_CheckNewPassword(t *testing.T) {
	// Setup in-memory storage
	storage, _ := inmemory.NewInMemoryStorage()
	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("oldpassword")

	// Create a user in the storage
	err := storage.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Setup the usecase
	checkPasswordConstraints := authuser.NewCheckPasswordConstraintsUsecase()
	uc := authuser.NewChangePasswordUsecase(storage, checkPasswordConstraints)

	// Test checking new password with correct current password
	form := happydns.ChangePasswordForm{
		Current:         "oldpassword",
		Password:        "newPa$$w0rd",
		PasswordConfirm: "newPa$$w0rd",
	}
	err = uc.CheckNewPassword(user, form)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test checking new password with incorrect current password
	form.Current = "wrongpassword"
	err = uc.CheckNewPassword(user, form)
	if err == nil {
		t.Error("Expected error for incorrect current password")
	}
}

func TestChangePasswordUsecase_CheckResetPassword(t *testing.T) {
	// Setup the usecase
	checkPasswordConstraints := authuser.NewCheckPasswordConstraintsUsecase()
	uc := authuser.NewChangePasswordUsecase(nil, checkPasswordConstraints)

	// Test checking reset password with matching passwords
	form := happydns.ChangePasswordForm{
		Password:        "newPa$$w0rd",
		PasswordConfirm: "newPa$$w0rd",
	}
	err := uc.CheckResetPassword(&happydns.UserAuth{}, form)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test checking reset password with non-matching passwords
	form.PasswordConfirm = "differentpassword"
	err = uc.CheckResetPassword(&happydns.UserAuth{}, form)
	if err == nil {
		t.Error("Expected error for non-matching passwords")
	}
}
