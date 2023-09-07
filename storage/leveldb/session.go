// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydomain.org
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

func (s *LevelDBStorage) GetSession(id happydns.Identifier) (session *happydns.Session, err error) {
	return s.getSession(fmt.Sprintf("user.session-%s", id.String()))
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
		sessions = append(sessions, &s)
	}

	return
}

func (s *LevelDBStorage) GetUserSessions(user *happydns.User) (sessions []*happydns.Session, err error) {
	iter := s.search("user.session-")
	defer iter.Release()

	for iter.Next() {
		var s happydns.Session

		err = decodeData(iter.Value(), &s)
		if err != nil {
			return
		}
		sessions = append(sessions, &s)
	}

	return
}

func (s *LevelDBStorage) CreateSession(session *happydns.Session) error {
	key, id, err := s.findIdentifierKey("user.session-")
	if err != nil {
		return err
	}

	session.Id = id

	return s.put(key, session)
}

func (s *LevelDBStorage) UpdateSession(session *happydns.Session) error {
	return s.put(fmt.Sprintf("user.session-%s", session.Id.String()), session)
}

func (s *LevelDBStorage) DeleteSession(session *happydns.Session) error {
	return s.delete(fmt.Sprintf("user.session-%s", session.Id.String()))
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
