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
	"fmt"
	"log"
)

type KVMigrationFunc func(s *KVStorage) error

var migrations []KVMigrationFunc = []KVMigrationFunc{
	migrateFrom0,
	migrateFrom1,
	migrateFrom2,
	migrateFrom3,
	migrateFrom4,
	migrateFrom5,
	migrateFrom6,
	migrateFrom7,
}

type Version struct {
	Version int `json:"version"`
}

func (s *KVStorage) SchemaVersion() int {
	return len(migrations)
}

func (s *KVStorage) MigrateSchema() (err error) {
	found := false

	found, err = s.db.Has("version")
	if err != nil {
		return
	}

	var version Version

	if !found {
		version.Version = len(migrations)
		err = s.db.Put("version", version.Version)
		if err != nil {
			return
		}
	}

	err = s.db.Get("version", &version.Version)
	if err != nil {
		return
	}

	if version.Version > len(migrations) {
		return fmt.Errorf("Your database has revision %d, which is newer than the revision this happyDomain version can handle (max DB revision %d). Please update happyDomain", version.Version, len(migrations))
	}

	for v, migration := range migrations[version.Version:] {
		log.Printf("Doing migration from %d to %d", version.Version+v, version.Version+v+1)
		// Do the migration
		if err = migration(s); err != nil {
			return
		}

		// Save the step
		if err = s.db.Put("version", version.Version+v+1); err != nil {
			return
		}
		log.Printf("Migration from %d to %d DONE!", version.Version+v, version.Version+v+1)
	}

	return nil
}
