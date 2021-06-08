// Copyright or © or Copr. happyDNS (2020)
//
// contact@happydns.org
//
// This software is a computer program whose purpose is to provide a modern
// interface to interact with DNS systems.
//
// This software is governed by the CeCILL license under French law and abiding
// by the rules of distribution of free software.  You can use, modify and/or
// redistribute the software under the terms of the CeCILL license as
// circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and rights to copy, modify
// and redistribute granted by the license, users are provided only with a
// limited warranty and the software's author, the holder of the economic
// rights, and the successive licensors have only limited liability.
//
// In this respect, the user's attention is drawn to the risks associated with
// loading, using, modifying and/or developing or reproducing the software by
// the user in light of its specific status of free software, that may mean
// that it is complicated to manipulate, and that also therefore means that it
// is reserved for developers and experienced professionals having in-depth
// computer knowledge. Users are therefore encouraged to load and test the
// software's suitability as regards their requirements in conditions enabling
// the security of their systems and/or data to be ensured and, more generally,
// to use and operate it in the same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL license and that you accept its terms.

package database

import (
	"fmt"
	"log"
)

type LevelDBMigrationFunc func(s *LevelDBStorage) error

var migrations []LevelDBMigrationFunc = []LevelDBMigrationFunc{
	migrateFrom0,
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
		return fmt.Errorf("Your database has revision %d, which is newer than the revision this happyDNS version can handle (max DB revision %d). Please update happyDNS", version, len(migrations))
	}

	for v, migration := range migrations[version:] {
		log.Printf("Doing migration from %d to %d", v, v+1)
		// Do the migration
		if err = migration(s); err != nil {
			return
		}

		// Save the step
		if err = s.put("version", v+1); err != nil {
			return
		}
		log.Printf("Migration from %d to %d DONE!", v, v+1)
	}

	return nil
}
