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

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"

	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) ListAllDomainLogs() (storage.Iterator[happydns.DomainLogWithDomainId], error) {
	iter := s.search("domain.log|")
	return NewLevelDBIteratorCustomDecode[happydns.DomainLogWithDomainId](s.db, iter, func(data []byte, v interface{}) error {
		err := decodeData(data, v)
		if err != nil {
			return err
		}

		st := strings.Split(string(iter.Key()), "|")
		if len(st) < 3 {
			return fmt.Errorf("invalid domain log key: %s", string(iter.Key()))
		}

		v.(*happydns.DomainLogWithDomainId).DomainId, err = happydns.NewIdentifierFromString(st[1])
		return err
	}), nil
}

func (s *LevelDBStorage) GetDomainLogs(domain *happydns.Domain) (logs []*happydns.DomainLog, err error) {
	iter := s.search(fmt.Sprintf("domain.log|%s|", domain.Id.String()))
	defer iter.Release()

	for iter.Next() {
		var z happydns.DomainLog

		err = decodeData(iter.Value(), &z)
		if err != nil {
			return
		}

		logs = append(logs, &z)
	}

	return
}

func (s *LevelDBStorage) getDomainLog(id string) (l *happydns.DomainLog, d *happydns.Domain, err error) {
	l = &happydns.DomainLog{}
	err = s.get(id, l)

	st := strings.Split(id, "|")
	if len(st) < 3 {
		return
	}

	d = &happydns.Domain{}
	err = s.get(id, fmt.Sprintf("domain-%s", st[1]))

	return
}

func (s *LevelDBStorage) CreateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	key, id, err := s.findIdentifierKey(fmt.Sprintf("domain.log|%s|", d.Id.String()))
	if err != nil {
		return err
	}

	l.Id = id
	return s.put(key, l)
}

func (s *LevelDBStorage) UpdateDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.put(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()), l)
}

func (s *LevelDBStorage) DeleteDomainLog(d *happydns.Domain, l *happydns.DomainLog) error {
	return s.delete(fmt.Sprintf("domain.log|%s|%s", d.Id.String(), l.Id.String()))
}

func (s *LevelDBStorage) TidyDomainLogs() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("domain.log|")), nil)
	defer iter.Release()

	for iter.Next() {
		l, _, err := s.getDomainLog(string(iter.Key()))

		if err != nil {
			if l != nil {
				log.Printf("Deleting log without valid domain: %s (%s)\n", l.Id.String(), err.Error())
			} else {
				log.Printf("Deleting unreadable log (%s): %v\n", err.Error(), l)
			}
			err = tx.Delete(iter.Key(), nil)
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
