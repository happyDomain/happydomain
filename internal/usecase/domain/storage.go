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

package domain

import (
	"git.happydns.org/happyDomain/model"
)

type DomainStorage interface {
	// ListAllDomains retrieves the list of known Domains.
	ListAllDomains() (happydns.Iterator[happydns.Domain], error)

	// CountDomains returns the total number of Domains in storage.
	// Implementations should make this efficient (e.g. count keys without
	// decoding values) so it can be called from observability paths.
	CountDomains() (int, error)

	// ListDomains retrieves all Domains associated to the given User.
	ListDomains(user *happydns.User) ([]*happydns.Domain, error)

	// GetDomain retrieves the Domain with the given id and owned by the given User.
	GetDomain(domainid happydns.Identifier) (*happydns.Domain, error)

	// GetDomainByDN is like GetDomain but look for the domain name instead of identifier.
	GetDomainByDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error)

	// FindDomainsByName looks up Domains by FQDN across all users (no
	// ownership filter). Used by unauthenticated endpoints like the
	// email auto-configuration HTTP responders.
	FindDomainsByName(fqdn string) ([]*happydns.Domain, error)

	// CreateDomain creates a record in the database for the given Domain.
	CreateDomain(domain *happydns.Domain) error

	// UpdateDomain updates the fields of the given Domain.
	UpdateDomain(domain *happydns.Domain) error

	// DeleteDomain removes the given Domain from the database.
	DeleteDomain(domainid happydns.Identifier) error

	// ClearDomains deletes all Domains present in the database.
	ClearDomains() error

	// AddDomainShare grants the given user (grantee) access to the Domain.
	AddDomainShare(domainid, granteeid happydns.Identifier) error

	// DeleteDomainShare revokes a previously granted access to the Domain.
	DeleteDomainShare(domainid, granteeid happydns.Identifier) error

	// IsDomainSharedWith reports whether the Domain is shared with the grantee.
	IsDomainSharedWith(domainid, granteeid happydns.Identifier) (bool, error)

	// ListDomainShares lists the users the Domain is shared with.
	ListDomainShares(domainid happydns.Identifier) ([]happydns.Identifier, error)

	// ListSharedDomainIDs lists the Domains shared with the given user.
	ListSharedDomainIDs(granteeid happydns.Identifier) ([]happydns.Identifier, error)

	// ListAllDomainShares enumerates every (domain, grantee) sharing grant,
	// used by maintenance passes to detect orphaned grants.
	ListAllDomainShares() ([]happydns.DomainShareBinding, error)
}
