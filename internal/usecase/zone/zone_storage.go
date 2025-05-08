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

package zone

import (
	"git.happydns.org/happyDomain/model"
)

type ZoneStorage interface {
	// ListAllZones retrieves the list of known Zones.
	ListAllZones() (happydns.Iterator[happydns.ZoneMessage], error)

	// GetZoneMeta retrieves metadatas of the Zone with the given identifier.
	GetZoneMeta(zoneid happydns.Identifier) (*happydns.ZoneMeta, error)

	// GetZone retrieves the full Zone (including Services and metadatas) which have the given identifier.
	GetZone(zoneid happydns.Identifier) (*happydns.ZoneMessage, error)

	// CreateZone creates a record in the database for the given Zone.
	CreateZone(zone *happydns.Zone) error

	// UpdateZone updates the fields of the given Zone.
	UpdateZone(zone *happydns.Zone) error

	// DeleteZone removes the given Zone from the database.
	DeleteZone(zoneid happydns.Identifier) error

	// ClearZones deletes all Zones present in the database.
	ClearZones() error
}
