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

// Session holds informatin about a User's currently connected.
type Session struct {
	// Id is the Session's identifier.
	Id string `json:"id"`

	// IdUser is the User's identifier of the Session.
	IdUser Identifier `json:"login" swaggertype:"string"`

	// Description is a user defined string aims to identify each session.
	Description string `json:"description"`

	// IssuedAt holds the creation date of the Session.
	IssuedAt time.Time `json:"time"`

	// ExpiresOn holds the expirate date of the Session.
	ExpiresOn time.Time `json:"exp"`

	// ModifiedOn is the last time the session has been updated.
	ModifiedOn time.Time `json:"upd"`

	// Content stores data filled by other modules.
	Content string `json:"content,omitempty"`
}

// ClearSession removes all content from the Session.
func (s *Session) ClearSession() {
	s.Content = ""
}

type SessionUsecase interface {
	ClearUserSessions(user *User) error
	CreateUserSession(*User, string) (*Session, error)
	DeleteUserSession(*User, string) error
	GetUserSession(*User, string) (*Session, error)
	GetUserSessions(*User) ([]*Session, error)
	UpdateUserSession(*User, string, func(*Session)) error
}
