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

package database // import "git.happydns.org/happyDomain/internal/storage/leveldb"

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"

	"git.happydns.org/happyDomain/model"
)

type LevelDBStorage struct {
	db *leveldb.DB
}

// NewMySQLStorage establishes the connection to the database
func NewLevelDBStorage(path string) (s *LevelDBStorage, err error) {
	var db *leveldb.DB

	db, err = leveldb.OpenFile(path, nil)
	if err != nil {
		if _, ok := err.(*errors.ErrCorrupted); ok {
			log.Printf("LevelDB was corrupted; attempting recovery (%s)", err.Error())
			_, err = leveldb.RecoverFile(path, nil)
			if err != nil {
				return
			}
			log.Println("LevelDB recovery succeeded!")
		} else {
			return
		}
	}

	s = &LevelDBStorage{db}
	return
}

func (s *LevelDBStorage) Close() error {
	return s.db.Close()
}

func decodeData(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (s *LevelDBStorage) get(key string, v interface{}) error {
	data, err := s.db.Get([]byte(key), nil)
	if err != nil {
		return err
	}

	return decodeData(data, v)
}

func (s *LevelDBStorage) put(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return s.db.Put([]byte(key), data, nil)
}

func (s *LevelDBStorage) findIdentifierKey(prefix string) (key string, id happydns.Identifier, err error) {
	found := true
	for found {
		id, err = happydns.NewRandomIdentifier()
		if err != nil {
			return
		}
		key = fmt.Sprintf("%s%s", prefix, id.String())

		found, err = s.db.Has([]byte(key), nil)
		if err != nil {
			return
		}
	}
	return
}

func (s *LevelDBStorage) delete(key string) error {
	return s.db.Delete([]byte(key), nil)
}

func (s *LevelDBStorage) search(prefix string) iterator.Iterator {
	return s.db.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
}
