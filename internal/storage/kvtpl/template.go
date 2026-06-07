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
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

// obsRefLockShards bounds the table of per-primary-key mutexes used by
// PutDiscoveryObservationRef. A primary key hashes to a fixed shard so two
// writers targeting the same primary always pick the same lock, while
// writers targeting different primaries usually pick different ones and
// proceed in parallel. 64 is plenty for the volume of concurrent producers
// happyDomain runs in practice and keeps the per-storage memory cost flat.
const obsRefLockShards = 64

type KVStorage struct {
	db storage.KVStorage

	// obsRefMu protects the Get-then-batch-commit sequence inside
	// PutDiscoveryObservationRef against concurrent writes at the same
	// primary key. See the godoc on that method for the race it closes.
	obsRefMu [obsRefLockShards]sync.Mutex
}

func NewKVDatabase(impl storage.KVStorage) (storage.Storage, error) {
	return &KVStorage{
		db: impl,
	}, nil
}

// lockForObsRef returns the shard mutex guarding the given primary key.
// The shard is picked by FNV-1a hash of the key so the same primary key
// always maps to the same mutex.
func (s *KVStorage) lockForObsRef(primaryKey string) *sync.Mutex {
	h := fnv.New32a()
	_, _ = h.Write([]byte(primaryKey))
	return &s.obsRefMu[h.Sum32()%obsRefLockShards]
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

// lastTwoSegments returns the final two "|"-separated fields of key, joined
// with their separator (e.g. "{revChrono}|{entityId}"). It returns ok=false
// when key has fewer than two "|" separators. Recency indexes that share a
// "{revChrono}|{entityId}" tail use this to rebuild sibling index keys from a
// related index key without scanning.
func lastTwoSegments(key string) (tail string, ok bool) {
	i := strings.LastIndex(key, "|")
	if i < 0 {
		return "", false
	}
	j := strings.LastIndex(key[:i], "|")
	if j < 0 {
		return "", false
	}
	return key[j+1:], true
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

// maxKeySize is the hard backend limit some KV stores enforce on key lengths.
// Every key builder in this package is sized to stay within it; key_size_test.go
// asserts the bound for each.
const maxKeySize = 64

// hash28 returns the 28-char base64url encoding of the first 21 bytes of the
// SHA-256 of s, for use in key construction.
func hash28(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:21])
}

// sortableBase32 is a 32-character alphabet whose ASCII order matches its
// base32 value order (0=0, …, 9=9, A=10, …, V=31), making strings encoded
// with it sort lexicographically the same way as the underlying numbers.
const sortableBase32 = "0123456789ABCDEFGHIJKLMNOPQRSTUV"

// reverseChronoSegment encodes t as a fixed 13-char string whose ascending
// lexical order matches reverse chronological (newest first) order. A
// sort-preserving base32 alphabet is used so that lexicographic and numerical
// ordering agree, letting a forward KV prefix scan return the most recent
// entries first and stop after the requested limit.
func reverseChronoSegment(t time.Time) string {
	v := uint64(math.MaxInt64) - uint64(t.UnixNano())
	var buf [13]byte
	for i := 12; i >= 0; i-- {
		buf[i] = sortableBase32[v&31]
		v >>= 5
	}
	return string(buf[:])
}

// listByPresortedIndex scans a secondary index whose keys embed a
// reverseChronoSegment ahead of the trailing entity id, so iteration already
// yields entities newest first. It resolves each entity by the last key
// segment, applies the optional filter, and stops as soon as limit items have
// been collected (limit <= 0 means no limit). This pushes the limit down to the
// scan rather than sorting the whole match set in memory.
func listByPresortedIndex[T any](s *KVStorage, prefix string, getEntity func(happydns.Identifier) (*T, error), limit int, filter func(*T) bool) ([]*T, error) {
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
		if filter != nil && !filter(entity) {
			continue
		}
		results = append(results, entity)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results, iter.Err()
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

// tidyOwnerTimeIndex removes stale entries from a time sortable secondary index
// of the form prefix{ownerId}|{revTime}|{entityId}. The owner id is the first
// segment after the prefix and the entity id is the last segment, so the middle
// reverseChronoSegment is ignored. Entries with an unparseable owner or entity,
// a failing owner validation, or a missing entity are deleted.
func (s *KVStorage) tidyOwnerTimeIndex(prefix, label string, validateOwner func(happydns.Identifier) bool, entityExists func(happydns.Identifier) bool) {
	iter := s.db.Search(prefix)
	defer iter.Release()
	for iter.Next() {
		key := iter.Key()
		rest := strings.TrimPrefix(key, prefix)
		ownerStr, _, ok := strings.Cut(rest, "|")
		if !ok {
			_ = s.db.Delete(key)
			continue
		}

		ownerId, err := happydns.NewIdentifierFromString(ownerStr)
		if err != nil {
			_ = s.db.Delete(key)
			continue
		}

		lastPipe := strings.LastIndex(key, "|")
		entityId, err := happydns.NewIdentifierFromString(key[lastPipe+1:])
		if err != nil {
			_ = s.db.Delete(key)
			continue
		}

		if validateOwner != nil && !validateOwner(ownerId) {
			log.Printf("Deleting stale %s index (%s %s not found): %s\n", label, label, ownerStr, key)
			_ = s.db.Delete(key)
			continue
		}

		if !entityExists(entityId) {
			log.Printf("Deleting stale %s index (entity %s not found): %s\n", label, key[lastPipe+1:], key)
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

// countByPrefix counts the number of keys matching the given prefix without
// decoding their values. It is the foundation of the Count* methods exposed
// to observability code.
func (s *KVStorage) countByPrefix(prefix string) (int, error) {
	iter := s.db.Search(prefix)
	defer iter.Release()

	n := 0
	for iter.Next() {
		n++
	}
	return n, iter.Err()
}
