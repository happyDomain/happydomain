// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package database

import (
	"fmt"

	"git.happydns.org/happydns/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetUsers() (users happydns.Users, err error) {
	iter := s.search("user-")
	defer iter.Release()

	for iter.Next() {
		var u happydns.User

		err = decodeData(iter.Value(), &u)
		if err != nil {
			return
		}
		users = append(users, &u)
	}

	return
}

func (s *LevelDBStorage) GetUser(id int) (u *happydns.User, err error) {
	u = &happydns.User{}
	err = s.get(fmt.Sprintf("user-%d", id), &u)
	return
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

func (s *LevelDBStorage) CreateUser(u *happydns.User) error {
	key, id, err := s.findInt63Key("user-")
	if err != nil {
		return err
	}

	u.Id = id
	return s.put(key, u)
}

func (s *LevelDBStorage) UpdateUser(u *happydns.User) error {
	return s.put(fmt.Sprintf("user-%d", u.Id), u)
}

func (s *LevelDBStorage) DeleteUser(u *happydns.User) error {
	return s.delete(fmt.Sprintf("user-%d", u.Id))
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
