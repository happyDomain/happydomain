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
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/securecookie"

	"git.happydns.org/happyDomain/model"
)

// SESSION_MAX_DURATION is the lifetime assigned to every newly created session.
const SESSION_MAX_DURATION = 15 * 24 * time.Hour

// SESSION_RENEWAL_THRESHOLD is the remaining lifetime below which a session
// is automatically renewed to SESSION_MAX_DURATION on the next request.
const SESSION_RENEWAL_THRESHOLD = 7 * 24 * time.Hour

// Service handles all session-related operations for happyDomain users. It
// relies on a [SessionStorage] backend for persistence and enforces ownership
// checks so that one user can never read or modify another user's sessions.
type Service struct {
	store SessionStorage
}

// NewService creates a new session Service backed by the given store.
func NewService(store SessionStorage) *Service {
	return &Service{store: store}
}

// CreateUserSession creates and persists a new session for user. The session
// is assigned a random identifier, the current time as its issue date, and an
// expiry of [SESSION_MAX_DURATION] from now. description is a human-readable
// label that the user can use to identify the session (e.g. "browser login").
func (s *Service) CreateUserSession(user *happydns.User, description string) (*happydns.Session, error) {
	sessid := NewSessionID()

	newsession := &happydns.Session{
		Id:          sessid,
		IdUser:      user.Id,
		Description: description,
		IssuedAt:    time.Now(),
		ExpiresOn:   time.Now().Add(SESSION_MAX_DURATION),
	}

	if err := s.store.UpdateSession(newsession); err != nil {
		return nil, fmt.Errorf("unable to create new user session: %w", err)
	}

	return newsession, nil
}

// GetUserSession retrieves the session identified by sessionID and verifies
// that it belongs to user. Returns [happydns.ErrSessionNotFound] if the
// session does not exist or belongs to a different user.
func (s *Service) GetUserSession(user *happydns.User, sessionID string) (*happydns.Session, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(session.IdUser) {
		return nil, happydns.ErrSessionNotFound
	}

	return session, nil
}

// ListUserSessions returns all active sessions belonging to user.
func (s *Service) ListUserSessions(user *happydns.User) ([]*happydns.Session, error) {
	return s.store.ListUserSessions(user.GetUserId())
}

// listUserSessionsInternal is like [Service.ListUserSessions] but accepts the
// broader [happydns.UserInfo] interface.
func (s *Service) listUserSessionsInternal(user happydns.UserInfo) ([]*happydns.Session, error) {
	return s.store.ListUserSessions(user.GetUserId())
}

// UpdateUserSession applies updateFunc to the session identified by sessionID
// and persists the result. The session must belong to user; otherwise an error
// is returned. The function sets ModifiedOn automatically. Attempting to change
// the session ID inside updateFunc is rejected with an error.
func (s *Service) UpdateUserSession(
	user *happydns.User,
	sessionID string,
	updateFunc func(sess *happydns.Session),
) error {
	session, err := s.GetUserSession(user, sessionID)
	if err != nil {
		return err
	}

	updateFunc(session)
	session.ModifiedOn = time.Now()

	if session.Id != sessionID {
		return fmt.Errorf("you cannot change the session identifier")
	}

	if err := s.store.UpdateSession(session); err != nil {
		return fmt.Errorf("unable to update session: %w", err)
	}

	return nil
}

// DeleteUserSession removes the session identified by sessionID. The session
// must belong to user; an attempt to delete another user's session returns an
// error and leaves the session untouched.
func (s *Service) DeleteUserSession(user *happydns.User, sessionID string) error {
	sess, err := s.GetUserSession(user, sessionID)
	if err != nil {
		return err
	}

	if err := s.store.DeleteSession(sess.Id); err != nil {
		return fmt.Errorf("unable to delete session: %w", err)
	}

	return nil
}

// CloseUserSessions deletes all sessions belonging to user. Errors from
// individual deletions are collected and returned as a combined error so that
// a single failure does not prevent the remaining sessions from being removed.
func (s *Service) CloseUserSessions(user *happydns.User) error {
	sessions, err := s.ListUserSessions(user)
	if err != nil {
		return fmt.Errorf("unable to retrieve user sessions: %w", err)
	}

	var errs error
	for _, sess := range sessions {
		if err := s.store.DeleteSession(sess.Id); err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to delete session %q: %w", sess.Id, err))
		}
	}

	return errs
}

// closeUserSessionsInternal is like [Service.CloseUserSessions] but accepts
// the broader [happydns.UserInfo] interface.
func (s *Service) closeUserSessionsInternal(user happydns.UserInfo) error {
	sessions, err := s.listUserSessionsInternal(user)
	if err != nil {
		return fmt.Errorf("unable to retrieve user sessions: %w", err)
	}

	var errs error
	for _, sess := range sessions {
		if err := s.store.DeleteSession(sess.Id); err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to delete session %q: %w", sess.Id, err))
		}
	}

	return errs
}

// CloseAll deletes all sessions for user. It satisfies the
// SessionCloserUsecase interface and accepts the broader [happydns.UserInfo]
// type so callers are not required to hold a full [happydns.User] value.
func (s *Service) CloseAll(user happydns.UserInfo) error {
	return s.closeUserSessionsInternal(user)
}

// ByID deletes all sessions for the user identified by userID. It satisfies
// the SessionCloserUsecase interface, allowing callers that only have a user
// identifier to invalidate all of that user's sessions without constructing a
// full [happydns.User] value.
func (s *Service) ByID(userID happydns.Identifier) error {
	return s.CloseUserSessions(&happydns.User{Id: userID})
}

// sessionIDKeyLen is the number of random bytes used to generate a session ID.
const sessionIDKeyLen = 64

// SessionIDLen is the length of a session ID string (base32, no padding).
const SessionIDLen = (sessionIDKeyLen*8 + 4) / 5

// NewSessionID generates a random session identifier encoded
// as a base32 string without padding characters.
func NewSessionID() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(securecookie.GenerateRandomKey(sessionIDKeyLen))
}
