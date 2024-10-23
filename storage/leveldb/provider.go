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
	"fmt"
	"log"
	"reflect"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

func (s *LevelDBStorage) getProviderMeta(id happydns.Identifier) (srcMsg *happydns.ProviderMessage, err error) {
	var v []byte
	v, err = s.db.Get(id, nil)
	if err != nil {
		return
	}

	srcMsg = &happydns.ProviderMessage{}
	err = decodeData(v, srcMsg)
	return
}

func (s *LevelDBStorage) GetProviders(u *happydns.User) (srcs happydns.ProviderMessages, err error) {
	iter := s.search("provider-")
	defer iter.Release()

	for iter.Next() {
		var srcMsg happydns.ProviderMessage
		err = decodeData(iter.Value(), &srcMsg)
		if err != nil {
			return
		}

		if !bytes.Equal(srcMsg.OwnerId, u.Id) {
			continue
		}

		srcs = append(srcs, &srcMsg)
	}

	return
}

func (s *LevelDBStorage) GetProvider(u *happydns.User, id happydns.Identifier) (src *happydns.ProviderMessage, err error) {
	var v []byte
	v, err = s.db.Get([]byte(fmt.Sprintf("provider-%s", id.String())), nil)
	if err != nil {
		return
	}

	var srcMsg happydns.ProviderMessage
	err = decodeData(v, &srcMsg)
	if err != nil {
		return
	}

	if !bytes.Equal(srcMsg.OwnerId, u.Id) {
		src = nil
		err = leveldb.ErrNotFound
		return
	}

	return &srcMsg, err
}

func (s *LevelDBStorage) CreateProvider(u *happydns.User, src providers.Provider, comment string) (*happydns.ProviderCombined, error) {
	key, id, err := s.findIdentifierKey("provider-")
	if err != nil {
		return nil, err
	}

	sType := reflect.Indirect(reflect.ValueOf(src)).Type()

	st := &happydns.ProviderCombined{
		Provider: src,
		ProviderMeta: happydns.ProviderMeta{
			Type:    sType.Name(),
			Id:      id,
			OwnerId: u.Id,
			Comment: comment,
		},
	}
	return st, s.put(key, st)
}

func (s *LevelDBStorage) UpdateProvider(src *happydns.ProviderCombined) error {
	return s.put(fmt.Sprintf("provider-%s", src.Id.String()), src)
}

func (s *LevelDBStorage) UpdateProviderOwner(src *happydns.ProviderCombined, newOwner *happydns.User) error {
	src.OwnerId = newOwner.Id
	return s.UpdateProvider(src)
}

func (s *LevelDBStorage) DeleteProvider(src *happydns.ProviderMeta) error {
	return s.delete(fmt.Sprintf("provider-%s", src.Id.String()))
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

func (s *LevelDBStorage) TidyProviders() error {
	tx, err := s.db.OpenTransaction()
	if err != nil {
		return err
	}

	iter := tx.NewIterator(util.BytesPrefix([]byte("provider-")), nil)
	defer iter.Release()

	for iter.Next() {
		srcMeta, err := s.getProviderMeta(iter.Key())

		if err != nil {
			// Drop unreadable providers
			log.Printf("Deleting unreadable provider (%s): %v\n", err.Error(), srcMeta)
			err = tx.Delete(iter.Key(), nil)
		} else {
			_, err = s.GetUser(srcMeta.OwnerId)
			if err == leveldb.ErrNotFound {
				// Drop providers of unexistant users
				log.Printf("Deleting orphan provider (user %s not found): %v\n", srcMeta.OwnerId.String(), srcMeta)
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
