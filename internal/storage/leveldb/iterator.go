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
	"fmt"
	"log"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

// LevelDBIterator is a generic implementation of Iterator for LevelDB.
type LevelDBIterator[T any] struct {
	db     *leveldb.DB
	iter   iterator.Iterator
	err    error
	item   *T
	decode func([]byte, interface{}) error
}

// NewLevelDBIterator creates a new LevelDBIterator instance for the given LevelDB iterator and decode function.
func NewLevelDBIterator[T any](db *leveldb.DB, iter iterator.Iterator) *LevelDBIterator[T] {
	return &LevelDBIterator[T]{
		db:     db,
		iter:   iter,
		decode: decodeData,
	}
}

// NewLevelDBIterator creates a new LevelDBIterator instance for the given LevelDB iterator and decode function.
func NewLevelDBIteratorCustomDecode[T any](db *leveldb.DB, iter iterator.Iterator, decodeFunc func([]byte, interface{}) error) *LevelDBIterator[T] {
	return &LevelDBIterator[T]{
		db:     db,
		iter:   iter,
		decode: decodeFunc,
	}
}

// Next moves the iterator to the next valid item.
// Skips items that fail to decode and logs the error.
func (it *LevelDBIterator[T]) Next() bool {
	for it.iter.Next() {
		var value T
		err := it.decode(it.iter.Value(), &value)
		if err != nil {
			log.Printf("LevelDBIterator: error decoding item at key %q: %s", it.iter.Key(), err)
			it.err = err
			continue
		}
		it.item = &value
		return true
	}
	return false
}

// NextWithError advances the iterator to the next item, on decode error it doesn't continue to the next item.
// Returns true if there is a next item, false otherwise.
func (it *LevelDBIterator[T]) NextWithError() bool {
	if it.iter.Next() {
		var value T
		err := it.decode(it.iter.Value(), &value)
		if err != nil {
			it.err = err
			it.item = nil
		} else {
			it.err = nil
			it.item = &value
		}
		return true
	}
	return false
}

// Item returns the current item from the iterator.
// Only valid after a successful call to Next().
func (it *LevelDBIterator[T]) Item() *T {
	return it.item
}

// DropItem deletes the key currently pointed to by the iterator.
func (it *LevelDBIterator[T]) DropItem() error {
	if it.iter == nil || !it.iter.Valid() {
		return fmt.Errorf("DropItem: iterator is not valid")
	}
	return it.db.Delete(it.iter.Key(), nil)
}

// Raw returns the raw (non-decoded) value at the current iterator position.
// Should only be called after a successful call to Next().
func (it *LevelDBIterator[T]) Raw() []byte {
	if it.iter == nil || !it.iter.Valid() {
		return []byte{}
	}
	return it.iter.Value()
}

// Err returns the first error encountered during iteration, if any.
func (it *LevelDBIterator[T]) Err() error {
	return it.err
}

// Close releases resources held by the underlying LevelDB iterator.
func (it *LevelDBIterator[T]) Close() {
	it.iter.Release()
}
