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
	"regexp"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// Service groups all use cases related to user authentication and management.
type Service struct {
	store             AuthUserStorage
	mailer            happydns.Mailer
	config            *happydns.Options
	closeUserSessions happydns.SessionCloserUsecase
	emailValidation   *EmailValidationUsecase
	recovery          *RecoverAccountUsecase
}

// NewAuthUserUsecases initializes and returns a new AuthUserService, containing all use cases.
func NewAuthUserUsecases(
	cfg *happydns.Options,
	mailer happydns.Mailer,
	store AuthUserStorage,
	closeUserSessionsUseCase happydns.SessionCloserUsecase,
) *Service {
	emailValidation := NewEmailValidationUsecase(store, mailer, cfg)

	s := &Service{
		store:             store,
		mailer:            mailer,
		config:            cfg,
		closeUserSessions: closeUserSessionsUseCase,
		emailValidation:   emailValidation,
	}

	// Recovery needs a reference to the service for password changes
	s.recovery = NewRecoverAccountUsecase(store, mailer, cfg, s)

	return s
}

// CanRegister checks if user registration is allowed on this instance.
func (s *Service) CanRegister(user happydns.UserRegistration) error {
	if s.config.DisableRegistration {
		return fmt.Errorf("Registration are closed on this instance.")
	}
	return nil
}

// CheckPassword validates the user's current password and new password constraints.
func (s *Service) CheckPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	if !user.CheckPassword(request.Current) {
		return happydns.ValidationError{Msg: "bad current password"}
	}
	return s.checkPasswordConstraints(request.Password, request.PasswordConfirm)
}

// CheckNewPassword validates the new password without checking the current password.
func (s *Service) CheckNewPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	return s.checkPasswordConstraints(request.Password, request.PasswordConfirm)
}

// checkPasswordConstraints validates password strength and confirmation match.
func (s *Service) checkPasswordConstraints(password, confirmation string) error {
	if len(password) < 8 {
		return happydns.ValidationError{Msg: "password must be at least 8 characters long"}
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return happydns.ValidationError{Msg: "Password must contain lower case letters."}
	} else if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return happydns.ValidationError{Msg: "Password must contain upper case letters."}
	} else if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return happydns.ValidationError{Msg: "Password must contain numbers."}
	} else if len(password) < 11 && !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		return happydns.ValidationError{Msg: "Password must be longer or contain symbols."}
	}

	if confirmation != "" && password != confirmation {
		return happydns.ValidationError{Msg: "the new password and its confirmation are different."}
	}

	return nil
}

// ChangePassword changes the password of the given user.
func (s *Service) ChangePassword(user *happydns.UserAuth, newPassword string) error {
	// Validate the new password according to application constraints
	if err := s.checkPasswordConstraints(newPassword, ""); err != nil {
		return err
	}

	// Apply the new password to the user
	if err := user.DefinePassword(newPassword); err != nil {
		return fmt.Errorf("unable to change user password: %w", err)
	}

	// Persist the updated user information
	if err := s.store.UpdateAuthUser(user); err != nil {
		return fmt.Errorf("unable to save new password: %w", err)
	}

	return nil
}

// CreateAuthUser validates the registration request, creates the user, and optionally sends a validation email.
func (s *Service) CreateAuthUser(uu happydns.UserRegistration) (*happydns.UserAuth, error) {
	// Validate email format
	if len(uu.Email) <= 3 || !strings.Contains(string(uu.Email), "@") {
		return nil, happydns.ValidationError{Msg: "the given email is invalid"}
	}

	// Validate password strength
	err := s.checkPasswordConstraints(uu.Password, "")
	if err != nil {
		return nil, err
	}

	// Check if an account already exists with this email
	exists, err := s.store.AuthUserExists(string(uu.Email))
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to check if user exists: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}
	if exists {
		return nil, happydns.ValidationError{Msg: "an account already exists with the given address. Try logging in."}
	}

	// Create the user object
	user, err := happydns.NewUserAuth(string(uu.Email), uu.Password)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to create user object: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}
	user.AllowCommercials = uu.WantReceiveUpdate

	// Persist the new user in the storage layer
	if err := s.store.CreateAuthUser(user); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to create user in storage: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}

	// Optionally send the validation email if mailer is configured
	if s.mailer != nil {
		if err = s.emailValidation.SendLink(user); err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to send validation email: %w", err),
				UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
			}
		}
	}

	return user, nil
}

// DeleteAuthUser deletes an authenticated user from the system, ensuring their sessions are also removed.
func (s *Service) DeleteAuthUser(user *happydns.UserAuth, password string) error {
	// Verify the current password
	if !user.CheckPassword(password) {
		return fmt.Errorf("invalid current password")
	}

	// Delete the user's sessions
	if err := s.closeUserSessions.CloseAll(user); err != nil {
		return fmt.Errorf("unable to delete user sessions: %w", err)
	}

	// Delete the user from the storage
	if err := s.store.DeleteAuthUser(user); err != nil {
		return fmt.Errorf("unable to delete user: %w", err)
	}

	return nil
}

// GetAuthUser retrieves an authenticated user by their unique identifier.
func (s *Service) GetAuthUser(userID happydns.Identifier) (*happydns.UserAuth, error) {
	user, err := s.store.GetAuthUser(userID)
	if err != nil {
		return nil, fmt.Errorf("unable to get user by ID: %w", err)
	}
	return user, nil
}

// GetAuthUserByEmail retrieves an authenticated user by their email address.
func (s *Service) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	user, err := s.store.GetAuthUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("unable to get user by email: %w", err)
	}
	return user, nil
}

// GenerateRecoveryLink generates an account recovery link for the given user.
func (s *Service) GenerateRecoveryLink(user *happydns.UserAuth) (string, error) {
	return s.recovery.GenerateLink(user)
}

// SendRecoveryLink sends an account recovery link to the given user's email.
func (s *Service) SendRecoveryLink(user *happydns.UserAuth) error {
	return s.recovery.SendLink(user)
}

// GenerateValidationLink generates an email validation link for the given user.
func (s *Service) GenerateValidationLink(user *happydns.UserAuth) string {
	return s.emailValidation.GenerateLink(user)
}

// ResetPassword resets the user's password using a recovery form.
func (s *Service) ResetPassword(user *happydns.UserAuth, form happydns.AccountRecoveryForm) error {
	return s.recovery.ResetPassword(user, form)
}

// SendValidationLink sends an email validation link to the given user's email.
func (s *Service) SendValidationLink(user *happydns.UserAuth) error {
	return s.emailValidation.SendLink(user)
}

// ValidateEmail validates the user's email address using a validation form.
func (s *Service) ValidateEmail(user *happydns.UserAuth, form happydns.AddressValidationForm) error {
	return s.emailValidation.Validate(user, form)
}
