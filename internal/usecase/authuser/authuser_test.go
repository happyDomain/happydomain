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
	"fmt"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

// MockCloseUserSessionsUsecase is a mock implementation of SessionCloserUsecase.
type MockCloseUserSessionsUsecase struct {
	CloseAllFunc func(user happydns.UserInfo) error
}

func (m *MockCloseUserSessionsUsecase) CloseAll(user happydns.UserInfo) error {
	if m.CloseAllFunc != nil {
		return m.CloseAllFunc(user)
	}
	return nil
}

func (m *MockCloseUserSessionsUsecase) ByID(userID happydns.Identifier) error {
	return m.CloseAll(&happydns.UserAuth{Id: userID})
}

func setupTestService() (*authuser.Service, *inmemory.InMemoryStorage) {
	store, _ := inmemory.NewInMemoryStorage()
	cfg := &happydns.Options{
		DisableRegistration: false,
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{}
	// Pass nil mailer to avoid sending emails in tests
	service := authuser.NewAuthUserUsecases(cfg, nil, store, mockCloseSessions)
	return service, store
}

// ========== CanRegister Tests ==========

func TestCanRegister_Success(t *testing.T) {
	service, _ := setupTestService()

	reg := happydns.UserRegistration{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
	}

	err := service.CanRegister(reg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCanRegister_Closed(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	cfg := &happydns.Options{
		DisableRegistration: true, // Registration closed
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{}
	service := authuser.NewAuthUserUsecases(cfg, nil, store, mockCloseSessions)

	reg := happydns.UserRegistration{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
	}

	err := service.CanRegister(reg)
	if err == nil || err.Error() != "Registration are closed on this instance." {
		t.Errorf("expected registration closed error, got: %v", err)
	}
}

// ========== CreateAuthUser Tests ==========

func TestCreateAuthUser_Success(t *testing.T) {
	service, _ := setupTestService()

	reg := happydns.UserRegistration{
		Email:      "test@example.com",
		Password:   "StrongPassword123!",
		Newsletter: true,
	}

	user, err := service.CreateAuthUser(reg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != reg.Email {
		t.Errorf("expected email %s, got %s", reg.Email, user.Email)
	}
	if user.Password == nil {
		t.Errorf("expected defined password, got %s", user.Password)
	}
	if !user.AllowCommercials {
		t.Error("expected user to have AllowCommercials = true")
	}
}

func TestCreateAuthUser_InvalidEmail(t *testing.T) {
	service, _ := setupTestService()

	reg := happydns.UserRegistration{
		Email:    "bademail",
		Password: "StrongPassword123!",
	}

	_, err := service.CreateAuthUser(reg)
	if err == nil || err.Error() != "the given email is invalid" {
		t.Errorf("expected validation error for email, got: %v", err)
	}
}

func TestCreateAuthUser_WeakPassword(t *testing.T) {
	service, _ := setupTestService()

	reg := happydns.UserRegistration{
		Email:    "test@example.com",
		Password: "123",
	}

	_, err := service.CreateAuthUser(reg)
	if err == nil || err.Error() != "password must be at least 8 characters long" {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "Secur3$"
	_, err = service.CreateAuthUser(reg)
	if err == nil || err.Error() != "password must be at least 8 characters long" {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "secure123"
	_, err = service.CreateAuthUser(reg)
	if err == nil || err.Error() != "Password must contain upper case letters." {
		t.Errorf("expected password constraint error, got: %v", err)
	}

	reg.Password = "Secure123"
	_, err = service.CreateAuthUser(reg)
	if err == nil || err.Error() != "Password must be longer or contain symbols." {
		t.Errorf("expected password constraint error, got: %v", err)
	}
}

func TestCreateAuthUser_EmailAlreadyUsed(t *testing.T) {
	service, _ := setupTestService()

	// Create a user first
	reg := happydns.UserRegistration{
		Email:    "used@example.com",
		Password: "StrongPassword123!",
	}
	_, err := service.CreateAuthUser(reg)
	if err != nil {
		t.Fatalf("setup user creation failed: %v", err)
	}

	// Try creating again with the same email
	_, err = service.CreateAuthUser(reg)
	if err == nil || err.Error() != "an account already exists with the given address. Try logging in." {
		t.Errorf("expected duplicate email error, got: %v", err)
	}
}

// ========== GetAuthUser Tests ==========

func TestGetAuthUser(t *testing.T) {
	service, store := setupTestService()

	now := time.Now()
	user := &happydns.UserAuth{
		Email:             "test@example.com",
		EmailVerification: &now,
		CreatedAt:         now,
		LastLoggedIn:      &now,
		Password:          []byte("fakehash"),
	}

	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("Failed to create auth user: %v", err)
	}
	if user.Id == nil {
		t.Fatalf("Expected non-nil user ID, got %s", user.Id)
	}

	t.Run("GetAuthUser returns the correct user", func(t *testing.T) {
		got, err := service.GetAuthUser(user.Id)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if got.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %s", got.Email)
		}
	})

	t.Run("GetAuthUserByEmail returns the correct user", func(t *testing.T) {
		got, err := service.GetAuthUserByEmail("test@example.com")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !got.Id.Equals(user.Id) {
			t.Errorf("Expected ID '%s', got %s", user.Id, got.Id)
		}
	})

	t.Run("GetAuthUser returns error for unknown ID", func(t *testing.T) {
		_, err := service.GetAuthUser([]byte("unknown-id"))
		if err == nil {
			t.Error("Expected error for unknown ID, got nil")
		}
	})

	t.Run("GetAuthUserByEmail returns error for unknown email", func(t *testing.T) {
		_, err := service.GetAuthUserByEmail("unknown@example.com")
		if err == nil {
			t.Error("Expected error for unknown email, got nil")
		}
	})
}

// ========== ChangePassword Tests ==========

func TestChangePassword(t *testing.T) {
	service, store := setupTestService()

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("OldPassword123!")

	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	newPassword := "NewPa$$w0rd"
	err = service.ChangePassword(user, newPassword)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updatedUser, err := store.GetAuthUser(user.Id)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !updatedUser.CheckPassword(newPassword) {
		t.Error("Expected password to be updated")
	}
}

func TestCheckPassword(t *testing.T) {
	service, store := setupTestService()

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("OldPassword123!")

	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Run("CheckPassword with correct current password", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Current:         "OldPassword123!",
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "NewPa$$w0rd",
		}
		err := service.CheckPassword(user, form)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("CheckPassword with incorrect current password", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Current:         "WrongPassword123!",
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "NewPa$$w0rd",
		}
		err := service.CheckPassword(user, form)
		if err == nil {
			t.Error("Expected error for incorrect current password")
		}
	})
}

func TestCheckNewPassword(t *testing.T) {
	service, _ := setupTestService()

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}

	t.Run("CheckNewPassword with matching passwords", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "NewPa$$w0rd",
		}
		err := service.CheckNewPassword(user, form)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("CheckNewPassword with non-matching passwords", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "DifferentPassword123!",
		}
		err := service.CheckNewPassword(user, form)
		if err == nil {
			t.Error("Expected error for non-matching passwords")
		}
	})
}

