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

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type ZoneDifferUsecase struct {
	listRecords *ListRecordsUsecase
	getZone     *GetZoneUsecase
}

func NewZoneDifferUsecase(getZone *GetZoneUsecase, listRecords *ListRecordsUsecase) *ZoneDifferUsecase {
	return &ZoneDifferUsecase{
		listRecords: listRecords,
		getZone:     getZone,
	}
}

func (uc *ZoneDifferUsecase) Diff(domain *happydns.Domain, newzone *happydns.Zone, oldzoneid happydns.Identifier) ([]*happydns.Correction, error) {
	oldzone, err := uc.getZone.GetInDomain(oldzoneid, domain)
	if err != nil {
		return nil, err
	}

	oldrecords, err := uc.listRecords.List(domain, oldzone)
	if err != nil {
		return nil, happydns.InternalError{
			Err: fmt.Errorf("unable to retrieve records for old zone: %w", err),
		}
	}

	newrecords, err := uc.listRecords.List(domain, newzone)
	if err != nil {
		return nil, err
	}

	corrections, _, err := adapter.DNSControlDiffByRecord(oldrecords, newrecords, domain.DomainName)
	if err != nil {
		return nil, err
	}

	return corrections, nil
}
