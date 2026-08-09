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

// Secondary index for provider sharing, granting a non-owner user (grantee)
// the right to run zone operations (retrieve/apply/diff) that need the
// provider credentials:
//
//	provider.share|{providerId}|{granteeId}  -> ""
//
// A single index is enough: it answers both "is this provider shared with the
// user?" (membership test) and "who is it shared with?" (prefix scan for
// cleanup). The grantee never lists providers by this index.
const providerShareIndexPrefix = "provider.share|"

func providerShareIndexKey(providerId, granteeId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", providerShareIndexPrefix, providerId.String(), granteeId.String())
}

func (s *KVStorage) AddProviderShare(providerId, granteeId happydns.Identifier) error {
	return s.db.Put(providerShareIndexKey(providerId, granteeId), "")
}

func (s *KVStorage) DeleteProviderShare(providerId, granteeId happydns.Identifier) error {
	return s.db.Delete(providerShareIndexKey(providerId, granteeId))
}

func (s *KVStorage) IsProviderSharedWith(providerId, granteeId happydns.Identifier) (bool, error) {
	return s.db.Has(providerShareIndexKey(providerId, granteeId))
}

// ListAllProviderShares enumerates every (provider, grantee) sharing grant.
// Used by the tidy maintenance pass to detect and drop orphans.
func (s *KVStorage) ListAllProviderShares() ([]happydns.ProviderShareBinding, error) {
	iter := s.db.Search(providerShareIndexPrefix)
	defer iter.Release()

	var ret []happydns.ProviderShareBinding
	for iter.Next() {
		providerId, granteeId, err := parseTwoIdKey(iter.Key(), providerShareIndexPrefix)
		if err != nil {
			continue
		}
		ret = append(ret, happydns.ProviderShareBinding{ProviderId: providerId, GranteeId: granteeId})
	}
	return ret, iter.Err()
}

func (s *KVStorage) ListProviderShares(providerId happydns.Identifier) ([]happydns.Identifier, error) {
	prefix := fmt.Sprintf("%s%s|", providerShareIndexPrefix, providerId.String())
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
