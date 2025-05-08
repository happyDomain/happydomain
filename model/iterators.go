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

package happydns

// Iterator defines a generic interface for iterating over a sequence of items of type T.
type Iterator[T any] interface {
	// Next advances the iterator to the next item.
	// Returns true if there is a next item, false otherwise.
	Next() bool

	// NextWithError advances the iterator to the next item, on decode error it doesn't continue to the next item.
	// Returns true if there is a next item, false otherwise.
	NextWithError() bool

	// Item returns the current item in the iteration.
	// Should be called only after a successful call to Next().
	Item() *T

	// DropItem deletes the current item pointed to by the iterator.
	// Must be called only after a successful call to Next().
	DropItem() error

	// Raw returns the raw (non-decoded) value at the current iterator position.
	// Should only be called after a successful call to Next().
	Raw() []byte

	// Err returns the first error encountered during iteration, if any.
	Err() error

	// Close releases any resources associated with the iterator.
	Close()
}
