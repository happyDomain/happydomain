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
	"git.happydns.org/happyDomain/model"
)

type AuthUserStorage interface {
	// ListAllAuthUsers retrieves the list of known Users.
	ListAllAuthUsers() (happydns.Iterator[happydns.UserAuth], error)

	// GetAuthUser retrieves the User with the given identifier.
	GetAuthUser(id happydns.Identifier) (*happydns.UserAuth, error)

	// GetAuthUserByEmail retrieves the User with the given email address.
	GetAuthUserByEmail(email string) (*happydns.UserAuth, error)

	// AuthUserExists checks if the given email address is already associated to an User.
	AuthUserExists(email string) (bool, error)

	// CreateAuthUser creates a record in the database for the given User.
	CreateAuthUser(user *happydns.UserAuth) error

	// UpdateAuthUser updates the fields of the given User.
	UpdateAuthUser(user *happydns.UserAuth) error

	// DeleteAuthUser removes the given User from the database.
	DeleteAuthUser(user *happydns.UserAuth) error

	// ClearAuthUsers deletes all AuthUsers present in the database.
	ClearAuthUsers() error
}
