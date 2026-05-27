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
	"git.happydns.org/happyDomain/internal/storage"
)

type batchOpKind uint8

const (
	batchPut batchOpKind = iota
	batchDelete
)

type batchOp struct {
	kind batchOpKind
	key  string
	val  any // for batchPut; marshal happens at Commit so we reuse NoSQLStorage.Put
}

// Batch is the Oracle NoSQL implementation of storage.Batch. Oracle NoSQL
// only supports atomic multi-row writes when every row shares a shard key;
// our schema stores each entry in its own row keyed by `key`, so cross-key
// atomicity is not available on this backend. Commit therefore replays ops
// sequentially through the existing Put/Delete path and stops on the first
// error. Partial progress is possible — callers must handle it.
type Batch struct {
	s   *NoSQLStorage
	ops []batchOp
}

func (s *NoSQLStorage) NewBatch() storage.Batch {
	return &Batch{s: s}
}

func (b *Batch) Put(key string, v any) error {
	b.ops = append(b.ops, batchOp{kind: batchPut, key: key, val: v})
	return nil
}

func (b *Batch) Delete(key string) {
	b.ops = append(b.ops, batchOp{kind: batchDelete, key: key})
}

func (b *Batch) Commit() error {
	for _, op := range b.ops {
		var err error
		switch op.kind {
		case batchPut:
			err = b.s.Put(op.key, op.val)
		case batchDelete:
			err = b.s.Delete(op.key)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
