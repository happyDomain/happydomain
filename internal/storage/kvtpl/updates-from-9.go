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

	"git.happydns.org/happyDomain/model"
)

func migrateFrom9(s *KVStorage) (err error) {
	sessions, err := s.ListAllSessions()
	if err != nil {
		return err
	}

	for sessions.Next() {
		session := sessions.Item()

		// Records written by this very migration live under the same
		// prefix, already keyed by their public identifier.
		if happydns.IsValidSessionPublicID(session.Id) && sessions.Key() == sessionPrimaryKeyFromHash(session.Id) {
			continue
		}

		if len(session.Id) != 103 {
			err = sessions.DropItem()
			if err != nil {
				return fmt.Errorf("unable to drop invalid session: %s: %w", session.Id, err)
			}
			log.Printf("Drop invalid session identifier: %s", session.Id)
			continue
		}

		// Session.Id held the token when this migration was written; store
		// the record under its public identifier instead, as it is now.
		migrated := *session
		migrated.Id = happydns.SessionIDFromToken(session.Id)

		err := s.UpdateSession(&migrated)
		if err != nil {
			return err
		}
		log.Printf("Migrated session %s[...]", session.Id[:10])
		err = sessions.DropItem()
		if err != nil {
			log.Printf("Unable to delete original session %s[...]: %s", session.Id[:10], err.Error())
		}
	}

	if err := sessions.Err(); err != nil {
		return err
	}

	return nil
}
