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
	"fmt"
	"log"
)

type LevelDBMigrationFunc func(s *LevelDBStorage) error

var migrations []LevelDBMigrationFunc = []LevelDBMigrationFunc{
	migrateFrom0,
	migrateFrom1,
	migrateFrom2,
	migrateFrom3,
	migrateFrom4,
	migrateFrom5,
	migrateFrom6,
}

func (s *LevelDBStorage) SchemaVersion() int {
	return len(migrations)
}

func (s *LevelDBStorage) DoMigration() (err error) {
	found := false

	found, err = s.db.Has([]byte("version"), nil)
	if err != nil {
		return
	}

	var version int

	if !found {
		version = len(migrations)
		err = s.put("version", version)
		if err != nil {
			return
		}
	}

	err = s.get("version", &version)
	if err != nil {
		return
	}

	if version > len(migrations) {
		return fmt.Errorf("Your database has revision %d, which is newer than the revision this happyDomain version can handle (max DB revision %d). Please update happyDomain", version, len(migrations))
	}

	for v, migration := range migrations[version:] {
		log.Printf("Doing migration from %d to %d", version+v, version+v+1)
		// Do the migration
		if err = migration(s); err != nil {
			return
		}

		// Save the step
		if err = s.put("version", version+v+1); err != nil {
			return
		}
		log.Printf("Migration from %d to %d DONE!", version+v, version+v+1)
	}

	return nil
}
