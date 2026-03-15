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

// DeleteZoneUsecase handles removing a Zone from storage.
type DeleteZoneUsecase struct {
	store ZoneStorage
}

// NewDeleteZoneUsecase constructs a DeleteZoneUsecase backed by the given
// storage implementation.
func NewDeleteZoneUsecase(store ZoneStorage) *DeleteZoneUsecase {
	return &DeleteZoneUsecase{
		store: store,
	}
}

// Delete removes the zone identified by zoneID from storage.
func (uc *DeleteZoneUsecase) Delete(zoneID happydns.Identifier) error {
	return uc.store.DeleteZone(zoneID)
}
