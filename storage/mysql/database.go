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

package database // import "happydns.org/database"

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStorage struct {
	db *sql.DB
}

// NewMySQLStorage establishes the connection to the database
func NewMySQLStorage(dsn string) (*MySQLStorage, error) {
	if db, err := sql.Open("mysql", dsn+"?parseTime=true&foreign_key_checks=1"); err != nil {
		return nil, err
	} else {
		_, err := db.Exec(`SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO';`)
		for i := 0; err != nil && i < 45; i += 1 {
			if _, err = db.Exec(`SET SESSION sql_mode = 'STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO';`); err != nil && i <= 45 {
				log.Println("An error occurs when trying to connect to DB, will retry in 2 seconds: ", err)
				time.Sleep(2 * time.Second)
			}
		}

		if err != nil {
			return nil, err
		}

		return &MySQLStorage{db}, nil
	}
}

func (s *MySQLStorage) DoMigration() error {
	var currentVersion uint16
	s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&currentVersion)

	log.Println("Current schema version:", currentVersion)
	log.Println("Latest schema version:", schemaVersion)

	for version := currentVersion + 1; version <= schemaVersion; version++ {
		log.Println("Migrating to version:", version)

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		rawSQL := schemaRevisions[version]
		for _, request := range strings.Split(rawSQL, ";") {
			if len(strings.TrimSpace(request)) == 0 {
				continue
			}
			_, err = tx.Exec(request)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		if _, err := tx.Exec(`delete from schema_version`); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return err
		}
	}

	return nil
}

func (s *MySQLStorage) Close() error {
	return s.db.Close()
}
