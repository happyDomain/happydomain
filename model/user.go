// Copyright or © or Copr. happyDNS (2020)
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
	"time"
)

// User represents an account.
type User struct {
	// Id is the User's identifier.
	Id Identifier `json:"id"`

	// Email is the User's login and mean of contact.
	Email string `json:"email"`

	// CreatedAt is the time when the User logs in for the first time.
	CreatedAt time.Time `json:"created_at,omitempty"`

	// LastSeen is the time when the User used happyDNS for the last time (in a 12h frame).
	LastSeen time.Time `json:"last_seen,omitempty"`

	// Settings holds the settings for an account.
	Settings UserSettings `json:"settings,omitempty"`
}

// Users is a group of User.
type Users []*User

// NewUser fills a new User structure.
func NewUser(email string) (u *User, err error) {
	u = &User{
		Email:     email,
		CreatedAt: time.Now(),
	}

	return
}

// Update updates updatables user fields.
func (u *User) Update(email string) (err error) {
	u.Email = email
	u.LastSeen = time.Now()

	return
}
