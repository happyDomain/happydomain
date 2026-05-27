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
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
)

const (
	sessionPrimaryPrefix = "user.session-"
	sessionUserPrefix    = "user.session-user|"
)

// sessionHash returns the base64-RawURLEncoded SHA-256 of the raw session id.
func sessionHash(id string) string {
	h := sha256.Sum256([]byte(id))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// sessionKey generates a hashed database key for a session ID.
func sessionKey(id string) string {
	return sessionPrimaryPrefix + sessionHash(id)
}

// sessionPrimaryKeyFromHash builds the primary key from an already-hashed id.
func sessionPrimaryKeyFromHash(hash string) string {
	return sessionPrimaryPrefix + hash
}

// sessionUserIndexKey builds the per-user secondary index key for a session,
// keyed by the SHA-256 hash of the raw session id (same hash used for the
// primary key).
func sessionUserIndexKey(uid happydns.Identifier, hash string) string {
	return fmt.Sprintf("%s%s|%s", sessionUserPrefix, uid.String(), hash)
}

// hashFromSessionUserIndexKey extracts the session hash suffix from a user
// index key. Returns the empty string if the key is malformed.
func hashFromSessionUserIndexKey(key string) string {
	rest, ok := strings.CutPrefix(key, sessionUserPrefix)
	if !ok {
		return ""
	}
	_, hash, ok := strings.Cut(rest, "|")
	if !ok {
		return ""
	}
	return hash
}

func (s *KVStorage) ListAllSessions() (happydns.Iterator[happydns.Session], error) {
	// Restrict to the primary prefix only; the user-index prefix is a
	// subspace of "user.session-" and would otherwise be reported here too.
	iter := s.db.Search(sessionPrimaryPrefix)
	return &primarySessionIterator{KVIterator: NewKVIterator[happydns.Session](s.db, iter)}, nil
}

// primarySessionIterator filters out user-index keys that share the
// "user.session-" prefix with the primary records.
type primarySessionIterator struct {
	*KVIterator[happydns.Session]
}

func (it *primarySessionIterator) Next() bool {
	for it.KVIterator.Next() {
		if strings.HasPrefix(it.Key(), sessionUserPrefix) {
			continue
		}
		return true
	}
	return false
}

// NextWithError mirrors Next's filtering so Tidy doesn't observe
// user-index entries (whose values don't decode as Session) and drop them.
func (it *primarySessionIterator) NextWithError() bool {
	for it.KVIterator.NextWithError() {
		if strings.HasPrefix(it.Key(), sessionUserPrefix) {
			continue
		}
		return true
	}
	return false
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

// listSessionsByUserID resolves all sessions for a given user via the
// user secondary index, falling back to skipping the index entry if the
// primary record is gone.
func (s *KVStorage) listSessionsByUserID(userid happydns.Identifier) ([]*happydns.Session, error) {
	prefix := sessionUserPrefix + userid.String() + "|"
	iter := s.db.Search(prefix)
	defer iter.Release()

	var sessions []*happydns.Session
	for iter.Next() {
		hash := hashFromSessionUserIndexKey(iter.Key())
		if hash == "" {
			log.Printf("storage: malformed session index key %q", iter.Key())
			continue
		}
		session := &happydns.Session{}
		if err := s.db.Get(sessionPrimaryKeyFromHash(hash), session); err != nil {
			if errors.Is(err, happydns.ErrNotFound) {
				// Index drift: skip; tidy will clean it up.
				log.Printf("storage: session index %q points to missing primary", iter.Key())
				continue
			}
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *KVStorage) ListAuthUserSessions(user *happydns.UserAuth) ([]*happydns.Session, error) {
	return s.listSessionsByUserID(user.Id)
}

func (s *KVStorage) ListUserSessions(userid happydns.Identifier) ([]*happydns.Session, error) {
	return s.listSessionsByUserID(userid)
}

func (s *KVStorage) UpdateSession(session *happydns.Session) error {
	primary := sessionKey(session.Id)
	hash := sessionHash(session.Id)

	// If the same primary key already exists under a different user, drop
	// the stale user index so it doesn't outlive this update.
	old := &happydns.Session{}
	if err := s.db.Get(primary, old); err == nil && !old.IdUser.IsEmpty() && !old.IdUser.Equals(session.IdUser) {
		if delErr := s.db.Delete(sessionUserIndexKey(old.IdUser, hash)); delErr != nil {
			log.Printf("storage: failed to delete stale session user index for %s: %v", old.IdUser.String(), delErr)
		}
	}

	if err := s.db.Put(primary, session); err != nil {
		return err
	}

	// Only index sessions that belong to a known user; anonymous sessions
	// are still reachable via the primary key.
	if !session.IdUser.IsEmpty() {
		if err := s.db.Put(sessionUserIndexKey(session.IdUser, hash), ""); err != nil {
			return err
		}
	}
	return nil
}

func (s *KVStorage) DeleteSession(id string) error {
	primary := sessionKey(id)

	// Load first so we can clean up the user index. If the primary is gone,
	// fall through to a best-effort delete of the primary key.
	if session, err := s.getSession(primary); err == nil {
		if !session.IdUser.IsEmpty() {
			if delErr := s.db.Delete(sessionUserIndexKey(session.IdUser, sessionHash(id))); delErr != nil {
				log.Printf("storage: failed to delete session user index for %s: %v", session.IdUser.String(), delErr)
			}
		}
	} else if !errors.Is(err, happydns.ErrSessionNotFound) {
		return err
	}

	return s.db.Delete(primary)
}

func (s *KVStorage) ClearSessions() error {
	// Clear the user index space first; if a crash interrupts ClearSessions
	// halfway, leftover primaries remain reachable by id but no longer leak
	// into per-user listings.
	if err := s.clearByPrefix(sessionUserPrefix); err != nil {
		return err
	}

	iter := s.db.Search(sessionPrimaryPrefix)
	defer iter.Release()

	for iter.Next() {
		// Be defensive against future index keys sharing the prefix.
		if strings.HasPrefix(iter.Key(), sessionUserPrefix) {
			continue
		}
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}

	return iter.Err()
}
