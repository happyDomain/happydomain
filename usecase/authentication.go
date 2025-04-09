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
	"fmt"
	"time"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type loginUsecase struct {
	config      *config.Options
	store       storage.Storage
	userService happydns.UserUsecase
}

func NewAuthenticationUsecase(cfg *config.Options, store storage.Storage, userService happydns.UserUsecase) happydns.AuthenticationUsecase {
	return &loginUsecase{
		config:      cfg,
		store:       store,
		userService: userService,
	}
}

func (lu *loginUsecase) CompleteAuthentication(uinfo happydns.UserInfo) (*happydns.User, error) {
	// Check if the user already exists
	user, err := lu.store.GetUser(uinfo.GetUserId())
	if err != nil {
		// Create the user
		user, err = lu.userService.CreateUser(uinfo)
		if err != nil {
			return nil, fmt.Errorf("unable to create user account: %w", err)
		}
	} else if (uinfo.GetEmail() != "" && user.Email != uinfo.GetEmail()) || time.Since(user.LastSeen) > time.Hour*12 {
		if uinfo.GetEmail() != "" {
			user.Email = uinfo.GetEmail()
		}
		user.LastSeen = time.Now()

		err = lu.store.CreateOrUpdateUser(user)
		if err != nil {
			return nil, fmt.Errorf("has a correct JWT, user has been found, but an error occured when trying to update the user's information: %w", err)
		}
	}

	return user, nil
}

func (lu *loginUsecase) AuthenticateUserWithPassword(request happydns.LoginRequest) (*happydns.User, error) {
	// Retrieve the given user
	user, err := lu.store.GetAuthUserByEmail(request.Email)
	if err != nil {
		return nil, fmt.Errorf("user's email (%s) not found: %s", request.Email, err.Error())
	}

	if !user.CheckPassword(request.Password) {
		return nil, fmt.Errorf("tries to login as %q, but sent an invalid password", request.Email)
	}

	// Ensure the account is enabled
	if !lu.config.NoMail && user.EmailVerification == nil {
		return nil, fmt.Errorf("tries to login as %q, but has not verified email", request.Email)
	}

	return lu.CompleteAuthentication(user)
}
