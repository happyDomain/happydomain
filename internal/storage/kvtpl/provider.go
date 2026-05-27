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
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

const (
	providerPrimaryPrefix = "provider-"
	providerOwnerPrefix   = "provider.owner|"
)

func providerOwnerKey(ownerId, providerId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", providerOwnerPrefix, ownerId.String(), providerId.String())
}

func (s *KVStorage) ListAllProviders() (happydns.Iterator[happydns.ProviderMessage], error) {
	iter := s.db.Search(providerPrimaryPrefix)
	return NewKVIterator[happydns.ProviderMessage](s.db, iter), nil
}

func (s *KVStorage) CountProviders() (int, error) {
	return s.countByPrefix(providerPrimaryPrefix)
}

func (s *KVStorage) getProviderMeta(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	srcMsg := &happydns.ProviderMessage{}
	err := s.db.Get(fmt.Sprintf("%s%s", providerPrimaryPrefix, id.String()), srcMsg)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrProviderNotFound
	}
	return srcMsg, err
}

func (s *KVStorage) ListProviders(u *happydns.User) (srcs happydns.ProviderMessages, err error) {
	prefix := fmt.Sprintf("%s%s|", providerOwnerPrefix, u.Id.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	for iter.Next() {
		id, idErr := lastKeySegment(iter.Key())
		if idErr != nil {
			log.Printf("storage: malformed provider owner index key %q: %v", iter.Key(), idErr)
			continue
		}
		msg, getErr := s.getProviderMeta(id)
		if getErr != nil {
			// Index drift: skip rather than fail the whole list.
			log.Printf("storage: provider owner index points to missing provider %q: %v", id.String(), getErr)
			continue
		}
		srcs = append(srcs, msg)
	}

	if err = iter.Err(); err != nil {
		return
	}

	return
}

func (s *KVStorage) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	var prvdMsg happydns.ProviderMessage
	err := s.db.Get(fmt.Sprintf("%s%s", providerPrimaryPrefix, id.String()), &prvdMsg)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrProviderNotFound
	}
	if err != nil {
		return nil, err
	}

	return &prvdMsg, nil
}

func (s *KVStorage) CreateProvider(prvd *happydns.Provider) error {
	key, id, err := s.db.FindIdentifierKey(providerPrimaryPrefix)
	if err != nil {
		return err
	}

	prvd.Id = id

	if err := s.db.Put(key, prvd); err != nil {
		return err
	}

	if err := s.db.Put(providerOwnerKey(prvd.Owner, prvd.Id), true); err != nil {
		// Roll back primary so a failed index write doesn't orphan it.
		if delErr := s.db.Delete(key); delErr != nil {
			log.Printf("storage: orphan provider %q after index write failed (rollback also failed: %v)", prvd.Id.String(), delErr)
		}
		return err
	}
	return nil
}

func (s *KVStorage) UpdateProvider(prvd *happydns.Provider) error {
	// Load the existing record so we can detect an owner change and clean up
	// the stale index entry.
	old, err := s.GetProvider(prvd.Id)
	if err != nil {
		return err
	}

	if err := s.db.Put(fmt.Sprintf("%s%s", providerPrimaryPrefix, prvd.Id.String()), prvd); err != nil {
		return err
	}

	if !old.Owner.Equals(prvd.Owner) {
		if err := s.db.Delete(providerOwnerKey(old.Owner, prvd.Id)); err != nil {
			log.Printf("UpdateProvider: failed to delete stale owner index for owner %s: %v", old.Owner.String(), err)
		}
	}

	return s.db.Put(providerOwnerKey(prvd.Owner, prvd.Id), true)
}

func (s *KVStorage) DeleteProvider(prvdId happydns.Identifier) error {
	// Load first so we know which owner index to clean up.
	prvd, err := s.GetProvider(prvdId)
	if err != nil {
		return err
	}

	if err := s.db.Delete(providerOwnerKey(prvd.Owner, prvdId)); err != nil {
		log.Printf("DeleteProvider: failed to delete owner index for owner %s: %v", prvd.Owner.String(), err)
	}

	return s.db.Delete(fmt.Sprintf("%s%s", providerPrimaryPrefix, prvdId.String()))
}

func (s *KVStorage) ClearProviders() error {
	if err := s.clearByPrefix(providerOwnerPrefix); err != nil {
		return err
	}

	iter, err := s.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		err = s.db.Delete(iter.Key())
		if err != nil {
			return err
		}
	}

	return iter.Err()
}
