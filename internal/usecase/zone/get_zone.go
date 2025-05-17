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
	"fmt"

	"git.happydns.org/happyDomain/model"
)

type GetZoneUsecase struct {
	store ZoneStorage
}

func NewGetZoneUsecase(store ZoneStorage) *GetZoneUsecase {
	return &GetZoneUsecase{
		store: store,
	}
}

func (uc *GetZoneUsecase) Get(zoneID happydns.Identifier) (*happydns.Zone, error) {
	zonemsg, err := uc.store.GetZone(zoneID)
	if err != nil {
		return nil, err
	}

	return ParseZone(zonemsg)
}

func (uc *GetZoneUsecase) GetMeta(zoneID happydns.Identifier) (*happydns.ZoneMeta, error) {
	zonemsg, err := uc.store.GetZone(zoneID)
	if err != nil {
		return nil, err
	}

	return &zonemsg.ZoneMeta, nil
}

func (uc *GetZoneUsecase) GetInDomain(zoneID happydns.Identifier, domain *happydns.Domain) (*happydns.Zone, error) {
	// Check that the zoneid exists in the domain history
	if !domain.HasZone(zoneID) {
		return nil, happydns.NotFoundError{Msg: fmt.Sprintf("zone not found: %q", zoneID.String())}
	}

	zmsg, err := uc.store.GetZone(zoneID)
	if err != nil {
		return nil, happydns.NotFoundError{Msg: fmt.Sprintf("zone not found: %q", zoneID.String())}
	}

	return ParseZone(zmsg)
}
