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
	"time"
)

type User struct {
	Id               int64
	Email            string
	Password         []byte
	Salt             []byte
	RegistrationTime *time.Time
}

type Users []*User

func GenPassword(password string, salt []byte) []byte {
	return hmac.New(sha512.New512_224, []byte(password)).Sum(salt)
}

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

// DefinePassword computes the expected hash for the given password and also
// renew the User's Salt.
func (u *User) DefinePassword(password string) error {
	// Renew salt
	u.Salt = make([]byte, 64)
	if _, err := rand.Read(u.Salt); err != nil {
		return err
	}

	// Compute password hash
	u.Password = GenPassword(password, u.Salt)

	return nil
}

func (u *User) CheckAuth(password string) bool {
	pass := GenPassword(password, u.Salt)
	if len(pass) != len(u.Password) {
		return false
	} else {
		for k := range pass {
			if pass[k] != u.Password[k] {
				return false
			}
		}
		return true
	}
}
