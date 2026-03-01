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
	"database/sql"
	"log"
)

// PostgreSQLIterator implements the storage.Iterator interface for PostgreSQL
type PostgreSQLIterator struct {
	rows  *sql.Rows
	key   string
	value []byte
	valid bool
	err   error
}

// Release closes the underlying sql.Rows and releases resources
func (it *PostgreSQLIterator) Release() {
	if it.rows != nil {
		it.rows.Close()
		it.rows = nil
	}
	it.valid = false
}

// Next advances the iterator to the next row
func (it *PostgreSQLIterator) Next() bool {
	// If there was a previous error or rows is nil, return false
	if it.err != nil || it.rows == nil {
		it.valid = false
		return false
	}

	// Advance to next row
	if !it.rows.Next() {
		it.valid = false
		// Check for any error that occurred during iteration
		if err := it.rows.Err(); err != nil {
			it.err = err
			log.Printf("PostgreSQL iterator error: %v", err)
		}
		return false
	}

	// Scan the current row
	if err := it.rows.Scan(&it.key, &it.value); err != nil {
		it.err = err
		it.valid = false
		log.Printf("PostgreSQL iterator scan error: %v", err)
		return false
	}

	it.valid = true
	return true
}

// Valid returns whether the iterator is at a valid position
func (it *PostgreSQLIterator) Valid() bool {
	return it.valid && it.err == nil
}

// Key returns the current key
func (it *PostgreSQLIterator) Key() string {
	return it.key
}

// Value returns the current value as []byte
func (it *PostgreSQLIterator) Value() any {
	return it.value
}
