// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

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

type User struct {
	Id                  int64
	Email               string
	Password            []byte
	RegistrationTime    *time.Time
	EmailValidated      *time.Time
	PasswordRecoveryKey []byte `json:",omitempty"`
}

type Users []*User

func NewUser(email string, password string) (u *User, err error) {
	t := time.Now()

	u = &User{
		Id:               0,
		Email:            email,
		RegistrationTime: &t,
	}

	err = u.DefinePassword(password)

	return
}

func (u *User) CheckPasswordConstraints(password string) (err error) {
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

func (u *User) DefinePassword(password string) (err error) {
	if err = u.CheckPasswordConstraints(password); err != nil {
		return
	}

	u.Password, err = bcrypt.GenerateFromPassword([]byte(password), 0)
	u.PasswordRecoveryKey = nil

	return
}

func (u *User) CheckAuth(password string) bool {
	return bcrypt.CompareHashAndPassword(u.Password, []byte(password)) == nil
}

const RegistrationHashValidity = 24 * time.Hour

func (u *User) GenRegistrationHash(previous bool) string {
	date := time.Now()
	if previous {
		date = date.Add(RegistrationHashValidity * -1)
	}
	date = date.Truncate(RegistrationHashValidity)

	return base64.StdEncoding.EncodeToString(
		hmac.New(
			sha512.New,
			[]byte(u.RegistrationTime.Format(time.RFC3339Nano)),
		).Sum(date.AppendFormat([]byte{}, time.RFC3339)),
	)
}

func (u *User) ValidateEmail(key string) error {
	if key == u.GenRegistrationHash(false) || key == u.GenRegistrationHash(true) {
		now := time.Now()
		u.EmailValidated = &now
		return nil
	}

	return fmt.Errorf("The validation address link you follow is invalid or has expired (it is valid during %d hours)", RegistrationHashValidity/time.Hour)
}

const AccountRecoveryHashValidity = 2 * time.Hour

func (u *User) GenAccountRecoveryHash(previous bool) string {
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

	return base64.StdEncoding.EncodeToString(
		hmac.New(
			sha512.New,
			u.PasswordRecoveryKey,
		).Sum(date.AppendFormat([]byte{}, time.RFC3339)),
	)
}

func (u *User) CanRecoverAccount(key string) error {
	if key == u.GenAccountRecoveryHash(false) || key == u.GenAccountRecoveryHash(true) {
		return nil
	}

	return fmt.Errorf("The account recovery link you follow is invalid or has expired (it is valid during %d hours)", AccountRecoveryHashValidity/time.Hour)
}
