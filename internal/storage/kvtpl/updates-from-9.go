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
	"log"
)

func migrateFrom9(s *KVStorage) (err error) {
	sessions, err := s.ListAllSessions()
	if err != nil {
		return err
	}

	for sessions.Next() {
		session := sessions.Item()
		err := s.UpdateSession(session)
		if err != nil {
			return err
		}
		log.Printf("Migrated session %s[...]", session.Id[:10])
		err = sessions.DropItem()
		if err != nil {
			log.Printf("Unable to delete original session %s[...]: %s", session.Id[:10], err.Error())
		}
	}

	return nil
}
