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

	happydns "git.happydns.org/happyDomain/model"
)

// migrateFrom13 backfills the Session.Machine flag, introduced to keep machine
// sessions alive when a user changes their password.
//
// Before this migration the two kinds of sessions could only be told apart by
// their content: a session opened through the web interface always carries the
// values encoded by the session store, whereas a session created through the
// API to be used as a token has none. That is the rule applied here, once, so
// that later code can rely on the explicit flag.
func migrateFrom13(s *KVStorage) error {
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
		if session == nil || session.Content != "" {
			continue
		}

		if !strings.HasPrefix(iter.Key(), sessionPrimaryPrefix) {
			log.Printf("migrateFrom13: skipping session with unexpected key %q", iter.Key())
			continue
		}

		session.Machine = true
		toUpdate = append(toUpdate, update{key: iter.Key(), session: *session})
	}

	if err := iter.Err(); err != nil {
		return err
	}

	for _, u := range toUpdate {
		if err := s.db.Put(u.key, &u.session); err != nil {
			return fmt.Errorf("migrateFrom13: write session %q: %w", u.key, err)
		}
	}

	log.Printf("migrateFrom13: flagged %d machine sessions", len(toUpdate))
	return nil
}
