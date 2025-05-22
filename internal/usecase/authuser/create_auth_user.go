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
	"strings"

	"git.happydns.org/happyDomain/model"
)

// CreateAuthUserUsecase handles the creation of a new authenticated user account.
type CreateAuthUserUsecase struct {
	store                    AuthUserStorage
	mailer                   happydns.Mailer
	checkPasswordConstraints *CheckPasswordConstraintsUsecase
	emailValidation          happydns.EmailValidationUsecase
}

// NewCreateAuthUserUsecase initializes a new instance of CreateAuthUserUsecase.
func NewCreateAuthUserUsecase(store AuthUserStorage, mailer happydns.Mailer, checkPasswordConstraints *CheckPasswordConstraintsUsecase, emailValidation happydns.EmailValidationUsecase) *CreateAuthUserUsecase {
	return &CreateAuthUserUsecase{
		store:                    store,
		mailer:                   mailer,
		checkPasswordConstraints: checkPasswordConstraints,
		emailValidation:          emailValidation,
	}
}

// Create validates the registration request, creates the user, and optionally sends a validation email.
func (uc *CreateAuthUserUsecase) Create(uu happydns.UserRegistration) (*happydns.UserAuth, error) {
	// Validate email format
	if len(uu.Email) <= 3 || !strings.Contains(uu.Email, "@") {
		return nil, happydns.ValidationError{Msg: "the given email is invalid"}
	}

	// Validate password strength
	err := uc.checkPasswordConstraints.Check(uu.Password)
	if err != nil {
		return nil, happydns.ValidationError{Msg: err.Error()}
	}

	// Check if an account already exists with this email
	exists, err := uc.store.AuthUserExists(uu.Email)
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
	user, err := happydns.NewUserAuth(uu.Email, uu.Password)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to create user object: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}
	user.AllowCommercials = uu.Newsletter

	// Persist the new user in the storage layer
	if err := uc.store.CreateAuthUser(user); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to create user in storage: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}

	// Optionally send the validation email if mailer is configured
	if uc.mailer != nil {
		if err = uc.emailValidation.SendLink(user); err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to send validation email: %w", err),
				UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
			}
		}
	}

	return user, nil
}
