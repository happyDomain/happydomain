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
	"fmt"
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services"
)

func migrateFrom4(s *LevelDBStorage) (err error) {
	err = migrateFrom4_orphanrecords(s)
	if err != nil {
		return
	}

	err = s.Tidy()
	if err != nil {
		return
	}

	return
}

func migrateFrom4_orphanrecords(s *LevelDBStorage) (err error) {
	iter := s.search("domain.zone-")
	for iter.Next() {
		var tmpid string
		fmt.Sscanf(string(iter.Key()), "domain.zone-%s", &tmpid)

		var id happydns.Identifier
		id, err = happydns.NewIdentifierFromString(tmpid)
		if err != nil {
			return fmt.Errorf("unable to determine identifier of %s: %w", iter.Key(), err)
		}

		var zone *happydns.Zone
		zone, err = s.GetZone(id)
		if err != nil {
			return fmt.Errorf("%s: %w", iter.Key(), err)
		}

		changed := false
		for _, zServices := range zone.Services {
			for _, svc := range zServices {
				if orphan, ok := svc.Service.(*svcs.Orphan); ok {
					tmp := strings.Fields(orphan.RR)

					orphan.Type = tmp[0]
					orphan.RR = strings.TrimSpace(orphan.RR[len(tmp[0]):])

					changed = true
				}
			}
		}

		if changed {
			err = s.UpdateZone(zone)
			if err != nil {
				return fmt.Errorf("unable to write %s: %w", iter.Key(), err)
			}
			log.Printf("Migrating v3 -> v4: %s (contains Orphan)...", iter.Key())
		}
	}

	return nil
}
