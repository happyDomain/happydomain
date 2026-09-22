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

package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"git.happydns.org/happyDomain/model"
)

// GeneratePassword randomly generates a secure 12 chars long password.
func GeneratePassword() (password string, err error) {
	// This will make a 12 chars long password
	b := make([]byte, 9)

	if _, err = rand.Read(b); err != nil {
		return
	}

	password = base64.StdEncoding.EncodeToString(b)

	// Avoid hard to read characters
	for _, i := range [][2]string{
		{"v", "*"}, {"u", "("},
		{"l", "%"}, {"1", "?"},
		{"o", "@"}, {"O", "!"}, {"0", ">"},
		// This one is to avoid problem with openssl
		{"/", "^"},
	} {
		password = strings.ReplaceAll(password, i[0], i[1])
	}

	return
}

// BcryptCost is the target bcrypt cost used when hashing passwords, for user
// accounts and for the admin interface password alike.
const BcryptCost = 12

// NewUserAuth fills a new UserAuth structure and hashes the given password,
// when not empty.
func NewUserAuth(email string, password string) (u *happydns.UserAuth, err error) {
	u = happydns.NewUserAuth(email)

	if len(password) != 0 {
		err = DefinePassword(u, password)
	}

	return
}

// DefinePassword erases the current UserAuth's password by the new one given.
func DefinePassword(u *happydns.UserAuth, password string) (err error) {
	u.Password, err = bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	u.PasswordRecoveryKey = nil

	return
}

// CheckPassword compares the given password to the hashed one in the UserAuth struct.
func CheckPassword(u *happydns.UserAuth, password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}

	return bcrypt.CompareHashAndPassword(u.Password, []byte(password)) == nil
}

// NeedsRehash reports whether the stored password hash was generated with a
// lower cost than the current target, meaning it should be transparently
// upgraded on the next successful login.
func NeedsRehash(u *happydns.UserAuth) bool {
	cost, err := bcrypt.Cost(u.Password)
	return err != nil || cost < BcryptCost
}