// ========== DeleteAuthUser Tests ==========

func TestDeleteAuthUser(t *testing.T) {
	store, _ := inmemory.NewInMemoryStorage()
	cfg := &happydns.Options{
		DisableRegistration: false,
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{
		CloseAllFunc: func(user happydns.UserInfo) error {
			return nil
		},
	}
	service := authuser.NewAuthUserUsecases(cfg, nil, store, mockCloseSessions)

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("TestPassword123!")

	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	t.Run("DeleteAuthUser with invalid password", func(t *testing.T) {
		err := service.DeleteAuthUser(user, "WrongPassword")
		if err == nil || err.Error() != "invalid current password" {
			t.Errorf("Expected error 'invalid current password', got %v", err)
		}
	})

	t.Run("DeleteAuthUser with error in closing sessions", func(t *testing.T) {
		mockCloseSessions.CloseAllFunc = func(user happydns.UserInfo) error {
			return fmt.Errorf("error closing sessions")
		}
		err := service.DeleteAuthUser(user, "TestPassword123!")
		if err == nil || err.Error() != "unable to delete user sessions: error closing sessions" {
			t.Errorf("Expected error 'unable to delete user sessions: error closing sessions', got %v", err)
		}
	})

	t.Run("DeleteAuthUser successful deletion", func(t *testing.T) {
		mockCloseSessions.CloseAllFunc = func(user happydns.UserInfo) error {
			return nil
		}
		err := service.DeleteAuthUser(user, "TestPassword123!")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
