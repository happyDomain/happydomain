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
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// sessionKey generates a hashed database key for a session ID
func sessionKey(id string) string {
	hash := sha256.Sum256([]byte(id))
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	return fmt.Sprintf("user.session-%s", encoded)
}

func (s *KVStorage) ListAllSessions() (happydns.Iterator[happydns.Session], error) {
	iter := s.db.Search("user.session-")
	return NewKVIterator[happydns.Session](s.db, iter), nil
}

func (s *KVStorage) getSession(id string) (*happydns.Session, error) {
	session := &happydns.Session{}
	err := s.db.Get(id, &session)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrSessionNotFound
	}
	return session, err
}

func (s *KVStorage) GetSession(id string) (session *happydns.Session, err error) {
	return s.getSession(sessionKey(id))
}

func (s *KVStorage) ListAuthUserSessions(user *happydns.UserAuth) (sessions []*happydns.Session, err error) {
	iter := s.db.Search("user.session-")
	defer iter.Release()

	for iter.Next() {
		var session happydns.Session

		err = s.db.DecodeData(iter.Value(), &session)
		if err != nil {
			return
		}
		if session.IdUser.Equals(user.Id) {
			sessions = append(sessions, &session)
		}
	}

	return
}

func (s *KVStorage) ListUserSessions(userid happydns.Identifier) (sessions []*happydns.Session, err error) {
	iter := s.db.Search("user.session-")
	defer iter.Release()

	for iter.Next() {
		var session happydns.Session

		err = s.db.DecodeData(iter.Value(), &session)
		if err != nil {
			return
		}
		if session.IdUser.Equals(userid) {
			sessions = append(sessions, &session)
		}
	}

	return
}

func (s *KVStorage) UpdateSession(session *happydns.Session) error {
	return s.db.Put(sessionKey(session.Id), session)
}

func (s *KVStorage) DeleteSession(id string) error {
	return s.db.Delete(sessionKey(id))
}

func (s *KVStorage) ClearSessions() error {
	iter := s.db.Search("user.session-")
	defer iter.Release()

	for iter.Next() {
		err := s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return nil
}
