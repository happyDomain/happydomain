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
	"io"
	"net/http"
	"time"

	"git.happydns.org/happyDomain/internal/avatar"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type userUsecase struct {
	newsletter happydns.NewsletterSubscriptor
	store      storage.Storage
}

func NewUserUsecase(store storage.Storage, ns happydns.NewsletterSubscriptor) happydns.UserUsecase {
	return &userUsecase{
		newsletter: ns,
		store:      store,
	}
}

func (uu *userUsecase) ChangeUserSettings(user *happydns.User, settings happydns.UserSettings) error {
	user.Settings = settings

	if err := uu.store.CreateOrUpdateUser(user); err != nil {
		return err
	}

	return nil
}

func (uu *userUsecase) CloseUserSessions(userid happydns.Identifier) error {
	// Retrieve all user's sessions to disconnect them
	sessions, err := uu.store.GetUserSessions(userid)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to GetUserSessions in deleteUser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	var errs error
	for _, session := range sessions {
		err = uu.store.DeleteSession(session.Id)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (uu *userUsecase) CreateUser(uinfo happydns.UserInfo) (*happydns.User, error) {
	if uinfo.GetEmail() == "" {
		return nil, fmt.Errorf("unable to create new user as user email is empty")
	}

	user := &happydns.User{
		Id:        uinfo.GetUserId(),
		Email:     uinfo.GetEmail(),
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		Settings:  *happydns.DefaultUserSettings(),
	}

	err := uu.store.CreateOrUpdateUser(user)
	if err != nil {
		return user, err
	}

	if uinfo.JoinNewsletter() {
		err = uu.newsletter.SubscribeToNewsletter(uinfo)
		if err != nil {
			return user, fmt.Errorf("something goes wrong during newsletter subscription: %w", err)
		}
	}

	return user, nil
}

func (uu *userUsecase) DeleteUser(userid happydns.Identifier) error {
	// Disallow route if user is authenticated through local service
	if _, err := uu.store.GetAuthUser(userid); err == nil {
		return fmt.Errorf("This route is for external account only. Please use the route ./delete instead.")
	}

	if err := uu.store.DeleteUser(userid); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteAuthUser in deleteauthuser: %s", err.Error()),
			UserMessage: "Sorry, we are currently unable to delete your profile. Please try again later.",
		}
	}

	return uu.CloseUserSessions(userid)
}

func (uu *userUsecase) GenerateUserAvatar(user *happydns.User, size int, writer io.Writer) error {
	return avatar.GenerateUserAvatar(user, size, writer)
}

func (uu *userUsecase) GetUser(userid happydns.Identifier) (*happydns.User, error) {
	return uu.store.GetUser(userid)
}

func (uu *userUsecase) GetUserByEmail(email string) (*happydns.User, error) {
	return uu.store.GetUserByEmail(email)
}

func (uu *userUsecase) UpdateUser(id happydns.Identifier, upd func(*happydns.User)) error {
	user, err := uu.GetUser(id)
	if err != nil {
		return err
	}

	upd(user)

	if !user.Id.Equals(id) {
		return happydns.InternalError{
			Err:        fmt.Errorf("you cannot change the user identifier"),
			HTTPStatus: http.StatusBadRequest,
		}
	}

	err = uu.store.CreateOrUpdateUser(user)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateUser in UpdateUser: %w", err),
			UserMessage: "Sorry, we are currently unable to update your user. Please retry later.",
		}
	}

	return nil
}
