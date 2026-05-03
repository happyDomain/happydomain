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

package provider

import (
	happydns "git.happydns.org/happyDomain/model"
)

// ListAllProviderMetas returns the metadata of every provider in the
// system. Intended for administrative callers; iterator drainage is hidden
// from the caller.
func (s *Service) ListAllProviderMetas() ([]*happydns.ProviderMeta, error) {
	iter, err := s.store.ListAllProviders()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var res []*happydns.ProviderMeta
	for iter.Next() {
		p := iter.Item()
		res = append(res, &p.ProviderMeta)
	}
	return res, iter.Err()
}

// DeleteProviderByID force-removes the provider identified by providerID
// without ownership verification. Intended for administrative callers.
func (s *Service) DeleteProviderByID(providerID happydns.Identifier) error {
	return s.store.DeleteProvider(providerID)
}

// ClearProviders removes every provider from the database. Intended for
// administrative callers performing a full reset.
func (s *Service) ClearProviders() error {
	return s.store.ClearProviders()
}
