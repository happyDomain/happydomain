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
	"sort"
	"strings"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

type KVStorage struct {
	db storage.KVStorage
}

func NewKVDatabase(impl storage.KVStorage) (storage.Storage, error) {
	return &KVStorage{
		impl,
	}, nil
}

func (s *KVStorage) Close() error {
	return s.db.Close()
}

// lastKeySegment extracts the identifier after the last "|" in a KV key.
func lastKeySegment(key string) (happydns.Identifier, error) {
	i := strings.LastIndex(key, "|")
	if i < 0 {
		return happydns.Identifier{}, fmt.Errorf("key %q has no pipe separator", key)
	}
	return happydns.NewIdentifierFromString(key[i+1:])
}

// listByIndex scans a secondary index prefix, resolves each entity by its
// last key segment, and returns the collected results.
func listByIndex[T any](s *KVStorage, prefix string, getEntity func(happydns.Identifier) (*T, error)) ([]*T, error) {
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*T
	for iter.Next() {
		id, err := lastKeySegment(iter.Key())
		if err != nil {
			continue
		}
		entity, err := getEntity(id)
		if err != nil {
			continue
		}
		results = append(results, entity)
	}
	return results, nil
}

// listByIndexSorted is like listByIndex but sorts results and applies a limit.
func listByIndexSorted[T any](s *KVStorage, prefix string, getEntity func(happydns.Identifier) (*T, error), less func(*T, *T) bool, limit int) ([]*T, error) {
	results, err := listByIndex(s, prefix, getEntity)
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return less(results[i], results[j])
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// tidyTwoPartIndex removes stale secondary index entries of the form
// prefix{ownerId}|{entityId}. If validateOwner is non-nil, entries whose
// owner ID fails validation are also removed.
func (s *KVStorage) tidyTwoPartIndex(prefix, label string, validateOwner func(happydns.Identifier) bool, entityExists func(happydns.Identifier) bool) {
	iter := s.db.Search(prefix)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		rest := strings.TrimPrefix(key, prefix)
		parts := strings.SplitN(rest, "|", 2)
		if len(parts) != 2 {
			_ = s.db.Delete(key)
			continue
		}

		ownerId, err := happydns.NewIdentifierFromString(parts[0])
		if err != nil {
			_ = s.db.Delete(key)
			continue
		}

		entityId, err := happydns.NewIdentifierFromString(parts[1])
		if err != nil {
			_ = s.db.Delete(key)
			continue
		}

		if validateOwner != nil && !validateOwner(ownerId) {
			log.Printf("Deleting stale %s index (%s %s not found): %s\n", label, label, parts[0], key)
			_ = s.db.Delete(key)
			continue
		}

		if !entityExists(entityId) {
			log.Printf("Deleting stale %s index (entity %s not found): %s\n", label, parts[1], key)
			_ = s.db.Delete(key)
		}
	}
}

// tidyLastSegmentIndex removes stale index entries where the entity ID is the
// last "|"-separated segment. Used for multi-part indexes like
// prefix{checkerID}|{target}|{entityId}.
func (s *KVStorage) tidyLastSegmentIndex(prefix, label string, entityExists func(happydns.Identifier) bool) {
	iter := s.db.Search(prefix)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		lastPipe := strings.LastIndex(key, "|")
		if lastPipe < 0 {
			_ = s.db.Delete(key)
			continue
		}
		idStr := key[lastPipe+1:]

		id, err := happydns.NewIdentifierFromString(idStr)
		if err != nil {
			_ = s.db.Delete(key)
			continue
		}

		if !entityExists(id) {
			log.Printf("Deleting stale %s index (entity %s not found): %s\n", label, idStr, key)
			_ = s.db.Delete(key)
		}
	}
}

// clearByPrefix deletes all KV entries matching the given prefix.
func (s *KVStorage) clearByPrefix(prefix string) error {
	iter := s.db.Search(prefix)
	defer iter.Release()
	for iter.Next() {
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}
	return nil
}
