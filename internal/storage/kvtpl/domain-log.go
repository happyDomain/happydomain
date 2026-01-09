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
	"strings"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// DomainLogIterator wraps KVIterator to populate DomainId from the key
type DomainLogIterator struct {
	*KVIterator[happydns.DomainLogWithDomainId]
}

// NewDomainLogIterator creates a new DomainLogIterator
func NewDomainLogIterator(db storage.KVStorage, iter storage.Iterator) *DomainLogIterator {
	return &DomainLogIterator{
		KVIterator: NewKVIterator[happydns.DomainLogWithDomainId](db, iter),
	}
}

// Next advances the iterator and extracts the DomainId from the key
func (it *DomainLogIterator) Next() bool {
	if it.KVIterator.Next() {
		// Extract domain ID from key
		key := it.Key()
		st := strings.Split(key, "|")
		if len(st) >= 3 && it.item != nil {
			domainId, err := happydns.NewIdentifierFromString(st[1])
			if err == nil {
				it.item.DomainId = domainId
			}
		}
		return true
	}
	return false
}

func (s *KVStorage) ListAllDomainLogs() (happydns.Iterator[happydns.DomainLogWithDomainId], error) {
	iter := s.db.Search("domain.log|")
	return NewDomainLogIterator(s.db, iter), nil
}

func (s *KVStorage) ListDomainLogs(domain *happydns.Domain) (logs []*happydns.DomainLog, err error) {
	iter := s.db.Search(fmt.Sprintf("domain.log|%s|", domain.Id.String()))
	defer iter.Release()

	for iter.Next() {
		var z happydns.DomainLog

		err = s.db.DecodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		logs = append(logs, &z)
	}

	return
}

func (s *KVStorage) getDomainLog(id string) (l *happydns.DomainLog, d *happydns.Domain, err error) {
	l = &happydns.DomainLog{}
	err = s.db.Get(id, l)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, nil, happydns.ErrDomainLogNotFound
	}

	st := strings.Split(id, "|")
	if len(st) < 3 {
		return
	}

	d = &happydns.Domain{}
	err = s.db.Get(fmt.Sprintf("domain-%s", st[1]), d)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, nil, happydns.ErrDomainNotFound
	}

	return
}

func (s *KVStorage) CreateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	key, id, err := s.db.FindIdentifierKey(fmt.Sprintf("domain.log|%s|", d.Id.String()))
	if err != nil {
		return err
	}

	l.Id = id
	return s.db.Put(key, l)
}

func (s *KVStorage) UpdateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.db.Put(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()), l)
}

func (s *KVStorage) DeleteDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.db.Delete(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()))
}
