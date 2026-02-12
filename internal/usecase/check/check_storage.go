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

package check

import (
	"git.happydns.org/happyDomain/model"
)

type CheckerStorage interface {
	// ListAllCheckConfigurations retrieves the list of known Providers.
	ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptions], error)

	// ListCheckerConfiguration retrieves all providers own by the given User.
	ListCheckerConfiguration(string) ([]*happydns.CheckerOptionsPositional, error)

	// GetCheckerConfiguration retrieves the full Provider with the given identifier and owner.
	GetCheckerConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier) ([]*happydns.CheckerOptionsPositional, error)

	// UpdateCheckerConfiguration updates the fields of the given Provider.
	UpdateCheckerConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier, happydns.CheckerOptions) error

	// DeleteCheckerConfiguration removes the given Provider from the database.
	DeleteCheckerConfiguration(string, *happydns.Identifier, *happydns.Identifier, *happydns.Identifier) error

	// ClearCheckerConfigurations deletes all Providers present in the database.
	ClearCheckerConfigurations() error
}

// CheckAutoFillStorage provides the domain/zone/user lookups needed to
// resolve auto-fill variables for test check options.
type CheckAutoFillStorage interface {
	// GetDomain retrieves the Domain with the given identifier.
	GetDomain(domainid happydns.Identifier) (*happydns.Domain, error)

	// GetUser retrieves the User with the given identifier.
	GetUser(userid happydns.Identifier) (*happydns.User, error)

	// ListDomains retrieves all Domains associated to the given User.
	ListDomains(user *happydns.User) ([]*happydns.Domain, error)

	// GetZone retrieves the full Zone (including Services and metadata) for the given identifier.
	GetZone(zoneid happydns.Identifier) (*happydns.ZoneMessage, error)
}
