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
	userPrimaryPrefix = "user-"
	userEmailPrefix   = "user.email|"
)

// userEmailIndexKey returns the secondary-index key used to look up a user
// by email. The email is lowercased and trimmed before hashing so
// "Foo@Bar.COM" and " foo@bar.com " resolve to the same index entry.
//
// Total key length: len("user.email|") + base64.RawURLEncoding(sha256) =
// 11 + 43 = 54 chars, comfortably under the 64-char limit that some KV
// backends enforce on keys.
func userEmailIndexKey(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])
	return userEmailPrefix + encoded
}

func userPrimaryKey(id happydns.Identifier) string {
	return userPrimaryPrefix + id.String()
}

func (s *KVStorage) ListAllUsers() (happydns.Iterator[happydns.User], error) {
	iter := s.db.Search(userPrimaryPrefix)
	return NewKVIterator[happydns.User](s.db, iter), nil
}

func (s *KVStorage) CountUsers() (int, error) {
	return s.countByPrefix(userPrimaryPrefix)
}

func (s *KVStorage) getUser(key string) (*happydns.User, error) {
	u := &happydns.User{}
	err := s.db.Get(key, &u)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrUserNotFound
	}
	return u, err
}

func (s *KVStorage) GetUser(id happydns.Identifier) (u *happydns.User, err error) {
	return s.getUser(userPrimaryKey(id))
}

// GetUserByEmail resolves a user via the user-email index. Returns
// happydns.ErrUserNotFound when no user matches.
func (s *KVStorage) GetUserByEmail(email string) (*happydns.User, error) {
	var idStr string
	err := s.db.Get(userEmailIndexKey(email), &idStr)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	id, err := happydns.NewIdentifierFromString(idStr)
	if err != nil {
		log.Printf("storage: malformed user-email index value for %q: %v", email, err)
		return nil, happydns.ErrUserNotFound
	}

	user, err := s.GetUser(id)
	if err != nil {
		return nil, err
	}

	// Defend against stale indexes: confirm the resolved record actually
	// carries the queried email before handing it back.
	if !strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
		return nil, happydns.ErrUserNotFound
	}
	return user, nil
}

func (s *KVStorage) UserExists(email string) bool {
	_, err := s.GetUserByEmail(email)
	return err == nil
}

func (s *KVStorage) CreateOrUpdateUser(u *happydns.User) error {
	isNew := u.Id.IsEmpty()

	if isNew {
		// Reject creation if another account already owns the email; the
		// email index is a single key per address and a blind Put below
		// would otherwise overwrite the existing user's index entry,
		// hiding that account from lookup.
		if u.Email != "" {
			if existing, err := s.GetUserByEmail(u.Email); err == nil && existing != nil {
				return happydns.ErrUserAlreadyExist
			} else if err != nil && !errors.Is(err, happydns.ErrUserNotFound) {
				return err
			}
		}

		_, id, err := s.db.FindIdentifierKey(userPrimaryPrefix)
		if err != nil {
			return err
		}
		u.Id = id
	}

	// On update, load the previous record so a changed email can deprecate
	// the stale index entry. Missing old record is not an error: this entry
	// point is also used by the backup restore path.
	var old *happydns.User
	if !isNew {
		prev, err := s.GetUser(u.Id)
		if err != nil && !errors.Is(err, happydns.ErrUserNotFound) {
			return err
		}
		old = prev
	}

	if err := s.db.Put(userPrimaryKey(u.Id), u); err != nil {
		return err
	}

	newIndexKey := userEmailIndexKey(u.Email)
	if old != nil {
		oldIndexKey := userEmailIndexKey(old.Email)
		if oldIndexKey != newIndexKey {
			if delErr := s.db.Delete(oldIndexKey); delErr != nil {
				log.Printf("storage: failed to delete stale user-email index for user %q: %v", u.Id.String(), delErr)
			}
		}
	}

	if err := s.db.Put(newIndexKey, u.Id.String()); err != nil {
		if isNew {
			// Roll back the primary so a failed index write doesn't orphan
			// an account that nobody can resolve by email.
			if delErr := s.db.Delete(userPrimaryKey(u.Id)); delErr != nil {
				log.Printf("storage: orphan user %q after index write failed (rollback also failed: %v)", u.Id.String(), delErr)
			}
		}
		return err
	}
	return nil
}

func (s *KVStorage) DeleteUser(uId happydns.Identifier) error {
	// Best-effort index cleanup: if the primary is already gone we still
	// want the caller's Delete to succeed, and any orphan index entry will
	// be skipped harmlessly by readers and reaped by tidy.
	if u, err := s.GetUser(uId); err == nil {
		if delErr := s.db.Delete(userEmailIndexKey(u.Email)); delErr != nil {
			log.Printf("storage: failed to delete user-email index for user %q: %v", uId.String(), delErr)
		}
	}
	return s.db.Delete(userPrimaryKey(uId))
}

func (s *KVStorage) ClearUsers() error {
	if err := s.ClearSessions(); err != nil {
		return err
	}

	// Wipe the secondary index first; clearByPrefix uses a snapshot
	// iterator so this is safe to do before the primaries.
	if err := s.clearByPrefix(userEmailPrefix); err != nil {
		return err
	}

	iter, err := s.ListAllUsers()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if err := s.db.Delete(fmt.Sprintf("%s%s", userPrimaryPrefix, iter.Item().Id.String())); err != nil {
			return err
		}
	}

	return iter.Err()
}
