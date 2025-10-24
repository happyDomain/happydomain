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
	"strings"
	"time"

	"github.com/gorilla/securecookie"

	"git.happydns.org/happyDomain/model"
)

const SESSION_MAX_DURATION = 24 * 365 * time.Hour

// Service handles all session-related operations.
// This consolidates what were previously separate usecase structs into a single service.
type Service struct {
	store SessionStorage
}

// NewService creates a new session service.
// This replaces the old NewSessionUsecases factory function.
func NewService(store SessionStorage) *Service {
	return &Service{store: store}
}

// NewSessionUsecases is a backward-compatible alias for NewService.
// Deprecated: Use NewService instead.
func NewSessionUsecases(store SessionStorage) *Service {
	return NewService(store)
}

// CreateUserSession creates a new session for the given user.
// Replaces: CreateUserSessionUsecase.Create
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

// GetUserSession retrieves a session for the given user.
// Replaces: GetUserSessionUsecase.Get
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

// ListUserSessions retrieves all sessions for the given user.
// Replaces: ListUserSessionsUsecase.List
func (s *Service) ListUserSessions(user *happydns.User) ([]*happydns.Session, error) {
	return s.store.ListUserSessions(user.GetUserId())
}

// listUserSessionsInternal is a helper that accepts UserInfo interface.
func (s *Service) listUserSessionsInternal(user happydns.UserInfo) ([]*happydns.Session, error) {
	return s.store.ListUserSessions(user.GetUserId())
}

// UpdateUserSession updates a user's session using the provided update function.
// Replaces: UpdateUserSessionUsecase.Update
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

// DeleteUserSession deletes a specific session for the given user.
// Replaces: DeleteUserSessionUsecase.Delete
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

// CloseUserSessions closes (deletes) all sessions for the given user.
// Replaces: CloseUserSessionsUsecase.CloseAll
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

// closeUserSessionsInternal is a helper that accepts UserInfo interface.
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

// CloseAll implements SessionCloserUsecase interface.
// Closes all sessions for the given user.
func (s *Service) CloseAll(user happydns.UserInfo) error {
	return s.closeUserSessionsInternal(user)
}

// ByID implements SessionCloserUsecase interface.
// Closes all sessions for a user identified by ID.
func (s *Service) ByID(userID happydns.Identifier) error {
	return s.CloseUserSessions(&happydns.User{Id: userID})
}

// NewSessionID generates a new random session identifier.
func NewSessionID() string {
	return strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(64)), "=")
}
