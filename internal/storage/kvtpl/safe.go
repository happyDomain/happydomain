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
	"slices"

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

// safeOwnerKey indexes the safe of a kind a user owns; its value is the
// identifier of the safe.
func safeOwnerKey(owner happydns.Identifier, kind string) string {
	return fmt.Sprintf("%s%s|%s", safeOwnerPrefix, owner.String(), kind)
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

func (s *KVStorage) GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error) {
	var id happydns.Identifier
	err := s.db.Get(safeOwnerKey(owner, kind), &id)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrSafeNotFound
	}
	if err != nil {
		return nil, err
	}

	safe, err := s.GetSafe(id)
	if err != nil {
		return nil, err
	}
	if !safe.Owner.Equals(owner) || safe.Kind != kind {
		return nil, fmt.Errorf("safe index %s points to a safe of another owner or kind", safeOwnerKey(owner, kind))
	}
	return safe, nil
}

// CreateSafe stores a new safe under the identifier it carries. It fails with
// happydns.ErrInvalidIdentifier when that identifier is missing or malformed,
// and with happydns.ErrAlreadyExists when it is taken or when the owner
// already has a safe of this kind.
//
// Concurrent creations for the same owner and kind, from this process or
// another sharing the database, leave a single safe: the safe is written
// first, then claims the owner index, and a safe losing that claim is
// removed.
//
// An owner index pointing to a safe that no longer exists is replaced: left
// in place, it would keep its owner from ever getting a safe of that kind.
func (s *KVStorage) CreateSafe(safe *happydns.Safe) error {
	if safe.Owner.IsEmpty() || safe.Kind == "" {
		return errors.New("a safe needs an owner and a kind")
	}

	return s.createNew("safe", safe.Id, safePrimaryKey(safe.Id), safe, func() error {
		return s.claimSafeOwnerIndex(safe)
	})
}

// RestoreSafe stores a safe coming from elsewhere, such as a backup, beside
// the one its owner may already have: that one keeps the owner index, so that
// new secrets keep going to it.
func (s *KVStorage) RestoreSafe(safe *happydns.Safe) error {
	if safe.Owner.IsEmpty() || safe.Kind == "" {
		return errors.New("a safe needs an owner and a kind")
	}

	return s.createNew("safe", safe.Id, safePrimaryKey(safe.Id), safe, func() error {
		err := s.claimSafeOwnerIndex(safe)
		if errors.Is(err, happydns.ErrAlreadyExists) {
			return nil
		}
		return err
	})
}

// claimSafeOwnerIndex points the owner index of safe to it, unless it already
// points to another safe that exists.
func (s *KVStorage) claimSafeOwnerIndex(safe *happydns.Safe) error {
	index := safeOwnerKey(safe.Owner, safe.Kind)
	if claimed, err := s.db.PutIfAbsent(index, safe.Id); err != nil || claimed {
		return err
	}

	_, err := s.GetSafeByOwner(safe.Owner, safe.Kind)
	if err == nil {
		return fmt.Errorf("safe %s: %w", index, happydns.ErrAlreadyExists)
	}
	if !errors.Is(err, happydns.ErrSafeNotFound) {
		return err
	}

	// Dangling: a safe is written before its index, so this one does not
	// belong to a creation in progress. Two creations replacing the same
	// dangling index at once may still leave an unindexed safe behind.
	return s.db.Put(index, safe.Id)
}

// UpdateSafe replaces a stored safe. Its owner and kind cannot change.
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

// ReplaceSafe rewrites a stored safe, unless it changed or was deleted in
// between. Its owner and kind cannot change.
func (s *KVStorage) ReplaceSafe(id happydns.Identifier, update func(*happydns.Safe) (*happydns.Safe, error)) error {
	return replace(s.db, safePrimaryKey(id), happydns.ErrSafeNotFound, func(old *happydns.Safe) (*happydns.Safe, error) {
		owner, kind := slices.Clone(old.Owner), old.Kind
		next, err := update(old)
		if err != nil || next == nil {
			return next, err
		}
		if !next.Id.Equals(id) || !next.Owner.Equals(owner) || next.Kind != kind {
			return nil, errors.New("the identifier, owner and kind of a safe cannot change")
		}
		return next, nil
	})
}

func (s *KVStorage) DeleteSafe(id happydns.Identifier) error {
	safe, err := s.GetSafe(id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	// An owner can hold more than one safe of a kind, restored beside each
	// other: the index goes only with the safe it points to.
	var indexed happydns.Identifier
	if err := s.db.Get(safeOwnerKey(safe.Owner, safe.Kind), &indexed); err == nil && indexed.Equals(id) {
		batch.Delete(safeOwnerKey(safe.Owner, safe.Kind))
	} else if err != nil && !errors.Is(err, happydns.ErrNotFound) {
		return err
	}
	batch.Delete(safePrimaryKey(id))
	return batch.Commit()
}
