// Copyright or © or Copr. happyDNS (2021)
//
// contact@happydomain.org
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

// CheckAuth compares the given password to the hashed one in the UserAuth struct.
func (u *UserAuth) CheckAuth(password string) bool {
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
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// CanRecoverAccount checks if the given key is a valid recovery hash.
func (u *UserAuth) CanRecoverAccount(key string) error {
	if key == u.GenAccountRecoveryHash(false) || key == u.GenAccountRecoveryHash(true) {
		return nil
	}

	return fmt.Errorf("The account recovery link you follow is invalid or has expired (it is valid during %d hours)", AccountRecoveryHashValidity/time.Hour)
}
