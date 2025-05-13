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

// DefinePassword erases the current UserAuth's password by the new one given.
func (u *UserAuth) DefinePassword(password string) (err error) {
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

type AuthUserUsecase interface {
	CanRegister(UserRegistration) error
	CheckPassword(*UserAuth, ChangePasswordForm) error
	ChangePassword(*UserAuth, string) error
	CreateAuthUser(UserRegistration) (*UserAuth, error)
	DeleteAuthUser(*UserAuth, string) error
	GenerateRecoveryLink(*UserAuth) (string, error)
	GenerateValidationLink(*UserAuth) string
	GetAuthUser(Identifier) (*UserAuth, error)
	GetAuthUserByEmail(string) (*UserAuth, error)
	ResetPassword(*UserAuth, AccountRecoveryForm) error
	SendRecoveryLink(*UserAuth) error
	SendValidationLink(*UserAuth) error
	ValidateEmail(*UserAuth, AddressValidationForm) error
}
