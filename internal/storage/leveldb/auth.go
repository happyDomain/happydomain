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

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) ListAllAuthUsers() (storage.Iterator[happydns.UserAuth], error) {
	iter := s.search("auth-")
	return NewLevelDBIterator[happydns.UserAuth](s.db, iter), nil
}

func (s *LevelDBStorage) getAuthUser(key string) (*happydns.UserAuth, error) {
	u := &happydns.UserAuth{}
	err := s.get(key, &u)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, happydns.ErrAuthUserNotFound
	}
	return u, err
}

func (s *LevelDBStorage) GetAuthUser(id happydns.Identifier) (u *happydns.UserAuth, err error) {
	return s.getAuthUser(fmt.Sprintf("auth-%s", id.String()))
}

func (s *LevelDBStorage) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	users, err := s.ListAllAuthUsers()
	if err != nil {
		return nil, err
	}

	for users.Next() {
		user := users.Item()
		if user.Email == email {
			return user, nil
		}
	}

	return nil, fmt.Errorf("Unable to find user with email address '%s'.", email)
}

func (s *LevelDBStorage) AuthUserExists(email string) (bool, error) {
	users, err := s.ListAllAuthUsers()
	if err != nil {
		return false, err
	}

	for users.Next() {
		user := users.Item()
		if user.Email == email {
			return true, nil
		}
	}

	return false, nil
}

func (s *LevelDBStorage) CreateAuthUser(u *happydns.UserAuth) error {
	key, id, err := s.findIdentifierKey("auth-")
	if err != nil {
		return err
	}

	u.Id = id
	return s.put(key, u)
}

func (s *LevelDBStorage) UpdateAuthUser(u *happydns.UserAuth) error {
	return s.put(fmt.Sprintf("auth-%s", u.Id.String()), u)
}

func (s *LevelDBStorage) DeleteAuthUser(u *happydns.UserAuth) error {
	return s.delete(fmt.Sprintf("auth-%s", u.Id.String()))
}

func (s *LevelDBStorage) ClearAuthUsers() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("auth-")), nil)
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
