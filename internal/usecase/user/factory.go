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
	"io"

	authuserUC "git.happydns.org/happyDomain/internal/usecase/authuser"
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	AvatarUC     *Avatar
	CreateUserUC *CreateUser
	DeleteUserUC *DeleteUser
	GetUserUC    *GetUser
	UpdateUserUC *UpdateUser
}

func NewUserUsecases(
	store UserStorage,
	newsletter happydns.NewsletterSubscriptor,
	getAuthUser *authuserUC.GetAuthUserUsecase,
	closeUserSessions happydns.SessionCloserUsecase,
) *Service {
	return &Service{
		AvatarUC:     NewAvatar(),
		CreateUserUC: NewCreateUser(store, newsletter),
		DeleteUserUC: NewDeleteUser(store, getAuthUser, closeUserSessions),
		GetUserUC:    NewGetUser(store),
		UpdateUserUC: NewUpdateUser(store),
	}
}

func (s *Service) ChangeUserSettings(user *happydns.User, settings happydns.UserSettings) error {
	return s.UpdateUserUC.UpdateSettings(user, settings)
}

func (s *Service) CreateUser(uinfo happydns.UserInfo) (*happydns.User, error) {
	return s.CreateUserUC.Create(uinfo)
}

func (s *Service) DeleteUser(userid happydns.Identifier) error {
	return s.DeleteUserUC.Delete(userid)
}

func (s *Service) GenerateUserAvatar(user *happydns.User, size int, writer io.Writer) error {
	return s.AvatarUC.Generate(user, size, writer)
}

func (s *Service) GetUser(userid happydns.Identifier) (*happydns.User, error) {
	return s.GetUserUC.ByID(userid)
}

func (s *Service) GetUserByEmail(email string) (*happydns.User, error) {
	return s.GetUserUC.ByEmail(email)
}

func (s *Service) UpdateUser(userID happydns.Identifier, updateFn func(*happydns.User)) error {
	return s.UpdateUserUC.Update(userID, updateFn)
}
