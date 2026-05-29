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
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

// Secondary index for the availability-watch entity.
//
//	availwatch.owner|{ownerId}|{watchId} -> ""   reverse lookup by owner
const (
	availWatchPrimaryPrefix    = "availwatch-"
	availWatchOwnerIndexPrefix = "availwatch.owner|"
)

func availWatchOwnerIndexKey(ownerId, watchId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", availWatchOwnerIndexPrefix, ownerId.String(), watchId.String())
}

func (s *KVStorage) ListAllDomainAvailabilityWatches() (happydns.Iterator[happydns.DomainAvailabilityWatch], error) {
	iter := s.db.Search(availWatchPrimaryPrefix)
	return NewKVIterator[happydns.DomainAvailabilityWatch](s.db, iter), nil
}

func (s *KVStorage) ListDomainAvailabilityWatches(u *happydns.User) ([]*happydns.DomainAvailabilityWatch, error) {
	prefix := fmt.Sprintf("%s%s|", availWatchOwnerIndexPrefix, u.Id.String())
	return listByIndex(s, prefix, s.GetDomainAvailabilityWatch)
}

func (s *KVStorage) getDomainAvailabilityWatch(key string) (*happydns.DomainAvailabilityWatch, error) {
	watch := &happydns.DomainAvailabilityWatch{}
	err := s.db.Get(key, watch)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrDomainAvailabilityWatchNotFound
	}
	return watch, err
}

func (s *KVStorage) GetDomainAvailabilityWatch(id happydns.Identifier) (*happydns.DomainAvailabilityWatch, error) {
	return s.getDomainAvailabilityWatch(fmt.Sprintf("%s%s", availWatchPrimaryPrefix, id.String()))
}

func (s *KVStorage) CreateDomainAvailabilityWatch(w *happydns.DomainAvailabilityWatch) error {
	key, id, err := s.db.FindIdentifierKey(availWatchPrimaryPrefix)
	if err != nil {
		return err
	}

	w.Id = id
	if err := s.db.Put(key, w); err != nil {
		return err
	}
	return s.db.Put(availWatchOwnerIndexKey(w.Owner, w.Id), "")
}

func (s *KVStorage) UpdateDomainAvailabilityWatch(w *happydns.DomainAvailabilityWatch) error {
	primaryKey := fmt.Sprintf("%s%s", availWatchPrimaryPrefix, w.Id.String())

	// Load the previous record to detect owner changes. UpdateDomainAvailabilityWatch
	// is used by the backup restore path where the primary may not exist yet, so
	// a missing old record is not an error.
	old, err := s.GetDomainAvailabilityWatch(w.Id)
	if err != nil && !errors.Is(err, happydns.ErrDomainAvailabilityWatchNotFound) {
		return err
	}

	if err := s.db.Put(primaryKey, w); err != nil {
		return err
	}

	if old != nil && !old.Owner.Equals(w.Owner) {
		if delErr := s.db.Delete(availWatchOwnerIndexKey(old.Owner, old.Id)); delErr != nil {
			log.Printf("UpdateDomainAvailabilityWatch: failed to delete stale owner index for owner %s: %v", old.Owner.String(), delErr)
		}
	}

	return s.db.Put(availWatchOwnerIndexKey(w.Owner, w.Id), "")
}

func (s *KVStorage) DeleteDomainAvailabilityWatch(id happydns.Identifier) error {
	if w, err := s.GetDomainAvailabilityWatch(id); err == nil {
		if delErr := s.db.Delete(availWatchOwnerIndexKey(w.Owner, w.Id)); delErr != nil {
			log.Printf("DeleteDomainAvailabilityWatch: failed to delete owner index for owner %s: %v", w.Owner.String(), delErr)
		}
	}

	return s.db.Delete(fmt.Sprintf("%s%s", availWatchPrimaryPrefix, id.String()))
}

func (s *KVStorage) ClearDomainAvailabilityWatches() error {
	if err := s.clearByPrefix(availWatchOwnerIndexPrefix); err != nil {
		return err
	}

	iter, err := s.ListAllDomainAvailabilityWatches()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if err = s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}

	return iter.Err()
}
