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
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/mailer"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/utils"
	"git.happydns.org/happyDomain/model"
)

type authUserUsecase struct {
	config *config.Options
	mailer *mailer.Mailer
	store  storage.AuthUserAndSessionStorage
}

func NewAuthUserUsecase(cfg *config.Options, m *mailer.Mailer, store storage.AuthUserAndSessionStorage) happydns.AuthUserUsecase {
	return &authUserUsecase{
		config: cfg,
		mailer: m,
		store:  store,
	}
}

func (auu *authUserUsecase) CanRegister(user happydns.UserRegistration) error {
	if auu.config.DisableRegistration {
		return fmt.Errorf("Registration are closed on this instance.")
	}

	return nil
}

func (auu *authUserUsecase) CheckNewPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	if !user.CheckPassword(request.Current) {
		return fmt.Errorf("The given current password is invalid.")
	}

	return auu.CheckPassword(user, request)
}

func (auu *authUserUsecase) CheckPassword(user *happydns.UserAuth, request happydns.ChangePasswordForm) error {
	if err := user.CheckPasswordConstraints(request.Password); err != nil {
		return err
	}

	if request.Password != request.PasswordConfirm {
		return fmt.Errorf("The new password and its confirmation are different.")
	}

	return nil
}

func (auu *authUserUsecase) ChangePassword(user *happydns.UserAuth, newPassword string) error {
	if err := user.DefinePassword(newPassword); err != nil {
		return fmt.Errorf("unable to change user password: %w", err)
	}

	return auu.store.UpdateAuthUser(user)
}

func (auu *authUserUsecase) CloseAuthUserSessions(user *happydns.UserAuth) error {
	// Retrieve all user's sessions to disconnect them
	sessions, err := auu.store.ListAuthUserSessions(user)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to GetUserSessions in deleteAuthUser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	var errs error
	for _, session := range sessions {
		err = auu.store.DeleteSession(session.Id)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (auu *authUserUsecase) CreateAuthUser(uu happydns.UserRegistration) (*happydns.UserAuth, error) {
	if len(uu.Email) <= 3 || strings.Index(uu.Email, "@") == -1 {
		return nil, fmt.Errorf("The given email is invalid.")
	}

	if len(uu.Password) <= 7 {
		return nil, fmt.Errorf("The given password is invalid.")
	}

	exists, err := auu.store.AuthUserExists(uu.Email)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to AuthUserExists in CreateAuthUser: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}
	if exists {
		return nil, fmt.Errorf("An account already exists with the given address. Try login now.")
	}

	user, err := happydns.NewUserAuth(uu.Email, uu.Password)
	if err != nil {
		return nil, err
	}

	user.AllowCommercials = uu.Newsletter

	if err := auu.store.CreateAuthUser(user); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateUser in CreateAuthUser: %w", err),
			UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
		}
	}

	if auu.mailer != nil {
		if err = auu.SendValidationLink(user); err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to SendValidationLink in registerUser: %w", err),
				UserMessage: "Sorry, we are currently unable to create your account. Please try again later.",
			}
		}
	}

	return user, auu.store.CreateAuthUser(user)
}

func (auu *authUserUsecase) DeleteAuthUser(user *happydns.UserAuth, password string) error {
	if !user.CheckPassword(password) {
		return fmt.Errorf("The given current password is invalid.")
	}

	if err := auu.store.DeleteAuthUser(user); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteAuthUser in deleteauthuser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	return auu.CloseAuthUserSessions(user)
}

func (auu *authUserUsecase) GetAuthUser(uid happydns.Identifier) (*happydns.UserAuth, error) {
	return auu.store.GetAuthUser(uid)
}

func (auu *authUserUsecase) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	return auu.store.GetAuthUserByEmail(email)
}

func (auu *authUserUsecase) ListAuthUserSessions(user *happydns.UserAuth) ([]*happydns.Session, error) {
	return auu.store.ListAuthUserSessions(user)
}

func (auu *authUserUsecase) GenerateRecoveryLink(user *happydns.UserAuth) string {
	return user.GetAccountRecoveryURL(auu.config.GetBaseURL())
}

func (auu *authUserUsecase) SendRecoveryLink(user *happydns.UserAuth) error {
	toName := utils.GenUsername(user.Email)

	link := auu.GenerateRecoveryLink(user)

	err := auu.store.UpdateAuthUser(user)
	if err != nil {

	}

	return auu.mailer.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Recover your happyDomain account",
		`Hi `+toName+`,

You've just ask on our platform to recover your account.

In order to define a new password, please follow this link now:

[Recover my account](`+link+`)`,
	)
}

func (auu *authUserUsecase) GenerateValidationLink(user *happydns.UserAuth) string {
	return user.GetRegistrationURL(auu.config.GetBaseURL())
}

func (auu *authUserUsecase) SendValidationLink(user *happydns.UserAuth) error {
	if auu.mailer == nil {
		return fmt.Errorf("No mailer configured")
	}

	toName := utils.GenUsername(user.Email)
	return auu.mailer.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Your new account on happyDomain",
		`Welcome to happyDomain!
--------------------

Hi `+toName+`,

We are pleased that you created an account on our great domain name
management platform!

In order to validate your account, please follow this link now:

[Validate my account](`+auu.GenerateValidationLink(user)+`)`,
	)
}

func (auu *authUserUsecase) ValidateEmail(user *happydns.UserAuth, form happydns.AddressValidationForm) error {
	if err := user.ValidateEmail(form.Key); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("bad email validation key: %w", err),
			HTTPStatus:  http.StatusBadRequest,
			UserMessage: fmt.Sprintf("bad email validation key: %s", err.Error()),
		}
	}

	if err := auu.store.UpdateAuthUser(user); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateUser in ValidateUserAddress: %w", err),
			UserMessage: "Sorry, we are currently unable to update your profile. Please try again later.",
		}
	}

	return nil
}
