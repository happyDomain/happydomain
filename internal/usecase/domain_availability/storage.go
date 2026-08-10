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

package domain_availability

import (
	"git.happydns.org/happyDomain/model"
)

type DomainAvailabilityWatchStorage interface {
	// ListAllDomainAvailabilityWatches retrieves every registered watch.
	ListAllDomainAvailabilityWatches() (happydns.Iterator[happydns.DomainAvailabilityWatch], error)

	// ListDomainAvailabilityWatches retrieves all watches owned by the User.
	ListDomainAvailabilityWatches(user *happydns.User) ([]*happydns.DomainAvailabilityWatch, error)

	// ExistsDomainAvailabilityWatch reports whether owner already watches
	// domainName, without listing and loading every watch they own.
	ExistsDomainAvailabilityWatch(owner happydns.Identifier, domainName string) (bool, error)

	// GetDomainAvailabilityWatch retrieves the watch with the given id.
	GetDomainAvailabilityWatch(id happydns.Identifier) (*happydns.DomainAvailabilityWatch, error)

	// CreateDomainAvailabilityWatch persists a new watch.
	CreateDomainAvailabilityWatch(watch *happydns.DomainAvailabilityWatch) error

	// UpdateDomainAvailabilityWatch persists an existing watch, preserving its
	// id. Used by the backup restore path.
	UpdateDomainAvailabilityWatch(watch *happydns.DomainAvailabilityWatch) error

	// DeleteDomainAvailabilityWatch removes the watch with the given id.
	DeleteDomainAvailabilityWatch(id happydns.Identifier) error

	// ClearDomainAvailabilityWatches deletes all watches.
	ClearDomainAvailabilityWatches() error
}
