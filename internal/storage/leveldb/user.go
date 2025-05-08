// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) ListAllUsers() (happydns.Iterator[happydns.User], error) {
	iter := s.search("user-")
	return NewLevelDBIterator[happydns.User](s.db, iter), nil
}

func (s *LevelDBStorage) getUser(key string) (*happydns.User, error) {
	u := &happydns.User{}
	err := s.get(key, &u)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, happydns.ErrUserNotFound
	}
	return u, err
}

func (s *LevelDBStorage) GetUser(id happydns.Identifier) (u *happydns.User, err error) {
	return s.getUser(fmt.Sprintf("user-%s", id.String()))
}

func (s *LevelDBStorage) GetUserByEmail(email string) (*happydns.User, error) {
	users, err := s.ListAllUsers()
	if err != nil {
		return nil, err
	}

	for users.Next() {
		user := users.Item()
		if user.Email == email {
			return user, nil
		}
	}

	return nil, happydns.ErrUserNotFound
}

func (s *LevelDBStorage) UserExists(email string) bool {
	users, err := s.ListAllUsers()
	if err != nil {
		return false
	}

	for users.Next() {
		if users.Item().Email == email {
			return true
		}
	}

	return false
}

func (s *LevelDBStorage) CreateOrUpdateUser(u *happydns.User) error {
	if u.Id.IsEmpty() {
		_, id, err := s.findIdentifierKey("user-")
		if err != nil {
			return err
		}

		u.Id = id
	}

	return s.put(fmt.Sprintf("user-%s", u.Id.String()), u)
}

func (s *LevelDBStorage) DeleteUser(uId happydns.Identifier) error {
	return s.delete(fmt.Sprintf("user-%s", uId.String()))
}

func (s *LevelDBStorage) ClearUsers() error {
	if err := s.ClearSessions(); err != nil {
		return err
	}

	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user-")), nil)
	defer iter.Release()

	for iter.Next() {
		err = tx.Delete(iter.Key(), nil)
		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}
