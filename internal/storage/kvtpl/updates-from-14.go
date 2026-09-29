// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"strings"

	"git.happydns.org/happyDomain/model"
)

// migrateFrom14 stops storing session tokens.
//
// Session.Id used to hold the token itself: the primary key was already its
// hash, but the stored value carried the token in clear. The record now holds
// the public identifier, which is the hash found in the key. Tokens held by
// clients are unaffected: they still hash to the same key.
func migrateFrom14(s *KVStorage) error {
	iter, err := s.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	type update struct {
		key     string
		session happydns.Session
	}
	var toUpdate []update

	for iter.Next() {
		session := iter.Item()
		if session == nil {
			continue
		}

		if !strings.HasPrefix(iter.Key(), sessionPrimaryPrefix) {
			log.Printf("migrateFrom14: skipping session with unexpected key %q", iter.Key())
			continue
		}
		hash := strings.TrimPrefix(iter.Key(), sessionPrimaryPrefix)

		if session.Id == hash {
			continue
		}
		if happydns.SessionIDFromToken(session.Id) != hash {
			// The key is what lookups use, so it stays authoritative; the
			// stored Id is overwritten all the same, whatever it held.
			log.Printf("migrateFrom14: session %q held an identifier not matching its key", iter.Key())
		}

		session.Id = hash
		toUpdate = append(toUpdate, update{key: iter.Key(), session: *session})
	}
	if err := iter.Err(); err != nil {
		return err
	}

	for _, u := range toUpdate {
		if err := s.db.Put(u.key, &u.session); err != nil {
			return fmt.Errorf("migrateFrom14: write session %q: %w", u.key, err)
		}
	}

	log.Printf("migrateFrom14: removed the token from %d stored sessions", len(toUpdate))
	return nil
}
