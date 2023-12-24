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
	"bytes"
	"fmt"
	"log"
)

func migrateFrom3(s *LevelDBStorage) (err error) {
	err = migrateFrom3_records(s)
	if err != nil {
		return
	}

	err = s.Tidy()
	if err != nil {
		return
	}

	return
}

func migrateFrom3_records(s *LevelDBStorage) (err error) {
	TypeStr := []byte("\"_svctype\":\"abstract.Origin\"")

	iter := s.search("domain.zone-")
	for iter.Next() {
		zonestr, err := s.db.Get(iter.Key(), nil)
		if err != nil {
			return fmt.Errorf("unable to find/decode %s: %w", iter.Key(), err)
		}

		if bytes.Contains(zonestr, TypeStr) {
			migstr := zonestr
			migstr = bytes.Replace(migstr, []byte("000000000,\"retry\":"), []byte(",\"retry\":"), 1)
			migstr = bytes.Replace(migstr, []byte("000000000,\"expire\":"), []byte(",\"expire\":"), 1)
			migstr = bytes.Replace(migstr, []byte("000000000,\"nxttl\":"), []byte(",\"nxttl\":"), 1)
			migstr = bytes.Replace(migstr, []byte("000000000,\"ns\":"), []byte(",\"ns\":"), 1)

			if !bytes.Equal(migstr, zonestr) {
				err = s.db.Put(iter.Key(), migstr, nil)
				if err != nil {
					return fmt.Errorf("unable to write %s: %w", iter.Key(), err)
				}
				log.Printf("Migrating v2 -> v3: %s (contains Origin)...", iter.Key())
			}
		}
	}

	return
}
