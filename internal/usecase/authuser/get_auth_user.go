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

package authuser

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// GetAuthUserUsecase handles retrieval of authenticated users by ID.
type GetAuthUserUsecase struct {
	store AuthUserStorage
}

// NewGetAuthUserUsecase creates a new instance of GetAuthUserUsecase.
func NewGetAuthUserUsecase(store AuthUserStorage) *GetAuthUserUsecase {
	return &GetAuthUserUsecase{
		store: store,
	}
}

// ByID retrieves an authenticated user from the storage by their unique identifier.
// Returns the user if found, or an error otherwise.
func (uc *GetAuthUserUsecase) ByID(id happydns.Identifier) (*happydns.UserAuth, error) {
	user, err := uc.store.GetAuthUser(id)
	if err != nil {
		return nil, fmt.Errorf("unable to get user by ID: %w", err)
	}
	return user, nil
}

// ByEmail retrieves an authenticated user from the storage by their email address.
// Returns the user if found, or an error otherwise.
func (uc *GetAuthUserUsecase) ByEmail(email string) (*happydns.UserAuth, error) {
	user, err := uc.store.GetAuthUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("unable to get user by email: %w", err)
	}
	return user, nil
}
