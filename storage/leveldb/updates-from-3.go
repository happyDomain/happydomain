// Copyright or © or Copr. happyDNS (2023)
//
// contact@happydomain.org
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
