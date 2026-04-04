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

func (s *KVStorage) ListAllSnapshots() (happydns.Iterator[happydns.ObservationSnapshot], error) {
	iter := s.db.Search("chcksnap|")
	return NewKVIterator[happydns.ObservationSnapshot](s.db, iter), nil
}

func (s *KVStorage) GetSnapshot(snapID happydns.Identifier) (*happydns.ObservationSnapshot, error) {
	snap := &happydns.ObservationSnapshot{}
	err := s.db.Get(fmt.Sprintf("chcksnap|%s", snapID.String()), snap)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrSnapshotNotFound
	}
	return snap, err
}

func (s *KVStorage) CreateSnapshot(snap *happydns.ObservationSnapshot) error {
	key, id, err := s.db.FindIdentifierKey("chcksnap|")
	if err != nil {
		return err
	}
	snap.Id = id
	return s.db.Put(key, snap)
}

func (s *KVStorage) DeleteSnapshot(snapID happydns.Identifier) error {
	return s.db.Delete(fmt.Sprintf("chcksnap|%s", snapID.String()))
}

func (s *KVStorage) ClearSnapshots() error {
	iter, err := s.ListAllSnapshots()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}
	return nil
}
