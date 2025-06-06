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
	"bytes"
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (s *LevelDBStorage) ListAllProviders() (happydns.Iterator[happydns.ProviderMessage], error) {
	iter := s.search("provider-")
	return NewLevelDBIterator[happydns.ProviderMessage](s.db, iter), nil
}

func (s *LevelDBStorage) getProviderMeta(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	v, err := s.db.Get(id, nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, happydns.ErrProviderNotFound
		} else {
			return nil, err
		}
	}

	srcMsg := &happydns.ProviderMessage{}
	err = decodeData(v, srcMsg)
	return srcMsg, err
}

func (s *LevelDBStorage) ListProviders(u *happydns.User) (srcs happydns.ProviderMessages, err error) {
	iter := s.search("provider-")
	defer iter.Release()

	for iter.Next() {
		var srcMsg happydns.ProviderMessage
		err = decodeData(iter.Value(), &srcMsg)
		if err != nil {
			return
		}

		if !bytes.Equal(srcMsg.Owner, u.Id) {
			continue
		}

		srcs = append(srcs, &srcMsg)
	}

	return
}

func (s *LevelDBStorage) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	v, err := s.db.Get([]byte(fmt.Sprintf("provider-%s", id.String())), nil)
	if err != nil {
		if errors.Is(err, leveldb.ErrNotFound) {
			return nil, happydns.ErrProviderNotFound
		} else {
			return nil, err
		}
	}

	var prvdMsg happydns.ProviderMessage
	err = decodeData(v, &prvdMsg)
	if err != nil {
		return nil, err
	}

	return &prvdMsg, err
}

func (s *LevelDBStorage) CreateProvider(prvd *happydns.Provider) error {
	key, id, err := s.findIdentifierKey("provider-")
	if err != nil {
		return err
	}

	prvd.Id = id

	return s.put(key, prvd)
}

func (s *LevelDBStorage) UpdateProvider(prvd *happydns.Provider) error {
	return s.put(fmt.Sprintf("provider-%s", prvd.Id.String()), prvd)
}

func (s *LevelDBStorage) DeleteProvider(prvdId happydns.Identifier) error {
	return s.delete(fmt.Sprintf("provider-%s", prvdId.String()))
}

func (s *LevelDBStorage) ClearProviders() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("provider-")), nil)
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
