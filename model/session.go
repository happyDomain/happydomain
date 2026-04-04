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

// Session holds information about a User's currently connected.
type Session struct {
	// Id is the Session's identifier.
	Id string `json:"id" binding:"required" readonly:"true"`

	// IdUser is the User's identifier of the Session.
	IdUser Identifier `json:"login" swaggertype:"string" binding:"required" readonly:"true"`

	// Description is a user defined string aims to identify each session.
	Description string `json:"description" binding:"required"`

	// IssuedAt holds the creation date of the Session.
	IssuedAt time.Time `json:"time" binding:"required" format:"date-time" readonly:"true"`

	// ExpiresOn holds the expirate date of the Session.
	ExpiresOn time.Time `json:"exp" binding:"required" format:"date-time"`

	// ModifiedOn is the last time the session has been updated.
	ModifiedOn time.Time `json:"upd" binding:"required" format:"date-time"`

	// Content stores data filled by other modules.
	Content string `json:"content,omitempty"`
}

// SessionInput is used for creating or updating a session.
type SessionInput struct {
	// Description is a user defined string aims to identify each session.
	Description string `json:"description"`

	// ExpiresOn holds the expirate date of the Session.
	ExpiresOn time.Time `json:"exp" format:"date-time"`
}

// ClearSession removes all content from the Session.
func (s *Session) ClearSession() {
	s.Content = ""
}

type SessionCloserUsecase interface {
	CloseAll(user UserInfo) error
	ByID(userID Identifier) error
}

type SessionUsecase interface {
	CloseUserSessions(user *User) error
	CreateUserSession(*User, string) (*Session, error)
	DeleteUserSession(*User, string) error
	GetUserSession(*User, string) (*Session, error)
	ListUserSessions(*User) ([]*Session, error)
	UpdateUserSession(*User, string, func(*Session)) error
}
