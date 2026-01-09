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

package database

import (
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListAllAuthUsers() (happydns.Iterator[happydns.UserAuth], error) {
	iter := s.db.Search("auth-")
	return NewKVIterator[happydns.UserAuth](s.db, iter), nil
}

func (s *KVStorage) getAuthUser(key string) (*happydns.UserAuth, error) {
	u := &happydns.UserAuth{}
	err := s.db.Get(key, &u)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrAuthUserNotFound
	}
	return u, err
}

func (s *KVStorage) GetAuthUser(id happydns.Identifier) (u *happydns.UserAuth, err error) {
	return s.getAuthUser(fmt.Sprintf("auth-%s", id.String()))
}

func (s *KVStorage) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	users, err := s.ListAllAuthUsers()
	if err != nil {
		return nil, err
	}
	defer users.Close()

	for users.Next() {
		user := users.Item()
		if user.Email == email {
			return user, nil
		}
	}

	return nil, fmt.Errorf("Unable to find user with email address '%s'.", email)
}

func (s *KVStorage) AuthUserExists(email string) (bool, error) {
	users, err := s.ListAllAuthUsers()
	if err != nil {
		return false, err
	}
	defer users.Close()

	for users.Next() {
		user := users.Item()
		if user.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func (s *KVStorage) CreateAuthUser(u *happydns.UserAuth) error {
	key, id, err := s.db.FindIdentifierKey("auth-")
	if err != nil {
		return err
	}

	u.Id = id
	return s.db.Put(key, u)
}

func (s *KVStorage) UpdateAuthUser(u *happydns.UserAuth) error {
	return s.db.Put(fmt.Sprintf("auth-%s", u.Id.String()), u)
}

func (s *KVStorage) DeleteAuthUser(u *happydns.UserAuth) error {
	return s.db.Delete(fmt.Sprintf("auth-%s", u.Id.String()))
}

func (s *KVStorage) ClearAuthUsers() error {
	iter, err := s.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		err = s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return nil
}
