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
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

// NoopMailer is a mock mailer that discards all emails.
type NoopMailer struct{}

func (n *NoopMailer) SendMail(to *mail.Address, subject, content string) error {
	return nil
}

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

func setupTestService() (*authuser.Service, storage.Storage) {
	mem, _ := inmemory.NewInMemoryStorage()
	store, _ := kv.NewKVDatabase(mem)
	cfg := &happydns.Options{
		DisableRegistration: false,
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{}
	service := authuser.NewAuthUserUsecases(cfg, &NoopMailer{}, store, mockCloseSessions)
	return service, store
}

func requireValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var ve happydns.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
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
	mem, _ := inmemory.NewInMemoryStorage()
	store, _ := kv.NewKVDatabase(mem)
	cfg := &happydns.Options{
		DisableRegistration: true,
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{}
	service := authuser.NewAuthUserUsecases(cfg, &NoopMailer{}, store, mockCloseSessions)

	reg := happydns.UserRegistration{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
	}

	err := service.CanRegister(reg)
	if err == nil {
		t.Error("expected registration closed error, got nil")
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
		t.Error("expected defined password")
	}
	if !user.AllowCommercials {
		t.Error("expected user to have AllowCommercials = true")
	}
}

func TestCreateAuthUser_InvalidEmail(t *testing.T) {
	service, _ := setupTestService()

	cases := []string{"", "ab", "bademail", "a@"}
	for _, email := range cases {
		t.Run(email, func(t *testing.T) {
			reg := happydns.UserRegistration{
				Email:    email,
				Password: "StrongPassword123!",
			}
			_, err := service.CreateAuthUser(reg)
			requireValidationError(t, err)
		})
	}
}

func TestCreateAuthUser_WeakPassword(t *testing.T) {
	service, _ := setupTestService()

	cases := []struct {
		name     string
		password string
	}{
		{"too short", "123"},
		{"short with symbols", "Secur3$"},
		{"no uppercase", "secure123"},
		{"short without symbols", "Secure123"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := happydns.UserRegistration{
				Email:    "test@example.com",
				Password: tc.password,
			}
			_, err := service.CreateAuthUser(reg)
			requireValidationError(t, err)
		})
	}
}

func TestCreateAuthUser_PasswordMaxLength(t *testing.T) {
	service, _ := setupTestService()

	// Exactly 72 characters should be accepted (bcrypt limit)
	pw72 := "Abcdefg1!" + strings.Repeat("x", 63) // 9 + 63 = 72
	reg := happydns.UserRegistration{
		Email:    "max72@example.com",
		Password: pw72,
	}
	_, err := service.CreateAuthUser(reg)
	if err != nil {
		t.Fatalf("expected 72-char password to be accepted, got %v", err)
	}

	// 73 characters should be rejected
	pw73 := pw72 + "x"
	reg = happydns.UserRegistration{
		Email:    "max73@example.com",
		Password: pw73,
	}
	_, err = service.CreateAuthUser(reg)
	requireValidationError(t, err)
}

