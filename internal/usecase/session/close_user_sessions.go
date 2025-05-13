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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type CloseUserSessionsUsecase struct {
	store             SessionStorage
	listUserSessions  *ListUserSessionsUsecase
	deleteUserSession *DeleteUserSessionUsecase
}

func NewCloseUserSessionsUsecase(
	store SessionStorage,
	listUserSessions *ListUserSessionsUsecase,
	deleteUserSession *DeleteUserSessionUsecase,
) *CloseUserSessionsUsecase {
	return &CloseUserSessionsUsecase{
		store:             store,
		listUserSessions:  listUserSessions,
		deleteUserSession: deleteUserSession,
	}
}

func (uc *CloseUserSessionsUsecase) CloseAll(user happydns.UserInfo) error {
	sessions, err := uc.listUserSessions.List(user)
	if err != nil {
		return fmt.Errorf("unable to retrieve user sessions: %w", err)
	}

	var errs error
	for _, sess := range sessions {
		if err := uc.store.DeleteSession(sess.Id); err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to delete session %q: %w", sess.Id, err))
		}
	}

	return errs
}
