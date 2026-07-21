// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

	"git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

// The tests in this file guard against regressions of the account recovery and
// email validation vulnerabilities:
//
//   - An empty submitted key must never authenticate, even though
//     GenAccountRecoveryHash / GenRegistrationHash return an empty string for an
//     empty recovery key. Otherwise "" == "" would grant an unauthenticated
//     password reset for any user without an active recovery key (which is the
//     default state, since the key is cleared on every password change).
//   - A user with no recovery key set must never be recoverable/validatable.

// ========== CanRecoverAccount: empty-key bypass ==========

// TestCanRecoverAccount_EmptyKeyNilRecoveryKey is the direct regression test for
// the critical account-takeover bypass: an empty key against a user with no
// recovery key must be rejected.
func TestCanRecoverAccount_EmptyKeyNilRecoveryKey(t *testing.T) {
	user := &happydns.UserAuth{
		Email:               "victim@example.com",
		PasswordRecoveryKey: nil,
	}

	if err := authuser.CanRecoverAccount(user, ""); err == nil {
		t.Fatal("SECURITY: empty key with nil recovery key must be rejected")
	}
}

// TestCanRecoverAccount_EmptyKeyEmptyRecoveryKey covers the same bypass when the
// recovery key is an empty (non-nil) slice.
func TestCanRecoverAccount_EmptyKeyEmptyRecoveryKey(t *testing.T) {
	user := &happydns.UserAuth{
		Email:               "victim@example.com",
		PasswordRecoveryKey: []byte{},
	}

	if err := authuser.CanRecoverAccount(user, ""); err == nil {
		t.Fatal("SECURITY: empty key with empty recovery key must be rejected")
	}
}

// TestCanRecoverAccount_EmptyKeyValidRecoveryKey ensures that even when a real
// recovery key is set, submitting an empty key is rejected.
func TestCanRecoverAccount_EmptyKeyValidRecoveryKey(t *testing.T) {
	user := &happydns.UserAuth{
		Email:               "user@example.com",
		PasswordRecoveryKey: []byte("recovery-key-for-empty-submitted-key-test-1234"),
	}

	if err := authuser.CanRecoverAccount(user, ""); err == nil {
		t.Fatal("SECURITY: empty submitted key must be rejected even with a valid recovery key")
	}
}

// TestResetPassword_EmptyKeyAfterPasswordChange reproduces the full exploit
// scenario end-to-end: a user changes their password (which clears the recovery
// key), then an attacker attempts a reset with an empty key. It must fail and
// leave the existing password intact.
func TestResetPassword_EmptyKeyAfterPasswordChange(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "takeover@example.com",
		Password: "OriginalPass123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// A password change clears PasswordRecoveryKey, leaving the user in the
	// vulnerable state the bypass relied on.
	if err := service.ChangePassword(user, "CurrentPass456!"); err != nil {
		t.Fatalf("failed to change password: %v", err)
	}
	if user.PasswordRecoveryKey != nil {
		t.Fatal("precondition failed: expected recovery key to be nil after password change")
	}

	err = service.ResetPassword(user, happydns.AccountRecoveryForm{
		Key:      "",
		Password: "AttackerPass789!",
	})
	if err == nil {
		t.Fatal("SECURITY: password reset with empty key must be rejected")
	}

	if user.CheckPassword("AttackerPass789!") {
		t.Fatal("SECURITY: attacker password must not have been set")
	}
	if !user.CheckPassword("CurrentPass456!") {
		t.Fatal("expected the legitimate password to still work after the failed reset")
	}
}

// ========== Email validation: empty-key bypass ==========

// TestValidateEmail_EmptyKeyNilRecoveryKey guards the same empty-key bypass on
// the email validation path.
func TestValidateEmail_EmptyKeyNilRecoveryKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "novalidate@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Simulate a user with no recovery key (e.g. after a password change).
	user.PasswordRecoveryKey = nil

	err = service.ValidateEmail(user, happydns.AddressValidationForm{Key: ""})
	if err == nil {
		t.Fatal("SECURITY: empty validation key with nil recovery key must be rejected")
	}
	if user.EmailVerification != nil {
		t.Fatal("SECURITY: email must not be validated after a rejected empty key")
	}
}

// TestValidateEmail_EmptyKeyWithRecoveryKey ensures an empty submitted key is
// rejected even when key material is present.
func TestValidateEmail_EmptyKeyWithRecoveryKey(t *testing.T) {
	service, _ := setupTestService()

	user, err := service.CreateAuthUser(happydns.UserRegistration{
		Email:    "novalidate2@example.com",
		Password: "StrongPassword123!",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	if _, err = service.GenerateValidationLink(user); err != nil {
		t.Fatalf("failed to generate validation link: %v", err)
	}

	err = service.ValidateEmail(user, happydns.AddressValidationForm{Key: ""})
	if err == nil {
		t.Fatal("SECURITY: empty validation key must be rejected even with key material present")
	}
	if user.EmailVerification != nil {
		t.Fatal("SECURITY: email must not be validated after a rejected empty key")
	}
}
