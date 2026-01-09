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

func (s *KVStorage) ListAllUsers() (happydns.Iterator[happydns.User], error) {
	iter := s.db.Search("user-")
	return NewKVIterator[happydns.User](s.db, iter), nil
}

func (s *KVStorage) getUser(key string) (*happydns.User, error) {
	u := &happydns.User{}
	err := s.db.Get(key, &u)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrUserNotFound
	}
	return u, err
}

func (s *KVStorage) GetUser(id happydns.Identifier) (u *happydns.User, err error) {
	return s.getUser(fmt.Sprintf("user-%s", id.String()))
}

func (s *KVStorage) GetUserByEmail(email string) (*happydns.User, error) {
	users, err := s.ListAllUsers()
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

	return nil, happydns.ErrUserNotFound
}

func (s *KVStorage) UserExists(email string) bool {
	users, err := s.ListAllUsers()
	if err != nil {
		return false
	}
	defer users.Close()

	for users.Next() {
		if users.Item().Email == email {
			return true
		}
	}

	return false
}

func (s *KVStorage) CreateOrUpdateUser(u *happydns.User) error {
	if u.Id.IsEmpty() {
		_, id, err := s.db.FindIdentifierKey("user-")
		if err != nil {
			return err
		}

		u.Id = id
	}

	return s.db.Put(fmt.Sprintf("user-%s", u.Id.String()), u)
}

func (s *KVStorage) DeleteUser(uId happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("user-%s", uId.String()))
}

func (s *KVStorage) ClearUsers() error {
	if err := s.ClearSessions(); err != nil {
		return err
	}

	iter, err := s.ListAllUsers()
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
