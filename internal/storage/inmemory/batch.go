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

package inmemory

import (
	"encoding/json"

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
	data json.RawMessage // only set for batchPut
}

// Batch buffers operations and applies them under a single s.mu acquisition.
// Since the map is only mutated under that mutex, observers see all ops or
// none, which is the atomic contract.
type Batch struct {
	s   *InMemoryStorage
	ops []batchOp
}

func (s *InMemoryStorage) NewBatch() storage.Batch {
	return &Batch{s: s}
}

func (b *Batch) Put(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.ops = append(b.ops, batchOp{kind: batchPut, key: key, data: data})
	return nil
}

func (b *Batch) Delete(key string) {
	b.ops = append(b.ops, batchOp{kind: batchDelete, key: key})
}

func (b *Batch) Commit() error {
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	for _, op := range b.ops {
		switch op.kind {
		case batchPut:
			b.s.data[op.key] = op.data
		case batchDelete:
			delete(b.s.data, op.key)
		}
	}
	return nil
}
