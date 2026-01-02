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
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log"
	"net/mail"
	"reflect"
	"time"

	"git.happydns.org/happyDomain/internal/helpers"
	"git.happydns.org/happyDomain/model"
)

// AccountRecoveryHashValidity is the time during which the recovery link is at least valid.
const AccountRecoveryHashValidity = 2 * time.Hour

// GenAccountRecoveryHash generates the recovery hash for the current or previous period.
// It updates the UserAuth structure in some cases, when it needs to generate a new recovery key,
// so don't forget to save the changes made.
func GenAccountRecoveryHash(recoveryKey []byte, previous bool) string {
	date := time.Now()
	date = date.Truncate(AccountRecoveryHashValidity)
	if previous {
		date = date.Add(AccountRecoveryHashValidity * -1)
	}

	if len(recoveryKey) == 0 {
		return ""
	}

	h := hmac.New(sha512.New, recoveryKey)
	h.Write(date.AppendFormat([]byte{}, time.RFC3339))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// CanRecoverAccount checks if the given key is a valid recovery hash.
func CanRecoverAccount(u *happydns.UserAuth, key string) error {
	if key == GenAccountRecoveryHash(u.PasswordRecoveryKey, false) || key == GenAccountRecoveryHash(u.PasswordRecoveryKey, true) {
		return nil
	}

	return fmt.Errorf("The account recovery link you follow is invalid or has expired (it is valid during %d hours)", AccountRecoveryHashValidity/time.Hour)
}

// RecoverAccountUsecase handles account recovery operations.
type RecoverAccountUsecase struct {
	store   AuthUserStorage
	mailer  happydns.Mailer
	config  *happydns.Options
	service *Service
}

// NewRecoverAccountUsecase creates a new RecoverAccountUsecase instance.
func NewRecoverAccountUsecase(store AuthUserStorage, mailer happydns.Mailer, config *happydns.Options, service *Service) *RecoverAccountUsecase {
	return &RecoverAccountUsecase{
		store:   store,
		mailer:  mailer,
		config:  config,
		service: service,
	}
}

// GenerateLink returns the absolute URL corresponding to the recovery
// URL of the given account.
func (uc *RecoverAccountUsecase) GenerateLink(user *happydns.UserAuth) (string, error) {
	if user.PasswordRecoveryKey == nil {
		user.PasswordRecoveryKey = make([]byte, 64)
		if _, err := rand.Read(user.PasswordRecoveryKey); err != nil {
			return "", err
		}

		if err := uc.store.UpdateAuthUser(user); err != nil {
			return "", err
		}
	}

	return uc.config.GetBaseURL() + fmt.Sprintf("/forgotten-password?u=%s&k=%s", base64.RawURLEncoding.EncodeToString(user.Id), GenAccountRecoveryHash(user.PasswordRecoveryKey, false)), nil
}

// SendLink sends an account recovery link to the user's email.
func (uc *RecoverAccountUsecase) SendLink(user *happydns.UserAuth) error {
	link, err := uc.GenerateLink(user)
	if err != nil {
		return fmt.Errorf("unable to generate recovery link: %w", err)
	}

	toName := helpers.GenUsername(user.Email)

	if uc.mailer == nil || reflect.ValueOf(uc.mailer).IsNil() {
		log.Printf("No mailer configured. Recovery link for %s: %s", user.Email, link)
		return nil
	}

	return uc.mailer.SendMail(
		&mail.Address{Name: toName, Address: user.Email},
		"Recover your happyDomain account",
		fmt.Sprintf(`Hi %s,

You've just ask on our platform to recover your account.

In order to define a new password, please follow this link now:

[Recover my account](%s)`, toName, link),
	)
}

// ResetPassword resets the user's password using a recovery form.
func (uc *RecoverAccountUsecase) ResetPassword(user *happydns.UserAuth, form happydns.AccountRecoveryForm) error {
	if err := CanRecoverAccount(user, form.Key); err != nil {
		return err
	}

	if err := uc.service.ChangePassword(user, form.Password); err != nil {
		return err
	}

	return nil
}
