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

package domain

import (
	happydns "git.happydns.org/happyDomain/model"
)

// ListAllDomains returns every domain in the system. Intended for
// administrative callers; iterator drainage is hidden from the caller.
func (s *Service) ListAllDomains() ([]*happydns.Domain, error) {
	iter, err := s.store.ListAllDomains()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var domains []*happydns.Domain
	for iter.Next() {
		domains = append(domains, iter.Item())
	}
	return domains, iter.Err()
}

// GetDomainByID retrieves a domain by its identifier without ownership
// verification. Intended for administrative callers.
func (s *Service) GetDomainByID(domainID happydns.Identifier) (*happydns.Domain, error) {
	return s.store.GetDomain(domainID)
}

// GetDomainsByFQDN looks up domains owned by user that match the given
// fully-qualified domain name. Intended for administrative callers that
// have already resolved the owner.
func (s *Service) GetDomainsByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return s.store.GetDomainByDN(user, fqdn)
}

// AdminCreateDomain persists domain as-is, bypassing the registration-time
// validations performed by CreateDomain.
func (s *Service) AdminCreateDomain(domain *happydns.Domain) error {
	return s.store.CreateDomain(domain)
}

// AdminUpdateDomain persists changes to domain. It is the write-side
// counterpart used by admin endpoints that mutate a Domain in memory and
// need to commit the result without going through the user-scoped
// fetch-mutate-save cycle of UpdateDomain.
func (s *Service) AdminUpdateDomain(domain *happydns.Domain) error {
	return s.store.UpdateDomain(domain)
}

// ClearDomains removes every domain from the database. Intended for
// administrative callers performing a full reset.
func (s *Service) ClearDomains() error {
	return s.store.ClearDomains()
}
