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