func TestCreateAuthUser_EmailAlreadyUsed(t *testing.T) {
	service, _ := setupTestService()

	reg := happydns.UserRegistration{
		Email:    "used@example.com",
		Password: "StrongPassword123!",
	}
	_, err := service.CreateAuthUser(reg)
	if err != nil {
		t.Fatalf("setup user creation failed: %v", err)
	}

	// Try creating again with the same email.
	// The implementation silently succeeds (returns nil, nil) to prevent user enumeration.
	user, err := service.CreateAuthUser(reg)
	if err != nil {
		t.Errorf("expected no error for duplicate email (anti-enumeration), got: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user for duplicate email, got non-nil")
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

	t.Run("returns the correct user", func(t *testing.T) {
		got, err := service.GetAuthUser(user.Id)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if got.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %s", got.Email)
		}
	})

	t.Run("by email returns the correct user", func(t *testing.T) {
		got, err := service.GetAuthUserByEmail("test@example.com")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !got.Id.Equals(user.Id) {
			t.Errorf("Expected ID '%s', got %s", user.Id, got.Id)
		}
	})

	t.Run("returns error for unknown ID", func(t *testing.T) {
		_, err := service.GetAuthUser([]byte("unknown-id"))
		if err == nil {
			t.Error("Expected error for unknown ID, got nil")
		}
	})

	t.Run("returns error for unknown email", func(t *testing.T) {
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

func TestChangePassword_WeakNewPassword(t *testing.T) {
	service, store := setupTestService()

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("OldPassword123!")
	store.CreateAuthUser(user)

	err := service.ChangePassword(user, "short")
	requireValidationError(t, err)

	// Verify old password still works (change was not applied)
	if !user.CheckPassword("OldPassword123!") {
		t.Error("expected old password to still be valid after failed change")
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

	t.Run("correct current password", func(t *testing.T) {
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

	t.Run("incorrect current password", func(t *testing.T) {
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

	t.Run("correct current but weak new password", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Current:         "OldPassword123!",
			Password:        "weak",
			PasswordConfirm: "weak",
		}
		err := service.CheckPassword(user, form)
		requireValidationError(t, err)
	})
}

func TestCheckNewPassword(t *testing.T) {
	service, _ := setupTestService()

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}

	t.Run("matching passwords", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "NewPa$$w0rd",
		}
		err := service.CheckNewPassword(user, form)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})

	t.Run("non-matching passwords", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "DifferentPassword123!",
		}
		err := service.CheckNewPassword(user, form)
		requireValidationError(t, err)
	})

	t.Run("empty confirmation is accepted", func(t *testing.T) {
		form := happydns.ChangePasswordForm{
			Password:        "NewPa$$w0rd",
			PasswordConfirm: "",
		}
		err := service.CheckNewPassword(user, form)
		if err != nil {
			t.Fatalf("Expected empty confirmation to be accepted, got %v", err)
		}
	})
}

// ========== DeleteAuthUser Tests ==========

func TestDeleteAuthUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	store, _ := kv.NewKVDatabase(mem)
	cfg := &happydns.Options{
		DisableRegistration: false,
	}
	mockCloseSessions := &MockCloseUserSessionsUsecase{
		CloseAllFunc: func(user happydns.UserInfo) error {
			return nil
		},
	}
	service := authuser.NewAuthUserUsecases(cfg, &NoopMailer{}, store, mockCloseSessions)

	user := &happydns.UserAuth{
		Email: "test@example.com",
	}
	user.DefinePassword("TestPassword123!")

	err := store.CreateAuthUser(user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	t.Run("invalid password", func(t *testing.T) {
		err := service.DeleteAuthUser(user, "WrongPassword")
		if err == nil {
			t.Error("expected error for invalid password")
		}
	})

	t.Run("error in closing sessions", func(t *testing.T) {
		mockCloseSessions.CloseAllFunc = func(user happydns.UserInfo) error {
			return fmt.Errorf("error closing sessions")
		}
		err := service.DeleteAuthUser(user, "TestPassword123!")
		if err == nil {
			t.Error("expected error when session close fails")
		}
	})

	t.Run("successful deletion", func(t *testing.T) {
		mockCloseSessions.CloseAllFunc = func(user happydns.UserInfo) error {
			return nil
		}
		err := service.DeleteAuthUser(user, "TestPassword123!")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify user is gone
		_, err = store.GetAuthUser(user.Id)
		if err == nil {
			t.Error("expected error when fetching deleted user")
		}
	})
}

// ========== GenRegistrationHash Tests ==========

func TestGenRegistrationHash_Deterministic(t *testing.T) {
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	key := []byte("test-recovery-key-for-registration-hash-0123456")

	hash1 := authuser.GenRegistrationHash(createdAt, key, false)
	hash2 := authuser.GenRegistrationHash(createdAt, key, false)

	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Error("expected identical hashes for same input and time period")
	}
}

func TestGenRegistrationHash_EmptyKey(t *testing.T) {
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	hash := authuser.GenRegistrationHash(createdAt, nil, false)
	if hash != "" {
		t.Errorf("expected empty hash for nil key, got %q", hash)
	}

	hash = authuser.GenRegistrationHash(createdAt, []byte{}, false)
	if hash != "" {
		t.Errorf("expected empty hash for empty key, got %q", hash)
	}
}

func TestGenRegistrationHash_DifferentPeriods(t *testing.T) {
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	key := []byte("test-recovery-key-for-registration-hash-0123456")

	current := authuser.GenRegistrationHash(createdAt, key, false)
	previous := authuser.GenRegistrationHash(createdAt, key, true)

	if current == "" || previous == "" {
		t.Error("expected non-empty hashes for both periods")
	}
}

func TestGenRegistrationHash_DifferentCreatedAt(t *testing.T) {
	key := []byte("shared-key-for-different-createdat-test-1234567")
	createdAt1 := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	createdAt2 := time.Date(2025, 6, 20, 14, 0, 0, 0, time.UTC)

	hash1 := authuser.GenRegistrationHash(createdAt1, key, false)
	hash2 := authuser.GenRegistrationHash(createdAt2, key, false)

	if hash1 == hash2 {
		t.Error("expected different hashes for different CreatedAt")
	}
}

