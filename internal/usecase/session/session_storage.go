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

package session

import (
	"git.happydns.org/happyDomain/model"
)

type SessionStorage interface {
	// ListAllSessions retrieves the list of known Sessions.
	ListAllSessions() (happydns.Iterator[happydns.Session], error)

	// GetSession retrieves the Session with the given identifier.
	GetSession(sessionid string) (*happydns.Session, error)

	// ListAuthUserSessions retrieves all Session for the given AuthUser.
	ListAuthUserSessions(user *happydns.UserAuth) ([]*happydns.Session, error)

	// ListUserSessions retrieves all Session for the given User.
	ListUserSessions(userid happydns.Identifier) ([]*happydns.Session, error)

	// UpdateSession updates the fields of the given Session.
	UpdateSession(session *happydns.Session) error

	// DeleteSession removes the given Session from the database.
	DeleteSession(sessionid string) error

	// ClearSessions deletes all Sessions present in the database.
	ClearSessions() error
}
