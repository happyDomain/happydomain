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

	"git.happydns.org/happyDomain/model"
)

// Same primary+per-user-index layout as notification_preference.go.

const (
	domwatchPrimaryPrefix = "domwatch|"
	domwatchUserPrefix    = "domwatch-user|"
)

func domwatchPrimaryKey(id happydns.Identifier) string {
	return domwatchPrimaryPrefix + id.String()
}

func domwatchUserKey(userId, watchId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", domwatchUserPrefix, userId.String(), watchId.String())
}

func (s *KVStorage) ListAllDomainWatches() (happydns.Iterator[happydns.DomainWatch], error) {
	iter := s.db.Search(domwatchPrimaryPrefix)
	return NewKVIterator[happydns.DomainWatch](s.db, iter), nil
}

func (s *KVStorage) ListDomainWatchesByUser(userId happydns.Identifier) ([]*happydns.DomainWatch, error) {
	return listByIndex(s, domwatchUserPrefix+userId.String()+"|", s.GetDomainWatch)
}

func (s *KVStorage) GetDomainWatch(id happydns.Identifier) (*happydns.DomainWatch, error) {
	w := &happydns.DomainWatch{}
	err := s.db.Get(domwatchPrimaryKey(id), w)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrDomainWatchNotFound
	}
	return w, err
}

func (s *KVStorage) CreateDomainWatch(w *happydns.DomainWatch) error {
	key, id, err := s.db.FindIdentifierKey(domwatchPrimaryPrefix)
	if err != nil {
		return err
	}
	w.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, w); err != nil {
		return err
	}
	if err := batch.Put(domwatchUserKey(w.UserId, w.Id), ""); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) UpdateDomainWatch(w *happydns.DomainWatch) error {
	return s.db.Put(domwatchPrimaryKey(w.Id), w)
}

func (s *KVStorage) DeleteDomainWatch(id happydns.Identifier) error {
	w, err := s.GetDomainWatch(id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	batch.Delete(domwatchUserKey(w.UserId, id))
	batch.Delete(domwatchPrimaryKey(id))
	return batch.Commit()
}

func (s *KVStorage) RestoreDomainWatch(w *happydns.DomainWatch) error {
	if w.Id.IsEmpty() {
		return errors.New("a domain watch to restore needs its identifier")
	}

	batch := s.db.NewBatch()
	// Stored under another user, its index entry goes with it.
	if old, err := s.GetDomainWatch(w.Id); err == nil && !old.UserId.Equals(w.UserId) {
		batch.Delete(domwatchUserKey(old.UserId, w.Id))
	} else if err != nil && !errors.Is(err, happydns.ErrDomainWatchNotFound) {
		return err
	}
	if err := batch.Put(domwatchPrimaryKey(w.Id), w); err != nil {
		return err
	}
	if err := batch.Put(domwatchUserKey(w.UserId, w.Id), ""); err != nil {
		return err
	}
	return batch.Commit()
}
