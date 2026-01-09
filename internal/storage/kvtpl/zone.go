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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListAllZones() (happydns.Iterator[happydns.ZoneMessage], error) {
	iter := s.db.Search("domain.zone-")
	return NewKVIterator[happydns.ZoneMessage](s.db, iter), nil
}

func (s *KVStorage) GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error) {
	z := &happydns.ZoneMessage{}
	err := s.db.Get(fmt.Sprintf("domain.zone-%s", id.String()), &z)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrZoneNotFound
	}
	return z, err
}

func (s *KVStorage) getZoneMeta(id string) (z *happydns.ZoneMeta, err error) {
	z = &happydns.ZoneMeta{}
	err = s.db.Get(id, z)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrZoneNotFound
	}
	return
}

func (s *KVStorage) GetZoneMeta(id happydns.Identifier) (z *happydns.ZoneMeta, err error) {
	z, err = s.getZoneMeta(fmt.Sprintf("domain.zone-%s", id.String()))
	return
}

func (s *KVStorage) CreateZone(z *happydns.Zone) error {
	key, id, err := s.db.FindIdentifierKey("domain.zone-")
	if err != nil {
		return err
	}

	z.Id = id
	return s.db.Put(key, z)
}

func (s *KVStorage) UpdateZone(z *happydns.Zone) error {
	return s.db.Put(fmt.Sprintf("domain.zone-%s", z.Id.String()), z)
}

func (s *KVStorage) UpdateZoneMessage(z *happydns.ZoneMessage) error {
	return s.db.Put(fmt.Sprintf("domain.zone-%s", z.Id.String()), z)
}

func (s *KVStorage) DeleteZone(id happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("domain.zone-%s", id.String()))
}

func (s *KVStorage) ClearZones() error {
	iter, err := s.ListAllZones()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		err = s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return nil
}
