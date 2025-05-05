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

package inmemory

import (
	"encoding/json"
	"iter"
	"maps"
)

// InMemoryIterator provides an iterator over a map[string]*T.
type InMemoryIterator[T any] struct {
	origin *map[string]*T
	next   func() (string, *T, bool)
	stop   func()
	key    string
	item   *T
	err    error
	decode func([]byte, interface{}) error
}

// NewInMemoryIterator constructs an iterator over a map[string]*T.
// You can pass any map like map[string]*UserAuth, map[string]*Domain, etc.
func NewInMemoryIterator[T any](m *map[string]*T) *InMemoryIterator[T] {
	next, stop := iter.Pull2(maps.All[map[string]*T](*m))
	return &InMemoryIterator[T]{
		origin: m,
		next:   next,
		stop:   stop,
	}
}

// NewInMemoryIteratorCustomDecode creates a new LevelDBIterator instance for the given LevelDB iterator and decode function.
func NewInMemoryIteratorCustomDecode[T any](m *map[string]*T, decodeFunc func([]byte, interface{}) error) *InMemoryIterator[T] {
	next, stop := iter.Pull2(maps.All[map[string]*T](*m))
	return &InMemoryIterator[T]{
		origin: m,
		next:   next,
		stop:   stop,
		decode: decodeFunc,
	}
}

// Next advances the iterator to the next item.
func (it *InMemoryIterator[T]) Next() (valid bool) {
	it.key, it.item, valid = it.next()
	return
}

// NextWithError advances the iterator to the next item.
func (it *InMemoryIterator[T]) NextWithError() (valid bool) {
	it.key, it.item, valid = it.next()
	return
}

// Item returns the current item pointed to by the iterator.
func (it *InMemoryIterator[T]) Item() *T {
	return it.item
}

// DropItem deletes the key currently pointed to by the iterator.
func (it *InMemoryIterator[T]) DropItem() error {
	delete(*it.origin, it.key)
	return nil
}

// Raw returns the raw (non-decoded) value at the current iterator position.
// Should only be called after a successful call to Next().
func (it *InMemoryIterator[T]) Raw() []byte {
	j, _ := json.Marshal(it.item)
	return j
}

// Err returns any error encountered during iteration (always nil here).
func (it *InMemoryIterator[T]) Err() error {
	return it.err
}

// Close is a no-op for in-memory iterators, present to satisfy common interfaces.
func (it *InMemoryIterator[T]) Close() {
	return
}
