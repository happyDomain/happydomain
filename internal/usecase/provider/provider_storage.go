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

package provider

import (
	"git.happydns.org/happyDomain/model"
)

type ProviderStorage interface {
	// ListAllProviders retrieves the list of known Providers.
	ListAllProviders() (happydns.Iterator[happydns.ProviderMessage], error)

	// ListProviders retrieves all providers own by the given User.
	ListProviders(user *happydns.User) (happydns.ProviderMessages, error)

	// GetProvider retrieves the full Provider with the given identifier and owner.
	GetProvider(prvdid happydns.Identifier) (*happydns.ProviderMessage, error)

	// CreateProvider creates a record in the database for the given Provider.
	CreateProvider(prvd *happydns.Provider) error

	// UpdateProvider updates the fields of the given Provider.
	UpdateProvider(prvd *happydns.Provider) error

	// DeleteProvider removes the given Provider from the database.
	DeleteProvider(prvdid happydns.Identifier) error

	// ClearProviders deletes all Providers present in the database.
	ClearProviders() error
}
