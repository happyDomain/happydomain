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
	"crypto/sha256"
	"encoding/base64"
	"time"
)

// Session holds information about a User's currently connected.
type Session struct {
	// Id is the Session's public identifier: SessionIDFromToken of the token
	// the client presents. It names the session in the API and in storage,
	// but is not a credential: the token itself is never stored.
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

	// Machine tells the Session was created through the API, to be used as a
	// long lived token by a script or a third party tool, as opposed to a
	// Session opened by a human through the web interface.
	Machine bool `json:"machine,omitempty" readonly:"true"`
}

// SessionInput is used for creating or updating a session.
type SessionInput struct {
	// Description is a user defined string aims to identify each session.
	Description string `json:"description"`

	// ExpiresOn holds the expirate date of the Session.
	ExpiresOn time.Time `json:"exp" format:"date-time"`
}

// IsInteractive reports whether the Session was opened by a human through the
// web interface, as opposed to a machine session created through the API.
func (s *Session) IsInteractive() bool {
	return !s.Machine
}

type SessionCloserUsecase interface {
	CloseAll(user UserInfo) error

	// CloseInteractive closes the sessions opened through the web interface,
	// leaving the machine sessions of the user untouched.
	CloseInteractive(user UserInfo) error

	ByID(userID Identifier) error
}

// SessionWithToken is returned once, when a machine session is created: it is
// the only time the token is handed out, as it is never stored.
type SessionWithToken struct {
	Session

	// Token is the credential to present as a Bearer token.
	Token string `json:"token" binding:"required" readonly:"true"`
}

// SessionPublicIDLen is the length of a session public identifier, as
// returned by SessionIDFromToken.
const SessionPublicIDLen = 43

// SessionIDFromToken derives the public identifier of a session from the token
// the client presents: the base64url-encoded SHA-256 of the token. Whoever
// reads the identifier (a database dump, an access log, the API) cannot turn
// it back into a token.
func SessionIDFromToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// IsValidSessionPublicID reports whether s looks like a public identifier produced
// by SessionIDFromToken.
func IsValidSessionPublicID(s string) bool {
	if len(s) != SessionPublicIDLen {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(s)
	return err == nil
}

type SessionUsecase interface {
	CloseUserSessions(user *User) error
	CreateUserSession(*User, string) (*SessionWithToken, error)
	DeleteUserSession(*User, string) error
	GetUserSession(*User, string) (*Session, error)
	ListUserSessions(*User) ([]*Session, error)
	UpdateUserSession(*User, string, func(*Session)) error
}

// AdminSessionUsecase exposes administrative session operations that are not
// scoped to a specific User. It is intended for the admin API where requests
// already carry sufficient privilege to act on any session.
type AdminSessionUsecase interface {
	ClearAllSessions() error
	GetSessionByID(string) (*Session, error)
	DeleteSessionByID(string) error
}
