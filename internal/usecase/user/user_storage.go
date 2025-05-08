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

package user

import (
	"git.happydns.org/happyDomain/model"
)

type UserStorage interface {
	// ListAllUsers retrieves the list of known Users.
	ListAllUsers() (happydns.Iterator[happydns.User], error)

	// GetUser retrieves the User with the given identifier.
	GetUser(userid happydns.Identifier) (*happydns.User, error)

	// GetUserByEmail retrieves the User with the given email address.
	GetUserByEmail(email string) (*happydns.User, error)

	// CreateOrUpdateUser updates the fields of the given User.
	CreateOrUpdateUser(user *happydns.User) error

	// DeleteUser removes the given User from the database.
	DeleteUser(userid happydns.Identifier) error

	// ClearUsers deletes all Users present in the database.
	ClearUsers() error
}
