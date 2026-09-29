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

package secret

// iterEntry is a record listed by entriesIterator: an item, or an error
// for a record that does not decode.
type iterEntry[T any] struct {
	key  string
	item *T
	err  error
}

// entriesIterator behaves as the storage iterators: Next skips what does
// not decode, NextWithError stops on it. As theirs, Err keeps reporting a
// record Next skipped.
type entriesIterator[T any] struct {
	entries []iterEntry[T]
	idx     int
	err     error
	skipped error
	listErr error
	closed  bool
}

func newEntriesIterator[T any](entries ...iterEntry[T]) *entriesIterator[T] {
	return &entriesIterator[T]{entries: entries, idx: -1}
}

func (it *entriesIterator[T]) Next() bool {
	for it.NextWithError() {
		if it.err == nil {
			return true
		}
		it.skipped = it.err
	}
	return false
}

func (it *entriesIterator[T]) NextWithError() bool {
	it.idx++
	if it.idx >= len(it.entries) {
		it.err = nil
		return false
	}
	it.err = it.entries[it.idx].err
	return true
}

func (it *entriesIterator[T]) Item() *T {
	if it.err != nil {
		return nil
	}
	return it.entries[it.idx].item
}
func (it *entriesIterator[T]) DropItem() error { return nil }
func (it *entriesIterator[T]) Key() string     { return it.entries[it.idx].key }
func (it *entriesIterator[T]) Raw() any        { return nil }
func (it *entriesIterator[T]) Close()          { it.closed = true }
func (it *entriesIterator[T]) Err() error {
	if it.err != nil {
		return it.err
	}
	if it.skipped != nil {
		return it.skipped
	}
	return it.listErr
}
