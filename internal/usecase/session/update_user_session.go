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
	"fmt"
	"time"

	"git.happydns.org/happyDomain/model"
)

type UpdateUserSessionUsecase struct {
	store          SessionStorage
	getUserSession *GetUserSessionUsecase
}

func NewUpdateUserSessionUsecase(store SessionStorage, getUserSession *GetUserSessionUsecase) *UpdateUserSessionUsecase {
	return &UpdateUserSessionUsecase{
		store:          store,
		getUserSession: getUserSession,
	}
}

func (uc *UpdateUserSessionUsecase) Update(
	user *happydns.User,
	sessionID string,
	updateFunc func(sess *happydns.Session),
) error {
	session, err := uc.getUserSession.Get(user, sessionID)
	if err != nil {
		return err
	}

	updateFunc(session)
	session.ModifiedOn = time.Now()

	if session.Id != sessionID {
		return fmt.Errorf("you cannot change the session identifier")
	}

	if err := uc.store.UpdateSession(session); err != nil {
		return fmt.Errorf("unable to update session: %w", err)
	}

	return nil
}
