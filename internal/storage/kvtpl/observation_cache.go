// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

	"git.happydns.org/happyDomain/model"
)

func obsCacheKey(target happydns.CheckTarget, key happydns.ObservationKey) string {
	return fmt.Sprintf("obscache|%s-%s", target.String(), key)
}

func (s *KVStorage) ListAllCachedObservations() (happydns.Iterator[happydns.ObservationCacheEntry], error) {
	iter := s.db.Search("obscache|")
	return NewKVIterator[happydns.ObservationCacheEntry](s.db, iter), nil
}

func (s *KVStorage) GetCachedObservation(target happydns.CheckTarget, key happydns.ObservationKey) (*happydns.ObservationCacheEntry, error) {
	entry := &happydns.ObservationCacheEntry{}
	err := s.db.Get(obsCacheKey(target, key), entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *KVStorage) PutCachedObservation(target happydns.CheckTarget, key happydns.ObservationKey, entry *happydns.ObservationCacheEntry) error {
	return s.db.Put(obsCacheKey(target, key), entry)
}
