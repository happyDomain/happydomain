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
	"strings"

	"git.happydns.org/happyDomain/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) ListAllZones() (zones []*happydns.ZoneMessage, err error) {
	iter := s.search("domain.zone-")
	defer iter.Release()

	for iter.Next() {
		var zone happydns.ZoneMessage

		err = decodeData(iter.Value(), &zone)
		if err != nil {
			return
		}

		zones = append(zones, &zone)
	}

	return
}

func (s *LevelDBStorage) GetZone(id happydns.Identifier) (z *happydns.ZoneMessage, err error) {
	z = &happydns.ZoneMessage{}
	err = s.get(fmt.Sprintf("domain.zone-%s", id.String()), &z)
	return
}

func (s *LevelDBStorage) getZoneMeta(id string) (z *happydns.ZoneMeta, err error) {
	z = &happydns.ZoneMeta{}
	err = s.get(id, z)
	return
}

func (s *LevelDBStorage) GetZoneMeta(id happydns.Identifier) (z *happydns.ZoneMeta, err error) {
	z, err = s.getZoneMeta(fmt.Sprintf("domain.zone-%s", id.String()))
	return
}

func (s *LevelDBStorage) CreateZone(z *happydns.Zone) error {
	key, id, err := s.findIdentifierKey("domain.zone-")
	if err != nil {
		return err
	}

	z.Id = id
	return s.put(key, z)
}

func (s *LevelDBStorage) UpdateZone(z *happydns.Zone) error {
	return s.put(fmt.Sprintf("domain.zone-%s", z.Id.String()), z)
}

func (s *LevelDBStorage) DeleteZone(id happydns.Identifier) error {
	return s.delete(fmt.Sprintf("domain.zone-%s", id.String()))
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

	var referencedZones []happydns.Identifier

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
		if zoneId, err := happydns.NewIdentifierFromString(strings.TrimPrefix(string(iter.Key()), "domain.zone-")); err != nil {
			// Drop zones with invalid ID
			log.Printf("Deleting unindentified zone: key=%s\n", iter.Key())
			err = tx.Delete(iter.Key(), nil)
		} else {
			foundZone := false
			for _, zid := range referencedZones {
				if zid.Equals(zoneId) {
					foundZone = true
					break
				}
			}

			if !foundZone {
				// Drop orphan zones
				log.Printf("Deleting orphan zone: %s\n", zoneId.String())
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