func TestGenRegistrationHash_DifferentKeys(t *testing.T) {
	createdAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	key1 := []byte("key-one-for-registration-hash-different-keys-test")
	key2 := []byte("key-two-for-registration-hash-different-keys-test")

	hash1 := authuser.GenRegistrationHash(createdAt, key1, false)
	hash2 := authuser.GenRegistrationHash(createdAt, key2, false)

	if hash1 == hash2 {
		t.Error("expected different hashes for different keys")
	}
}

// ========== GenAccountRecoveryHash Tests ==========

func TestGenAccountRecoveryHash_Deterministic(t *testing.T) {
	key := []byte("some-secret-recovery-key-for-testing-1234567890")

	hash1 := authuser.GenAccountRecoveryHash(key, false)
	hash2 := authuser.GenAccountRecoveryHash(key, false)

	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash1 != hash2 {
		t.Error("expected identical hashes for same key and time period")
	}
}

func TestGenAccountRecoveryHash_EmptyKey(t *testing.T) {
	hash := authuser.GenAccountRecoveryHash(nil, false)
	if hash != "" {
		t.Errorf("expected empty hash for nil key, got %q", hash)
	}

	hash = authuser.GenAccountRecoveryHash([]byte{}, false)
	if hash != "" {
		t.Errorf("expected empty hash for empty key, got %q", hash)
	}
}

func TestGenAccountRecoveryHash_DifferentKeys(t *testing.T) {
	key1 := []byte("key-one-for-testing-recovery-hash-generation")
	key2 := []byte("key-two-for-testing-recovery-hash-generation")

	hash1 := authuser.GenAccountRecoveryHash(key1, false)
	hash2 := authuser.GenAccountRecoveryHash(key2, false)

	if hash1 == hash2 {
		t.Error("expected different hashes for different keys")
	}
}

// ========== CanRecoverAccount Tests ==========

func TestCanRecoverAccount_ValidKey(t *testing.T) {
	key := []byte("recovery-key-for-can-recover-test-1234567890ab")
	user := &happydns.UserAuth{
		Email:               "test@example.com",
		PasswordRecoveryKey: key,
	}

	validHash := authuser.GenAccountRecoveryHash(key, false)
	err := authuser.CanRecoverAccount(user, validHash)
	if err != nil {
		t.Fatalf("expected valid key to be accepted, got %v", err)
	}
}

func TestCanRecoverAccount_PreviousPeriodKey(t *testing.T) {
	key := []byte("recovery-key-for-previous-period-test-12345678")
	user := &happydns.UserAuth{
		Email:               "test@example.com",
		PasswordRecoveryKey: key,
	}

	previousHash := authuser.GenAccountRecoveryHash(key, true)
	err := authuser.CanRecoverAccount(user, previousHash)
	if err != nil {
		t.Fatalf("expected previous-period key to be accepted, got %v", err)
	}
}

func TestCanRecoverAccount_InvalidKey(t *testing.T) {
	key := []byte("recovery-key-for-invalid-key-test-1234567890ab")
	user := &happydns.UserAuth{
		Email:               "test@example.com",
		PasswordRecoveryKey: key,
	}

	err := authuser.CanRecoverAccount(user, "totally-invalid-key")
	if err == nil {
		t.Error("expected error for invalid recovery key")
	}
}

func TestCanRecoverAccount_NilRecoveryKey(t *testing.T) {
	user := &happydns.UserAuth{
		Email:               "test@example.com",
		PasswordRecoveryKey: nil,
	}

	err := authuser.CanRecoverAccount(user, "any-key")
	if err == nil {
		t.Error("expected error when user has no recovery key")
	}
}

// ========== Email Validation Flow Tests ==========

func TestEmailValidation_GenerateLink(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "validate@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	link, err := service.GenerateValidationLink(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link == "" {
		t.Fatal("expected non-empty validation link")
	}
	if !strings.Contains(link, "/email-validation") {
		t.Errorf("expected link to contain /email-validation, got %s", link)
	}
	if !strings.Contains(link, "u=") || !strings.Contains(link, "k=") {
		t.Errorf("expected link to contain u= and k= parameters, got %s", link)
	}
}

func TestEmailValidation_ValidateSuccess(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "validate@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if user.EmailVerification != nil {
		t.Fatal("expected EmailVerification to be nil before validation")
	}

	// Ensure recovery key exists (GenerateValidationLink generates it as side effect)
	_, err = service.GenerateValidationLink(user)
	if err != nil {
		t.Fatalf("failed to generate validation link: %v", err)
	}

	key := authuser.GenRegistrationHash(user.CreatedAt, user.PasswordRecoveryKey, false)
	err = service.ValidateEmail(user, happydns.AddressValidationForm{Key: key})
	if err != nil {
		t.Fatalf("expected validation to succeed, got %v", err)
	}

	if user.EmailVerification == nil {
		t.Error("expected EmailVerification to be set after validation")
	}
}

