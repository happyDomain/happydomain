// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package happydns

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserAuth represents an account used for authentication (not used in case of external auth).
type UserAuth struct {
	// Id is the User's identifier.
	Id Identifier

	// Email is the User's login and mean of contact.
	Email string

	// EmailVerification is the time when the User verify its email address.
	EmailVerification *time.Time

	// Password is hashed.
	Password []byte

	// PasswordRecoveryKey is a string generated when User asks to recover its account.
	PasswordRecoveryKey []byte `json:",omitempty"`

	// CreatedAt is the time when the User has register is account.
	CreatedAt time.Time

	// LastLoggedIn is the time when the User has logged in for the last time.
	LastLoggedIn *time.Time

	// AllowCommercials stores the user preference regarding email contacts.
	AllowCommercials bool
}

// UserAuths is a group of UserAuth.
type UserAuths []*UserAuth

// NewUserAuth fills a new UserAuth structure.
func NewUserAuth(email string, password string) (u *UserAuth, err error) {
	u = &UserAuth{
		Email:     email,
		CreatedAt: time.Now(),
	}

	if len(password) != 0 {
		err = u.DefinePassword(password)
	}

	return
}

func (u *UserAuth) GetUserId() Identifier {
	return u.Id
}

func (u *UserAuth) GetEmail() string {
	return u.Email
}

func (u *UserAuth) JoinNewsletter() bool {
	return u.AllowCommercials
}

// CheckPasswordConstraints checks the given password is strong enough.
func (u *UserAuth) CheckPasswordConstraints(password string) (err error) {
	if len(password) < 8 {
		return fmt.Errorf("Password has to be at least 8 characters long.")
	}

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("Password has to contain lower case letters.")
	} else if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("Password has to contain upper case letters.")
	} else if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("Password has to contain numbers.")
	} else if len(password) < 11 && !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		return fmt.Errorf("Password has to be longer or contain symbols.")
	}

	return nil
}

// DefinePassword erases the current UserAuth's password by the new one given.
func (u *UserAuth) DefinePassword(password string) (err error) {
	if err = u.CheckPasswordConstraints(password); err != nil {
		return
	}

	u.Password, err = bcrypt.GenerateFromPassword([]byte(password), 0)
	u.PasswordRecoveryKey = nil

	return
}

// CheckPassword compares the given password to the hashed one in the UserAuth struct.
func (u *UserAuth) CheckPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	return bcrypt.CompareHashAndPassword(u.Password, []byte(password)) == nil
}

// RegistrationHashValidity is the time during which the email validation link is at least valid.
const RegistrationHashValidity = 24 * time.Hour

// GenRegistrationHash generates the validation hash for the current or previous period.
// The hash computation is based on some already filled fields in the structure.
func (u *UserAuth) GenRegistrationHash(previous bool) string {
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

// ValidateEmail tries to validate the email address by comparing the given key to the expected one.
func (u *UserAuth) ValidateEmail(key string) error {
	if key == u.GenRegistrationHash(false) || key == u.GenRegistrationHash(true) {
		now := time.Now()
		u.EmailVerification = &now
		return nil
	}

	return fmt.Errorf("The validation address link you follow is invalid or has expired (it is valid during %d hours)", RegistrationHashValidity/time.Hour)
}

// AccountRecoveryHashValidityis the time during which the recovery link is at least valid.
const AccountRecoveryHashValidity = 2 * time.Hour

// GenAccountRecoveryHash generates the recovery hash for the current or previous period.
// It updates the UserAuth structure in some cases, when it needs to generate a new recovery key,
// so don't forget to save the changes made.
func (u *UserAuth) GenAccountRecoveryHash(previous bool) string {
	if u.PasswordRecoveryKey == nil {
		u.PasswordRecoveryKey = make([]byte, 64)
		if _, err := rand.Read(u.PasswordRecoveryKey); err != nil {
			return ""
		}
	}
	date := time.Now()
	date = date.Truncate(AccountRecoveryHashValidity)
	if previous {
		date = date.Add(AccountRecoveryHashValidity * -1)
	}

	if len(u.PasswordRecoveryKey) == 0 {
		return ""
	}

	h := hmac.New(
		sha512.New,
		u.PasswordRecoveryKey,
	)
	h.Write(date.AppendFormat([]byte{}, time.RFC3339))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// CanRecoverAccount checks if the given key is a valid recovery hash.
func (u *UserAuth) CanRecoverAccount(key string) error {
	if key == u.GenAccountRecoveryHash(false) || key == u.GenAccountRecoveryHash(true) {
		return nil
	}

	return fmt.Errorf("The account recovery link you follow is invalid or has expired (it is valid during %d hours)", AccountRecoveryHashValidity/time.Hour)
}

// GetAccountRecoveryURL returns the absolute URL corresponding to the recovery
// URL of the given account.
func (u *UserAuth) GetAccountRecoveryURL(baseurl string) string {
	return baseurl + fmt.Sprintf("/forgotten-password?u=%s&k=%s", base64.RawURLEncoding.EncodeToString(u.Id), u.GenAccountRecoveryHash(false))
}

// GetAccountRecoveryURL returns the absolute URL corresponding to the recovery
// URL of the given account.
func (u *UserAuth) GetRegistrationURL(baseurl string) string {
	return baseurl + fmt.Sprintf("/email-validation?u=%s&k=%s", base64.RawURLEncoding.EncodeToString(u.Id), u.GenRegistrationHash(false))
}

type AuthUserUsecase interface {
	CanRegister(UserRegistration) error
	CheckPassword(*UserAuth, ChangePasswordForm) error
	CheckNewPassword(*UserAuth, ChangePasswordForm) error
	ChangePassword(*UserAuth, string) error
	CloseAuthUserSessions(*UserAuth) error
	CreateAuthUser(UserRegistration) (*UserAuth, error)
	DeleteAuthUser(*UserAuth, string) error
	GetAuthUser(Identifier) (*UserAuth, error)
	GetAuthUserByEmail(string) (*UserAuth, error)
	GetRecoveryLink(*UserAuth) string
	GetValidationLink(*UserAuth) string
	SendRecoveryLink(*UserAuth) error
	SendValidationLink(*UserAuth) error
	ValidateEmail(*UserAuth, AddressValidationForm) error
}
