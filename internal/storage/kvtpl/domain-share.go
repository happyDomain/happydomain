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
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// Secondary indexes for domain sharing, i.e. granting a non-owner user
// (grantee) access to a Domain:
//
//	domain.share|{domainId}|{granteeId}  -> ""   list grantees of a domain
//	domain.grant|{granteeId}|{domainId}  -> ""   list domains shared with a user
//
// Both keys are written and deleted together in a single atomic batch so the
// two views never diverge on a partial failure.
const (
	domainShareIndexPrefix = "domain.share|"
	domainGrantIndexPrefix = "domain.grant|"
)

func domainShareIndexKey(domainId, granteeId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", domainShareIndexPrefix, domainId.String(), granteeId.String())
}

func domainGrantIndexKey(granteeId, domainId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", domainGrantIndexPrefix, granteeId.String(), domainId.String())
}

func (s *KVStorage) AddDomainShare(domainId, granteeId happydns.Identifier) error {
	batch := s.db.NewBatch()
	if err := batch.Put(domainShareIndexKey(domainId, granteeId), ""); err != nil {
		return err
	}
	if err := batch.Put(domainGrantIndexKey(granteeId, domainId), ""); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) DeleteDomainShare(domainId, granteeId happydns.Identifier) error {
	batch := s.db.NewBatch()
	batch.Delete(domainShareIndexKey(domainId, granteeId))
	batch.Delete(domainGrantIndexKey(granteeId, domainId))
	return batch.Commit()
}

func (s *KVStorage) IsDomainSharedWith(domainId, granteeId happydns.Identifier) (bool, error) {
	return s.db.Has(domainShareIndexKey(domainId, granteeId))
}

func (s *KVStorage) ListDomainShares(domainId happydns.Identifier) ([]happydns.Identifier, error) {
	prefix := fmt.Sprintf("%s%s|", domainShareIndexPrefix, domainId.String())
	return s.listShareGrantees(prefix)
}

func (s *KVStorage) ListSharedDomainIDs(granteeId happydns.Identifier) ([]happydns.Identifier, error) {
	prefix := fmt.Sprintf("%s%s|", domainGrantIndexPrefix, granteeId.String())
	return s.listShareGrantees(prefix)
}

// ListAllDomainShares enumerates every (domain, grantee) sharing grant. It
// reconciles both secondary indexes (share and grant views) so a grant left
// half-written by a crashed non-atomic write is still surfaced; duplicates are
// collapsed. Used by the tidy maintenance pass to detect and drop orphans.
func (s *KVStorage) ListAllDomainShares() ([]happydns.DomainShareBinding, error) {
	seen := map[string]bool{}
	var ret []happydns.DomainShareBinding

	// share index: domain.share|{domainId}|{granteeId}
	shareIter := s.db.Search(domainShareIndexPrefix)
	for shareIter.Next() {
		domainId, granteeId, err := parseTwoIdKey(shareIter.Key(), domainShareIndexPrefix)
		if err != nil {
			continue
		}
		key := domainId.String() + "|" + granteeId.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		ret = append(ret, happydns.DomainShareBinding{DomainId: domainId, GranteeId: granteeId})
	}
	err := shareIter.Err()
	shareIter.Release()
	if err != nil {
		return nil, err
	}

	// grant index: domain.grant|{granteeId}|{domainId} (segments reversed)
	grantIter := s.db.Search(domainGrantIndexPrefix)
	for grantIter.Next() {
		granteeId, domainId, err := parseTwoIdKey(grantIter.Key(), domainGrantIndexPrefix)
		if err != nil {
			continue
		}
		key := domainId.String() + "|" + granteeId.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		ret = append(ret, happydns.DomainShareBinding{DomainId: domainId, GranteeId: granteeId})
	}
	err = grantIter.Err()
	grantIter.Release()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// listShareGrantees iterates a share/grant prefix and returns the identifier in
// the last key segment of each entry.
func (s *KVStorage) listShareGrantees(prefix string) ([]happydns.Identifier, error) {
	iter := s.db.Search(prefix)
	defer iter.Release()

	var ret []happydns.Identifier
	for iter.Next() {
		id, err := lastKeySegment(iter.Key())
		if err != nil {
			continue
		}
		ret = append(ret, id)
	}
	return ret, iter.Err()
}
