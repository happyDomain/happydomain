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

package user

import (
	"fmt"

	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

type DeleteUser struct {
	store             UserStorage
	getAuthUser       *authuserUC.GetAuthUserUsecase
	closeUserSessions happydns.SessionCloserUsecase
}

func NewDeleteUser(store UserStorage, getAuthUser *authuserUC.GetAuthUserUsecase, closeSessions happydns.SessionCloserUsecase) *DeleteUser {
	return &DeleteUser{
		store:             store,
		getAuthUser:       getAuthUser,
		closeUserSessions: closeSessions,
	}
}

func (uc *DeleteUser) Delete(userid happydns.Identifier) error {
	// Disallow route if user is authenticated through local service
	if _, err := uc.getAuthUser.ByID(userid); err == nil {
		return fmt.Errorf("This route is for external account only. Please use the route ./delete instead.")
	}

	if err := uc.store.DeleteUser(userid); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteAuthUser in deleteauthuser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	return uc.closeUserSessions.ByID(userid)
}
