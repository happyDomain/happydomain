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
	"strings"

	"git.happydns.org/happyDomain/model"
)

// checkerOptionsKey builds the positional KV key for checker options.
// Format: chckrcfg|{checkerName}|{userId}|{domainId}|{serviceId}
func checkerOptionsKey(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) string {
	return fmt.Sprintf("chckrcfg|%s|%s|%s|%s", checkerName,
		happydns.FormatIdentifier(userId), happydns.FormatIdentifier(domainId), happydns.FormatIdentifier(serviceId))
}

// parseCheckerOptionsKey extracts the positional components from a KV key.
func parseCheckerOptionsKey(key string) (checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) {
	trimmed := strings.TrimPrefix(key, "chckrcfg|")
	parts := strings.SplitN(trimmed, "|", 4)
	if len(parts) < 4 {
		return trimmed, nil, nil, nil
	}

	checkerName = parts[0]
	if parts[1] != "" {
		if id, err := happydns.NewIdentifierFromString(parts[1]); err == nil {
			userId = &id
		}
	}
	if parts[2] != "" {
		if id, err := happydns.NewIdentifierFromString(parts[2]); err == nil {
			domainId = &id
		}
	}
	if parts[3] != "" {
		if id, err := happydns.NewIdentifierFromString(parts[3]); err == nil {
			serviceId = &id
		}
	}
	return
}

func (s *KVStorage) ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptionsPositional], error) {
	iter := s.db.Search("chckrcfg|")
	return &checkerOptionsIterator{KVIterator: NewKVIterator[happydns.CheckerOptions](s.db, iter)}, nil
}

// checkerOptionsIterator wraps KVIterator[CheckerOptions] and enriches each
// item with positional fields parsed from the storage key.
type checkerOptionsIterator struct {
	*KVIterator[happydns.CheckerOptions]
}

func (it *checkerOptionsIterator) Item() *happydns.CheckerOptionsPositional {
	opts := it.KVIterator.Item()
	if opts == nil {
		return nil
	}
	cn, uid, did, sid := parseCheckerOptionsKey(it.Key())
	return &happydns.CheckerOptionsPositional{
		CheckName: cn,
		UserId:    uid,
		DomainId:  did,
		ServiceId: sid,
		Options:   *opts,
	}
}

func (s *KVStorage) ListCheckerConfiguration(checkerName string) ([]*happydns.CheckerOptionsPositional, error) {
	prefix := fmt.Sprintf("chckrcfg|%s|", checkerName)
	iter := s.db.Search(prefix)
	defer iter.Release()

	var results []*happydns.CheckerOptionsPositional
	for iter.Next() {
		var opts happydns.CheckerOptions
		if err := s.db.DecodeData(iter.Value(), &opts); err != nil {
			log.Printf("ListCheckerConfiguration: error decoding checker config at key %q: %s", iter.Key(), err)
			continue
		}

		cn, uid, did, sid := parseCheckerOptionsKey(iter.Key())
		results = append(results, &happydns.CheckerOptionsPositional{
			CheckName: cn,
			UserId:    uid,
			DomainId:  did,
			ServiceId: sid,
			Options:   opts,
		})
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
		var opts happydns.CheckerOptions
		if err := s.db.Get(key, &opts); err == nil {
			results = append(results, &happydns.CheckerOptionsPositional{
				CheckName: checkerName,
				UserId:    sc.uid,
				DomainId:  sc.did,
				ServiceId: sc.sid,
				Options:   opts,
			})
		}
	}

	return results, nil
}

func (s *KVStorage) UpdateCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	key := checkerOptionsKey(checkerName, userId, domainId, serviceId)
	return s.db.Put(key, opts)
}

func (s *KVStorage) DeleteCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) error {
	key := checkerOptionsKey(checkerName, userId, domainId, serviceId)
	return s.db.Delete(key)
}

func (s *KVStorage) ClearCheckerConfigurations() error {
	iter, err := s.ListAllCheckerConfigurations()
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.Next() {
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}
	return nil
}
