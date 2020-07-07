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
	"strconv"
	"strings"

	"git.happydns.org/happydns/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) GetZone(id int64) (z *happydns.Zone, err error) {
	z = &happydns.Zone{}
	err = s.get(fmt.Sprintf("domain.zone-%d", id), &z)
	return
}

func (s *LevelDBStorage) getZoneMeta(id string) (z *happydns.ZoneMeta, err error) {
	z = &happydns.ZoneMeta{}
	err = s.get(id, z)
	return
}

func (s *LevelDBStorage) GetZoneMeta(id int64) (z *happydns.ZoneMeta, err error) {
	z, err = s.getZoneMeta(fmt.Sprintf("domain.zone-%d", id))
	return
}

func (s *LevelDBStorage) CreateZone(z *happydns.Zone) error {
	key, id, err := s.findInt63Key("domain.zone-")
	if err != nil {
		return err
	}

	z.Id = id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateZone(z *happydns.Zone) error {
	return s.put(fmt.Sprintf("domain.zone-%d", z.Id), z)
}

func (s *LevelDBStorage) DeleteZone(z *happydns.Zone) error {
	return s.delete(fmt.Sprintf("domain.zone-%d", z.Id))
}

func (s *LevelDBStorage) ClearZones() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain.zone-")), nil)
	defer iter.Release()

	for iter.Next() {
		err = tx.Delete(iter.Key(), nil)
		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}

func (s *LevelDBStorage) TidyZones() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain-")), nil)
	defer iter.Release()

	var referencedZones []int64

	for iter.Next() {
		domain, _ := s.getDomain(string(iter.Key()))

		if domain != nil {
			for _, zh := range domain.ZoneHistory {
				referencedZones = append(referencedZones, zh)
			}
		}
	}

	iter = tx.NewIterator(util.BytesPrefix([]byte("domain.zone-")), nil)
	defer iter.Release()

	for iter.Next() {
		if zoneId, err := strconv.ParseInt(strings.TrimPrefix(string(iter.Key()), "domain.zone-"), 10, 64); err != nil {
			// Drop zones with invalid ID
			log.Printf("Deleting unindentified zone: key=%s\n", iter.Key())
			err = tx.Delete(iter.Key(), nil)
		} else {
			foundZone := false
			for _, zid := range referencedZones {
				if zid == zoneId {
					foundZone = true
					break
				}
			}

			if !foundZone {
				// Drop orphan zones
				log.Printf("Deleting orphan zone: %d\n", zoneId)
				err = tx.Delete(iter.Key(), nil)
			}
		}

		if err != nil {
			tx.Discard()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Discard()
		return err
	}

	return nil
}
