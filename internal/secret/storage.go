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

import (
	"git.happydns.org/happyDomain/model"
)

// SafeStorage keeps the safes.
type SafeStorage interface {
	// ListAllSafes lists every safe.
	ListAllSafes() (happydns.Iterator[happydns.Safe], error)

	// GetSafe returns the safe with the given identifier, or
	// happydns.ErrSafeNotFound.
	GetSafe(id happydns.Identifier) (*happydns.Safe, error)

	// GetSafeByOwner returns the safe of the given kind owned by owner, or
	// happydns.ErrSafeNotFound.
	GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error)

	// CreateSafe stores a new safe, failing when its owner already has one
	// of its kind.
	CreateSafe(safe *happydns.Safe) error

	// UpdateSafe replaces a stored safe.
	UpdateSafe(safe *happydns.Safe) error

	// DeleteSafe removes a safe: every value sealed in it becomes
	// unreadable.
	DeleteSafe(id happydns.Identifier) error
}

// CheckStorage keeps the check record of the instance keyset.
type CheckStorage interface {
	// GetSecretCheck returns the check record, or happydns.ErrNotFound.
	GetSecretCheck() ([]byte, error)

	// PutSecretCheck stores the check record.
	PutSecretCheck(record []byte) error

	// DeleteSecretCheck removes the check record, once no safe is left.
	DeleteSecretCheck() error
}

// Storage is what secret management keeps in the database.
type Storage interface {
	CheckStorage
	SafeStorage
}

// eachSafe calls fn on every safe of store, and damaged, when not nil, on
// every record that does not decode, with its key. The listing is closed once
// it returns: fn may collect what to change, but not change the storage while
// it lists, which is not safe on every storage.
func eachSafe(store SafeStorage, fn func(*happydns.Safe), damaged func(key string, err error)) error {
	iter, err := store.ListAllSafes()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			if damaged != nil {
				damaged(iter.Key(), err)
			}
			continue
		}
		fn(iter.Item())
	}
	return iter.Err()
}
