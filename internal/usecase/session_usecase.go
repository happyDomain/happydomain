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

package usecase

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	hdSession "git.happydns.org/happyDomain/internal/session"
	"git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

type sessionUsecase struct {
	store session.SessionStorage
}

func NewSessionUsecase(store session.SessionStorage) happydns.SessionUsecase {
	return &sessionUsecase{
		store: store,
	}
}

func (su *sessionUsecase) ClearUserSessions(user *happydns.User) error {
	sessions, err := su.GetUserSessions(user)
	if err != nil {
		return fmt.Errorf("unable to retrieve user sessions: %w", err)
	}

	var errs error
	for _, session := range sessions {
		err = su.store.DeleteSession(session.Id)
		if err != nil {
			errs = errors.Join(errs, happydns.InternalError{
				Err:         fmt.Errorf("Unable to DeleteSession(sid=%s) in clearUsersSessions(uid=%s): %w", session.Id, user.Id.String(), err),
				UserMessage: fmt.Sprintf("Unable to delete session %q", session.Id),
			})
		}
	}

	return errs
}

func (su *sessionUsecase) CreateUserSession(user *happydns.User, description string) (*happydns.Session, error) {
	sessid := hdSession.NewSessionId()

	newsession := &happydns.Session{
		Id:          sessid,
		IdUser:      user.Id,
		Description: description,
		IssuedAt:    time.Now(),
		ExpiresOn:   time.Now().Add(24 * 365 * time.Hour),
	}

	err := su.store.UpdateSession(newsession)
	if err != nil {
		return nil, fmt.Errorf("unable to create new user session: %w", err)
	}

	return newsession, nil
}

func (su *sessionUsecase) DeleteUserSession(user *happydns.User, sessionid string) error {
	session, err := su.GetUserSession(user, sessionid)
	if err != nil {
		return err
	}

	err = su.store.DeleteSession(session.Id)
	if err != nil {
		return fmt.Errorf("unable to delete session: %w", err)
	}

	return nil
}

func (su *sessionUsecase) GetUserSession(user *happydns.User, sessionid string) (*happydns.Session, error) {
	session, err := su.store.GetSession(sessionid)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(session.IdUser) {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("The session is not affiliated witht this user"),
			UserMessage: "The session is not affiliated witht this user",
			HTTPStatus:  http.StatusNotFound,
		}
	}

	return session, err
}

func (su *sessionUsecase) GetUserSessions(user *happydns.User) ([]*happydns.Session, error) {
	return su.store.ListUserSessions(user.Id)
}

func (su *sessionUsecase) UpdateUserSession(user *happydns.User, sessionid string, upd func(*happydns.Session)) error {
	session, err := su.GetUserSession(user, sessionid)
	if err != nil {
		return err
	}

	upd(session)
	session.ModifiedOn = time.Now()

	if session.Id != sessionid {
		return fmt.Errorf("you cannot change the session identifier")
	}

	err = su.store.UpdateSession(session)
	if err != nil {
		return fmt.Errorf("unable to delete session: %w", err)
	}

	return nil
}
