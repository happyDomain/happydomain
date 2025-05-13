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

type Service struct {
	CreateUserSessionUC *CreateUserSessionUsecase
	DeleteUserSessionUC *DeleteUserSessionUsecase
	GetUserSessionUC    *GetUserSessionUsecase
	ListUserSessionsUC  *ListUserSessionsUsecase
	UpdateUserSessionUC *UpdateUserSessionUsecase
	CloseUserSessionsUC *CloseUserSessionsUsecase
}

func NewSessionUsecases(store SessionStorage) *Service {
	getSessionUC := NewGetUserSessionUsecase(store)
	listSessionsUC := NewListUserSessionsUsecase(store)
	deleteSessionUC := NewDeleteUserSessionUsecase(store, getSessionUC)

	return &Service{
		CreateUserSessionUC: NewCreateUserSessionUsecase(store),
		DeleteUserSessionUC: deleteSessionUC,
		GetUserSessionUC:    getSessionUC,
		ListUserSessionsUC:  listSessionsUC,
		UpdateUserSessionUC: NewUpdateUserSessionUsecase(store, getSessionUC),
		CloseUserSessionsUC: NewCloseUserSessionsUsecase(store, listSessionsUC, deleteSessionUC),
	}
}

func (s *Service) CloseUserSessions(user *happydns.User) error {
	return s.CloseUserSessionsUC.CloseAll(user)
}

func (s *Service) CreateUserSession(user *happydns.User, description string) (*happydns.Session, error) {
	return s.CreateUserSessionUC.Create(user, description)
}

func (s *Service) DeleteUserSession(user *happydns.User, sessionID string) error {
	return s.DeleteUserSessionUC.Delete(user, sessionID)
}

func (s *Service) GetUserSession(user *happydns.User, sessionID string) (*happydns.Session, error) {
	return s.GetUserSessionUC.Get(user, sessionID)
}

func (s *Service) ListUserSessions(user *happydns.User) ([]*happydns.Session, error) {
	return s.ListUserSessionsUC.List(user)
}

func (s *Service) UpdateUserSession(user *happydns.User, sessionID string, updateFunc func(sess *happydns.Session)) error {
	return s.UpdateUserSessionUC.Update(user, sessionID, updateFunc)
}
