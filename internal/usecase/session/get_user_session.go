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
	"git.happydns.org/happyDomain/model"
)

type GetUserSessionUsecase struct {
	store SessionStorage
}

func NewGetUserSessionUsecase(store SessionStorage) *GetUserSessionUsecase {
	return &GetUserSessionUsecase{
		store: store,
	}
}

func (uc *GetUserSessionUsecase) Get(user *happydns.User, sessionID string) (*happydns.Session, error) {
	session, err := uc.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(session.IdUser) {
		return nil, happydns.ErrSessionNotFound
	}

	return session, nil
}
