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

	"git.happydns.org/happyDomain/model"
)

type DeleteUserSessionUsecase struct {
	store          SessionStorage
	getUserSession *GetUserSessionUsecase
}

func NewDeleteUserSessionUsecase(store SessionStorage, getUserSession *GetUserSessionUsecase) *DeleteUserSessionUsecase {
	return &DeleteUserSessionUsecase{
		store:          store,
		getUserSession: getUserSession,
	}
}

func (uc *DeleteUserSessionUsecase) Delete(user *happydns.User, sessionID string) error {
	sess, err := uc.getUserSession.Get(user, sessionID)
	if err != nil {
		return err
	}

	if err := uc.store.DeleteSession(sess.Id); err != nil {
		return fmt.Errorf("unable to delete session: %w", err)
	}

	return nil
}
