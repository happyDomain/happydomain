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

// KVIterator implements the storage.Iterator interface for in-memory KVStorage.
type KVIterator struct {
	keys    []string
	data    map[string][]byte
	index   int
	current string
}

// NewKVIterator creates a new iterator for the given keys and data.
func NewKVIterator(keys []string, data map[string][]byte) *KVIterator {
	return &KVIterator{
		keys:  keys,
		data:  data,
		index: -1,
	}
}

// Next moves the iterator to the next item.
func (it *KVIterator) Next() bool {
	it.index++
	if it.index >= len(it.keys) {
		return false
	}
	it.current = it.keys[it.index]
	return true
}

// Key returns the current key.
func (it *KVIterator) Key() string {
	if it.index < 0 || it.index >= len(it.keys) {
		return ""
	}
	return it.current
}

// Value returns the current value.
func (it *KVIterator) Value() interface{} {
	if it.index < 0 || it.index >= len(it.keys) {
		return []byte{}
	}
	return it.data[it.current]
}

// Valid returns whether the iterator is at a valid position.
func (it *KVIterator) Valid() bool {
	return it.index >= 0 && it.index < len(it.keys)
}

// Release releases the iterator resources.
func (it *KVIterator) Release() {
	// No resources to release for in-memory iterator
}