func TestEmailValidation_ValidateWithPreviousPeriodKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "validate-prev@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = service.GenerateValidationLink(user)
	if err != nil {
		t.Fatalf("failed to generate validation link: %v", err)
	}

	key := authuser.GenRegistrationHash(user.CreatedAt, user.PasswordRecoveryKey, true)
	err = service.ValidateEmail(user, happydns.AddressValidationForm{Key: key})
	if err != nil {
		t.Fatalf("expected previous-period key to be accepted, got %v", err)
	}
}

func TestEmailValidation_ValidateInvalidKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "validate-bad@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = service.ValidateEmail(user, happydns.AddressValidationForm{Key: "invalid-key"})
	requireValidationError(t, err)

	if user.EmailVerification != nil {
		t.Error("expected EmailVerification to remain nil after failed validation")
	}
}

// ========== Recovery Flow Tests ==========

func TestRecovery_GenerateLink(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "recover@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	link, err := service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if link == "" {
		t.Fatal("expected non-empty recovery link")
	}
	if !strings.Contains(link, "/forgotten-password") {
		t.Errorf("expected link to contain /forgotten-password, got %s", link)
	}
	if !strings.Contains(link, "u=") || !strings.Contains(link, "k=") {
		t.Errorf("expected link to contain u= and k= parameters, got %s", link)
	}

	if user.PasswordRecoveryKey == nil {
		t.Error("expected PasswordRecoveryKey to be set after generating link")
	}
}

func TestRecovery_GenerateLinkIdempotent(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "recover-idem@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	link1, err := service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("expected no error on first call, got %v", err)
	}

	link2, err := service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("expected no error on second call, got %v", err)
	}

	if link1 != link2 {
		t.Error("expected same link for repeated calls (key already exists)")
	}
}

func TestRecovery_ResetPasswordSuccess(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "reset@example.com",
		Password: "OldPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("failed to generate recovery link: %v", err)
	}

	key := authuser.GenAccountRecoveryHash(user.PasswordRecoveryKey, false)
	newPassword := "NewPa$$w0rd99"

	err = service.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      key,
		Password: newPassword,
	})
	if err != nil {
		t.Fatalf("expected password reset to succeed, got %v", err)
	}

	if !user.CheckPassword(newPassword) {
		t.Error("expected new password to work after reset")
	}
}

func TestRecovery_ResetPasswordInvalidKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "reset-bad@example.com",
		Password: "OldPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("failed to generate recovery link: %v", err)
	}

	err = service.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      "invalid-key",
		Password: "NewPa$$w0rd99",
	})
	if err == nil {
		t.Error("expected error for invalid recovery key")
	}

	if !user.CheckPassword("OldPassword123!") {
		t.Error("expected old password to still work after failed reset")
	}
}

func TestRecovery_ResetPasswordWeakNewPassword(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "reset-weak@example.com",
		Password: "OldPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("failed to generate recovery link: %v", err)
	}

	key := authuser.GenAccountRecoveryHash(user.PasswordRecoveryKey, false)

	err = service.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      key,
		Password: "weak",
	})
	requireValidationError(t, err)
}

func TestRecovery_ResetPasswordInvalidatesKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "reset-invalidate@example.com",
		Password: "OldPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, err = service.GenerateRecoveryLink(user)
	if err != nil {
		t.Fatalf("failed to generate recovery link: %v", err)
	}

	key := authuser.GenAccountRecoveryHash(user.PasswordRecoveryKey, false)

	err = service.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      key,
		Password: "NewPa$$w0rd99",
	})
	if err != nil {
		t.Fatalf("expected first reset to succeed, got %v", err)
	}

	// DefinePassword clears PasswordRecoveryKey, so the same key should no longer work
	if user.PasswordRecoveryKey != nil {
		t.Error("expected PasswordRecoveryKey to be nil after password reset")
	}

	err = authuser.CanRecoverAccount(user, key)
	if err == nil {
		t.Error("expected recovery key to be invalidated after successful reset")
	}
}

// ========== SendRecoveryLink Tests ==========

func TestSendRecoveryLink(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "send-recover@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = service.SendRecoveryLink(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.PasswordRecoveryKey == nil {
		t.Error("expected PasswordRecoveryKey to be set after sending recovery link")
	}
}

// ========== SendValidationLink Tests ==========

func TestSendValidationLink(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "send-validate@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	err = service.SendValidationLink(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
