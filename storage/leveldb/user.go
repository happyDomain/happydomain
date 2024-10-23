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

func (s *LevelDBStorage) GetUsers() (users happydns.Users, err error) {
	iter := s.search("user-")
	defer iter.Release()

	for iter.Next() {
		var u happydns.User

		err = decodeData(iter.Value(), &u)
		if err != nil {
			log.Printf("GetUsers: Unable to decode user %q: %s", iter.Key(), err.Error())
		} else {
			users = append(users, &u)
		}
	}

	if len(users) > 0 {
		err = nil
	}

	return
}

func (s *LevelDBStorage) getUser(key string) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.get(key, &u)
	return
}

func (s *LevelDBStorage) GetUser(id happydns.Identifier) (u *happydns.User, err error) {
	return s.getUser(fmt.Sprintf("user-%s", id.String()))
}

func (s *LevelDBStorage) GetUserByEmail(email string) (u *happydns.User, err error) {
	var users happydns.Users

	users, err = s.GetUsers()
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

func (s *LevelDBStorage) UserExists(email string) bool {
	users, err := s.GetUsers()
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

func (s *LevelDBStorage) DeleteUser(u *happydns.User) error {
	return s.delete(fmt.Sprintf("user-%s", u.Id.String()))
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

func (s *LevelDBStorage) TidyUsers() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user-")), nil)
	defer iter.Release()

	for iter.Next() {
		user, err := s.getUser(string(iter.Key()))

		if err != nil {
			// Drop unreadable providers
			log.Printf("Deleting unreadable user (%s): %v\n", err.Error(), user)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetAuthUser(user.Id)
			if err == leveldb.ErrNotFound {
				// Drop providers of unexistant users
				log.Printf("Deleting orphan user (authuser %s not found): %v\n", user.Id.String(), user)
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
