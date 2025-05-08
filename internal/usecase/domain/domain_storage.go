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

	// ListDomains retrieves all Domains associated to the given User.
	ListDomains(user *happydns.User) ([]*happydns.Domain, error)

	// GetDomain retrieves the Domain with the given id and owned by the given User.
	GetDomain(domainid happydns.Identifier) (*happydns.Domain, error)

	// GetDomainByDN is like GetDomain but look for the domain name instead of identifier.
	GetDomainByDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error)

	// CreateDomain creates a record in the database for the given Domain.
	CreateDomain(domain *happydns.Domain) error

	// UpdateDomain updates the fields of the given Domain.
	UpdateDomain(domain *happydns.Domain) error

	// DeleteDomain removes the given Domain from the database.
	DeleteDomain(domainid happydns.Identifier) error

	// ClearDomains deletes all Domains present in the database.
	ClearDomains() error

	// DOMAIN LOGS --------------------------------------------------

	ListAllDomainLogs() (happydns.Iterator[happydns.DomainLogWithDomainId], error)

	GetDomainLogs(domain *happydns.Domain) ([]*happydns.DomainLog, error)

	CreateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error

	UpdateDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error

	DeleteDomainLog(domain *happydns.Domain, log *happydns.DomainLog) error
}
