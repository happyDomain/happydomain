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

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) getSession(id string) (session *happydns.Session, err error) {
	session = &happydns.Session{}
	err = s.get(id, &session)
	return
}

func (s *LevelDBStorage) GetSession(id string) (session *happydns.Session, err error) {
	return s.getSession(fmt.Sprintf("user.session-%s", id))
}

func (s *LevelDBStorage) GetAuthUserSessions(user *happydns.UserAuth) (sessions []*happydns.Session, err error) {
	iter := s.search("user.session-")
	defer iter.Release()

	for iter.Next() {
		var s happydns.Session

		err = decodeData(iter.Value(), &s)
		if err != nil {
			return
		}
		if s.IdUser.Equals(user.Id) {
			sessions = append(sessions, &s)
		}
	}

	return
}

func (s *LevelDBStorage) GetUserSessions(userid happydns.Identifier) (sessions []*happydns.Session, err error) {
	iter := s.search("user.session-")
	defer iter.Release()

	for iter.Next() {
		var s happydns.Session

		err = decodeData(iter.Value(), &s)
		if err != nil {
			return
		}
		if s.IdUser.Equals(userid) {
			sessions = append(sessions, &s)
		}
	}

	return
}

func (s *LevelDBStorage) UpdateSession(session *happydns.Session) error {
	return s.put(fmt.Sprintf("user.session-%s", session.Id), session)
}

func (s *LevelDBStorage) DeleteSession(id string) error {
	return s.delete(fmt.Sprintf("user.session-%s", id))
}

func (s *LevelDBStorage) ClearSessions() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user.session-")), nil)
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

func (s *LevelDBStorage) TidySessions() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("user.session-")), nil)
	defer iter.Release()

	for iter.Next() {
		session, err := s.getSession(string(iter.Key()))

		if err != nil {
			// Drop unreadable sessions
			log.Printf("Deleting unreadable session (%s): %v\n", err.Error(), session)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetUser(session.IdUser)
			if err == leveldb.ErrNotFound {
				// Drop session from unexistant users
				log.Printf("Deleting orphan session (user %s not found): %v\n", session.IdUser.String(), session)
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
