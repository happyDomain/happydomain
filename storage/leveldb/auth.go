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
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetAuthUsers() (users happydns.UserAuths, err error) {
	iter := s.search("auth-")
	defer iter.Release()

	for iter.Next() {
		var u happydns.UserAuth

		err = decodeData(iter.Value(), &u)
		if err != nil {
			log.Printf("GetAuthUsers: Unable to decode user %q: %s", iter.Key(), err.Error())
		} else {
			users = append(users, &u)
		}
	}

	if len(users) > 0 {
		err = nil
	}

	return
}

func (s *LevelDBStorage) getAuthUser(key string) (u *happydns.UserAuth, err error) {
	u = &happydns.UserAuth{}
	err = s.get(key, &u)
	return
}

func (s *LevelDBStorage) GetAuthUser(id happydns.Identifier) (u *happydns.UserAuth, err error) {
	return s.getAuthUser(fmt.Sprintf("auth-%s", id.String()))
}

func (s *LevelDBStorage) GetAuthUserByEmail(email string) (u *happydns.UserAuth, err error) {
	var users happydns.UserAuths

	users, err = s.GetAuthUsers()
	if err != nil {
		return
	}

	for _, user := range users {
		if user.Email == email {
			u = user
			return
		}
	}

	return nil, fmt.Errorf("Unable to find user with email address '%s'.", email)
}

func (s *LevelDBStorage) AuthUserExists(email string) bool {
	users, err := s.GetAuthUsers()
	if err != nil {
		return false
	}

	for _, user := range users {
		if user.Email == email {
			return true
		}
	}

	return false
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

func (s *LevelDBStorage) TidyAuthUsers() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("auth-")), nil)
	defer iter.Release()

	for iter.Next() {
		userAuth, err := s.getAuthUser(string(iter.Key()))

		if err != nil {
			// Drop unreadable providers
			log.Printf("Deleting unreadable authUser (%s): %v\n", err.Error(), userAuth)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetUser(userAuth.Id)
			if err == leveldb.ErrNotFound {
				// Drop providers of unexistant users
				log.Printf("Deleting orphan authuser (user %s not found): %v\n", userAuth.Id.String(), userAuth)
				err = tx.Delete(iter.Key(), nil)
			}
		}

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
