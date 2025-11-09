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
	"errors"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
)

func (s *LevelDBStorage) ListAllZones() (happydns.Iterator[happydns.ZoneMessage], error) {
	iter := s.search("domain.zone-")
	return NewLevelDBIterator[happydns.ZoneMessage](s.db, iter), nil
}

func (s *LevelDBStorage) GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error) {
	z := &happydns.ZoneMessage{}
	err := s.get(fmt.Sprintf("domain.zone-%s", id.String()), &z)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, happydns.ErrZoneNotFound
	}
	return z, err
}

func (s *LevelDBStorage) getZoneMeta(id string) (z *happydns.ZoneMeta, err error) {
	z = &happydns.ZoneMeta{}
	err = s.get(id, z)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, happydns.ErrZoneNotFound
	}
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

func (s *LevelDBStorage) UpdateZoneMessage(z *happydns.ZoneMessage) error {
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
