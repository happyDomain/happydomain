// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"time"

	happydns "git.happydns.org/happyDomain/model"
)

// ListAllAuthUsers returns every auth user in the system. It is intended for
// administrative callers and materialises the underlying iterator into a
// slice so the caller does not need to manage iteration.
func (s *Service) ListAllAuthUsers() ([]*happydns.UserAuth, error) {
	iter, err := s.store.ListAllAuthUsers()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var users []*happydns.UserAuth
	for iter.Next() {
		users = append(users, iter.Item())
	}
	return users, iter.Err()
}

// AdminCreateAuthUser persists user as-is, without any of the registration
// validations performed by CreateAuthUser. Intended for administrative
// callers who already constructed a complete UserAuth value.
func (s *Service) AdminCreateAuthUser(user *happydns.UserAuth) error {
	return s.store.CreateAuthUser(user)
}

// AdminUpdateAuthUser persists changes to user. It is the write-side
// counterpart used by admin endpoints that mutate a UserAuth in memory
// (password reset, raw update) and need to commit the result.
func (s *Service) AdminUpdateAuthUser(user *happydns.UserAuth) error {
	return s.store.UpdateAuthUser(user)
}

// AdminDeleteAuthUser removes user without verifying the current password.
// Sessions belonging to the corresponding user are also closed.
func (s *Service) AdminDeleteAuthUser(user *happydns.UserAuth) error {
	if err := s.store.DeleteAuthUser(user); err != nil {
		return err
	}
	return s.closeUserSessions.ByID(user.Id)
}

// ClearAuthUsers removes every auth user from the database. It is intended
// for administrative callers performing a full reset.
func (s *Service) ClearAuthUsers() error {
	return s.store.ClearAuthUsers()
}

// MarkEmailValidated stamps the user's email as verified now and persists
// the change. Intended for administrative callers bypassing the usual
// validation-link flow.
func (s *Service) MarkEmailValidated(user *happydns.UserAuth) error {
	now := time.Now()
	user.EmailVerification = &now
	return s.store.UpdateAuthUser(user)
}
