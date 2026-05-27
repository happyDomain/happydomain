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
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"

	"git.happydns.org/happyDomain/internal/storage"
)

// Batch is the LevelDB-backed implementation of storage.Batch. It defers
// every staged op to leveldb.Batch so Commit performs a single WAL append
// and fsync; this gives true cross-key atomicity in addition to being
// faster than the per-key Put/Delete path.
type Batch struct {
	db    *leveldb.DB
	batch *leveldb.Batch
}

func (s *LevelDBStorage) NewBatch() storage.Batch {
	return &Batch{db: s.db, batch: new(leveldb.Batch)}
}

func (b *Batch) Put(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.batch.Put([]byte(key), data)
	return nil
}

func (b *Batch) Delete(key string) {
	b.batch.Delete([]byte(key))
}

func (b *Batch) Commit() error {
	return b.db.Write(b.batch, nil)
}
