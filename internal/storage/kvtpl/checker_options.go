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
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
)

const (
	// checkerOptionPrimaryPrefix is the prefix for all checker option primaries.
	// Key layout: "cfg|" (4) + hash28(compound) (28) = 32 chars.
	checkerOptionPrimaryPrefix = "cfg|"

	// checkerOptionNameIndexPrefix is the secondary index for scan-by-checker-name.
	// Key layout: "cfg-c|" (6) + hash28(checkerName) (28) + "|" (1) + hash28(compound) (28) = 63 chars.
	checkerOptionNameIndexPrefix = "cfg-c|"
)

// checkerOptionsCompound builds the canonical compound string used as the hash
// input for the primary key.
func checkerOptionsCompound(checkerName string, userId, domainId, serviceId *happydns.Identifier) string {
	return checkerName + "|" +
		happydns.FormatIdentifier(userId) + "|" +
		happydns.FormatIdentifier(domainId) + "|" +
		happydns.FormatIdentifier(serviceId)
}

// checkerOptionsKey returns the primary key for a checker option entry.
// The full (checkerName, userId, domainId, serviceId) compound is hashed so
// the key is bounded regardless of checker name or identifier count.
func checkerOptionsKey(checkerName string, userId, domainId, serviceId *happydns.Identifier) string {
	return checkerOptionPrimaryPrefix + hash28(checkerOptionsCompound(checkerName, userId, domainId, serviceId))
}

// checkerOptionNameIndexKey returns the secondary index key used to enumerate
// all option entries for a given checker name.
func checkerOptionNameIndexKey(checkerName, compoundHash string) string {
	return checkerOptionNameIndexPrefix + hash28(checkerName) + "|" + compoundHash
}

func (s *KVStorage) ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptionsPositional], error) {
	iter := s.db.Search(checkerOptionPrimaryPrefix)
	return NewKVIterator[happydns.CheckerOptionsPositional](s.db, iter), nil
}

func (s *KVStorage) ListCheckerConfiguration(checkerName string) ([]*happydns.CheckerOptionsPositional, error) {
	prefix := checkerOptionNameIndexPrefix + hash28(checkerName) + "|"

	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.CheckerOptionsPositional
	for iter.Next() {
		i := strings.LastIndex(iter.Key(), "|")
		if i < 0 {
			continue
		}
		compoundHash := iter.Key()[i+1:]
		primaryKey := checkerOptionPrimaryPrefix + compoundHash

		var stored happydns.CheckerOptionsPositional
		if err := s.db.Get(primaryKey, &stored); err != nil {
			log.Printf("ListCheckerConfiguration: error loading checker config at key %q: %s", primaryKey, err)
			continue
		}
		results = append(results, &stored)
	}
	return results, nil
}

func (s *KVStorage) GetCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) ([]*happydns.CheckerOptionsPositional, error) {
	var results []*happydns.CheckerOptionsPositional

	// Try each scope level from admin up to the requested specificity.
	scopes := []struct {
		uid, did, sid *happydns.Identifier
	}{
		{nil, nil, nil},
		{userId, nil, nil},
		{userId, domainId, nil},
		{userId, domainId, serviceId},
	}

	for _, sc := range scopes {
		// Skip levels that require identifiers not provided.
		if (sc.uid != nil && userId == nil) || (sc.did != nil && domainId == nil) || (sc.sid != nil && serviceId == nil) {
			continue
		}

		key := checkerOptionsKey(checkerName, sc.uid, sc.did, sc.sid)
		var stored happydns.CheckerOptionsPositional
		if err := s.db.Get(key, &stored); err == nil {
			results = append(results, &stored)
		}
	}

	return results, nil
}

func (s *KVStorage) UpdateCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	compoundHash := hash28(checkerOptionsCompound(checkerName, userId, domainId, serviceId))
	primaryKey := checkerOptionPrimaryPrefix + compoundHash
	indexKey := checkerOptionNameIndexKey(checkerName, compoundHash)

	stored := happydns.CheckerOptionsPositional{
		CheckName: checkerName,
		UserId:    userId,
		DomainId:  domainId,
		ServiceId: serviceId,
		Options:   opts,
	}

	batch := s.db.NewBatch()
	if err := batch.Put(primaryKey, stored); err != nil {
		return err
	}
	if err := batch.Put(indexKey, ""); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) DeleteCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) error {
	compoundHash := hash28(checkerOptionsCompound(checkerName, userId, domainId, serviceId))
	primaryKey := checkerOptionPrimaryPrefix + compoundHash
	indexKey := checkerOptionNameIndexKey(checkerName, compoundHash)

	batch := s.db.NewBatch()
	batch.Delete(primaryKey)
	batch.Delete(indexKey)
	return batch.Commit()
}

func (s *KVStorage) ClearCheckerConfigurations() error {
	if err := s.clearByPrefix(checkerOptionNameIndexPrefix); err != nil {
		return err
	}
	return s.clearByPrefix(checkerOptionPrimaryPrefix)
}
