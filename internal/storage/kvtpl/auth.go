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
	authPrimaryPrefix = "auth-"
	authEmailPrefix   = "auth.email|"
)

// authEmailIndexKey returns the secondary-index key used to look up an
// auth user by email. The email is lowercased and trimmed before hashing
// so "Foo@Bar.COM" and " foo@bar.com " resolve to the same index entry.
//
// Total key length: len("auth.email|") + base64.RawURLEncoding(sha256) =
// 11 + 43 = 54 chars, comfortably under the 64-char limit that some KV
// backends enforce on keys.
func authEmailIndexKey(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])
	return authEmailPrefix + encoded
}

func authPrimaryKey(id happydns.Identifier) string {
	return authPrimaryPrefix + id.String()
}

func (s *KVStorage) ListAllAuthUsers() (happydns.Iterator[happydns.UserAuth], error) {
	iter := s.db.Search(authPrimaryPrefix)
	return NewKVIterator[happydns.UserAuth](s.db, iter), nil
}

func (s *KVStorage) getAuthUser(key string) (*happydns.UserAuth, error) {
	u := &happydns.UserAuth{}
	err := s.db.Get(key, &u)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrAuthUserNotFound
	}
	return u, err
}

func (s *KVStorage) GetAuthUser(id happydns.Identifier) (u *happydns.UserAuth, err error) {
	return s.getAuthUser(authPrimaryKey(id))
}

// GetAuthUserByEmail resolves a user via the auth-email index. Returns
// happydns.ErrAuthUserNotFound when no user matches.
func (s *KVStorage) GetAuthUserByEmail(email string) (*happydns.UserAuth, error) {
	var idStr string
	err := s.db.Get(authEmailIndexKey(email), &idStr)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrAuthUserNotFound
	}
	if err != nil {
		return nil, err
	}

	id, err := happydns.NewIdentifierFromString(idStr)
	if err != nil {
		log.Printf("storage: malformed auth-email index value for %q: %v", email, err)
		return nil, happydns.ErrAuthUserNotFound
	}

	user, err := s.GetAuthUser(id)
	if err != nil {
		return nil, err
	}

	// Defend against stale indexes: confirm the resolved record actually
	// carries the queried email before handing it back.
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
		return nil, happydns.ErrAuthUserNotFound
	}
	return user, nil
}

func (s *KVStorage) AuthUserExists(email string) (bool, error) {
	_, err := s.GetAuthUserByEmail(email)
	if errors.Is(err, happydns.ErrAuthUserNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *KVStorage) CreateAuthUser(u *happydns.UserAuth) error {
	key, id, err := s.db.FindIdentifierKey(authPrimaryPrefix)
	if err != nil {
		return err
	}

	u.Id = id
	if err := s.db.Put(key, u); err != nil {
		return err
	}

	if err := s.db.Put(authEmailIndexKey(u.Email), u.Id.String()); err != nil {
		// Roll back the primary so a failed index write doesn't orphan
		// an account that nobody can log in to.
		if delErr := s.db.Delete(key); delErr != nil {
			log.Printf("storage: orphan auth user %q after index write failed (rollback also failed: %v)", u.Id.String(), delErr)
		}
		return err
	}
	return nil
}

func (s *KVStorage) UpdateAuthUser(u *happydns.UserAuth) error {
	// Load the old record so a changed email can deprecate the stale index entry.
	old, err := s.GetAuthUser(u.Id)
	if err != nil && !errors.Is(err, happydns.ErrAuthUserNotFound) {
		return err
	}

	if err := s.db.Put(authPrimaryKey(u.Id), u); err != nil {
		return err
	}

	newIndexKey := authEmailIndexKey(u.Email)
	if old != nil {
		oldIndexKey := authEmailIndexKey(old.Email)
		if oldIndexKey != newIndexKey {
			if delErr := s.db.Delete(oldIndexKey); delErr != nil {
				log.Printf("storage: failed to delete stale auth-email index for user %q: %v", u.Id.String(), delErr)
			}
		}
	}

	if err := s.db.Put(newIndexKey, u.Id.String()); err != nil {
		return err
	}
	return nil
}

func (s *KVStorage) DeleteAuthUser(u *happydns.UserAuth) error {
	// Delete the index first so a partial failure hides the account
	// rather than leaving it visible-but-broken.
	if err := s.db.Delete(authEmailIndexKey(u.Email)); err != nil {
		log.Printf("storage: failed to delete auth-email index for user %q: %v", u.Id.String(), err)
	}
	return s.db.Delete(authPrimaryKey(u.Id))
}

func (s *KVStorage) ClearAuthUsers() error {
	// Wipe the secondary index first; clearByPrefix uses a snapshot
	// iterator so this is safe to do before the primaries.
	if err := s.clearByPrefix(authEmailPrefix); err != nil {
		return err
	}

	iter, err := s.ListAllAuthUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if err := s.db.Delete(fmt.Sprintf("%s%s", authPrimaryPrefix, iter.Item().Id.String())); err != nil {
			return err
		}
	}

	return iter.Err()
}
