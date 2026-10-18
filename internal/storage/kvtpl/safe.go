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
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

const (
	safePrimaryPrefix = "safe-"
	// Not "safe-owner|": it would be listed with the safes themselves.
	safeOwnerPrefix = "safe.owner|"
)

func safePrimaryKey(id happydns.Identifier) string {
	return safePrimaryPrefix + id.String()
}

// safeOwnerKey indexes a safe under its owner, who may have several of a
// kind; its value is the identifier of the safe.
func safeOwnerKey(owner happydns.Identifier, id happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", safeOwnerPrefix, owner.String(), id.String())
}

func (s *KVStorage) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	iter := s.db.Search(safePrimaryPrefix)
	return NewKVIterator[happydns.Safe](s.db, iter), nil
}

func (s *KVStorage) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	var safe happydns.Safe
	err := s.db.Get(safePrimaryKey(id), &safe)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrSafeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &safe, nil
}

// ListOwnerSafes returns every safe owner has, of every kind.
func (s *KVStorage) ListOwnerSafes(owner happydns.Identifier) ([]*happydns.Safe, error) {
	iter := s.db.Search(safeOwnerPrefix + owner.String() + "|")
	defer iter.Release()

	var safes []*happydns.Safe
	for iter.Next() {
		var id happydns.Identifier
		if err := s.db.DecodeData(iter.Value(), &id); err != nil {
			log.Printf("storage: malformed safe index %q: %v", iter.Key(), err)
			continue
		}
		safe, err := s.GetSafe(id)
		if err != nil {
			// Index drift: skip rather than fail the whole list.
			log.Printf("storage: safe index %q points to a missing safe: %v", iter.Key(), err)
			continue
		}
		if !safe.Owner.Equals(owner) {
			log.Printf("storage: safe index %q points to a safe of another owner", iter.Key())
			continue
		}
		safes = append(safes, safe)
	}
	return safes, nil
}

// GetSafeByOwner returns the oldest safe of kind the instance created for
// owner, one without a name: those they created are not.
func (s *KVStorage) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	safes, err := s.ListOwnerSafes(owner)
	if err != nil {
		return nil, err
	}

	var found *happydns.Safe
	for _, safe := range safes {
		if safe.Kind == kind && safe.Name == "" && (found == nil || safe.CreatedAt.Before(found.CreatedAt)) {
			found = safe
		}
	}
	if found == nil {
		return nil, happydns.ErrSafeNotFound
	}
	return found, nil
}

// CreateSafe stores a new safe. It fails when its identifier is taken.
func (s *KVStorage) CreateSafe(safe *happydns.Safe) error {
	if safe.Id.IsEmpty() || safe.Owner.IsEmpty() || safe.Kind == "" {
		return errors.New("a safe needs an identifier, an owner and a kind")
	}

	exists, err := s.db.Has(safePrimaryKey(safe.Id))
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("safe already exists: %s", safe.Id.String())
	}

	batch := s.db.NewBatch()
	if err := batch.Put(safePrimaryKey(safe.Id), safe); err != nil {
		return err
	}
	if err := batch.Put(safeOwnerKey(safe.Owner, safe.Id), safe.Id); err != nil {
		return err
	}
	return batch.Commit()
}

// UpdateSafe replaces a stored safe. Its owner and kind cannot change: the
// owner index points to it, and its keyring is bound to its owner.
func (s *KVStorage) UpdateSafe(safe *happydns.Safe) error {
	old, err := s.GetSafe(safe.Id)
	if err != nil {
		return err
	}
	if !old.Owner.Equals(safe.Owner) || old.Kind != safe.Kind {
		return errors.New("the owner and kind of a safe cannot change")
	}

	return s.db.Put(safePrimaryKey(safe.Id), safe)
}

func (s *KVStorage) DeleteSafe(id happydns.Identifier) error {
	safe, err := s.GetSafe(id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	batch.Delete(safeOwnerKey(safe.Owner, safe.Id))
	batch.Delete(safePrimaryKey(id))
	return batch.Commit()
}
