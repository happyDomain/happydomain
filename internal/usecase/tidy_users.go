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
	"log"
	"time"

	"git.happydns.org/happyDomain/model"
)

func (tu *tidyUpUsecase) TidyAuthUsers(dropInvalid bool) error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(userAuth *happydns.UserAuth) error {
		_, err := tu.store.GetUser(userAuth.Id)
		if errors.Is(err, happydns.ErrUserNotFound) && time.Since(userAuth.CreatedAt) > 24*time.Hour {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan authuser (user %s not found): %v\n", userAuth.Id.String(), userAuth)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (tu *tidyUpUsecase) TidyUsers(dropInvalid bool) error {
	iter, err := tu.store.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(authUser *happydns.UserAuth) error {
		if authUser.EmailVerification == nil && authUser.LastLoggedIn == nil && time.Since(authUser.CreatedAt) > 7*24*time.Hour {
			log.Printf("Deleting user with unverified email and no login (created %s): %s\n", authUser.CreatedAt.Format(time.RFC3339), authUser.Email)
			if err := tu.store.DeleteUser(authUser.Id); err != nil && !errors.Is(err, happydns.ErrUserNotFound) {
				return err
			}
			if err := iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
