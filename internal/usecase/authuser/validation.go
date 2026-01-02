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
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"net/mail"
	"reflect"
	"time"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// RegistrationHashValidity is the time during which the email validation link is at least valid.
const RegistrationHashValidity = 24 * time.Hour

// GenRegistrationHash generates the validation hash for the current or previous period.
// The hash computation is based on some already filled fields in the structure.
func GenRegistrationHash(u *happydns.UserAuth, previous bool) string {
	date := time.Now()
	if previous {
		date = date.Add(RegistrationHashValidity * -1)
	}
	date = date.Truncate(RegistrationHashValidity)

	h := hmac.New(
		sha512.New,
		[]byte(u.CreatedAt.Format(time.RFC3339Nano)),
	)
	h.Write(date.AppendFormat([]byte{}, time.RFC3339))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// EmailValidationUsecase handles email validation operations.
type EmailValidationUsecase struct {
	store  AuthUserStorage
	mailer happydns.Mailer
	config *happydns.Options
}

// NewEmailValidationUsecase creates a new EmailValidationUsecase instance.
func NewEmailValidationUsecase(store AuthUserStorage, mailer happydns.Mailer, config *happydns.Options) *EmailValidationUsecase {
	return &EmailValidationUsecase{
		store:  store,
		mailer: mailer,
		config: config,
	}
}

// GenerateLink returns the absolute URL corresponding to the recovery
// URL of the given account.
func (uc *EmailValidationUsecase) GenerateLink(user *happydns.UserAuth) string {
	return uc.config.GetBaseURL() + fmt.Sprintf("/email-validation?u=%s&k=%s", base64.RawURLEncoding.EncodeToString(user.Id), GenRegistrationHash(user, false))
}

// SendLink sends an email validation link to the user's email.
func (uc *EmailValidationUsecase) SendLink(user *happydns.UserAuth) error {
	if uc.mailer == nil || reflect.ValueOf(uc.mailer).IsNil() {
		return fmt.Errorf("no mailer configured")
	}

	toName := helpers.GenUsername(user.Email)
	return uc.mailer.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Your new account on happyDomain",
		fmt.Sprintf(`
Welcome to happyDomain!
-----------------------

Hi %s,

We are pleased that you created an account on our great domain name
management platform!

In order to validate your account, please follow this link now:

[Validate my account](%s)
`, toName, uc.GenerateLink(user)),
	)
}

// Validate tries to validate the email address by comparing the given key to the expected one.
func (uc *EmailValidationUsecase) Validate(user *happydns.UserAuth, form happydns.AddressValidationForm) error {
	if form.Key != GenRegistrationHash(user, false) && form.Key != GenRegistrationHash(user, true) {
		return happydns.ValidationError{Msg: fmt.Sprintf("bad email validation key: the validation address link you follow is invalid or has expired (it is valid during %d hours)", RegistrationHashValidity/time.Hour)}
	}

	now := time.Now()
	user.EmailVerification = &now

	if err := uc.store.UpdateAuthUser(user); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to update auth user: %w", err),
			UserMessage: "Sorry, we are currently unable to update your profile. Please try again later.",
		}
	}

	return nil
}
