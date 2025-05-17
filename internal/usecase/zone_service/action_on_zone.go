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

package zoneService

import (
	"fmt"

	domainUC "git.happydns.org/happyDomain/internal/usecase/domain"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type ActionOnDomainUsecase struct {
	domainUpdater *domainUC.UpdateDomainUsecase
	zoneCreator   *zoneUC.CreateZoneUsecase
}

func NewActionOnDomainUsecase(domainUpdater *domainUC.UpdateDomainUsecase, zoneCreator *zoneUC.CreateZoneUsecase) *ActionOnDomainUsecase {
	return &ActionOnDomainUsecase{
		domainUpdater: domainUpdater,
		zoneCreator:   zoneCreator,
	}
}

func (uc *ActionOnDomainUsecase) ActionOnEditableZone(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, act func(zone *happydns.Zone) error) (*happydns.Zone, error) {
	var err error
	newZone := zone

	if zone.CommitDate != nil || zone.Published != nil {
		// Create a new zone if the current one is in archived state
		newZone = zone.DerivateNew()

		err = uc.zoneCreator.Create(newZone)
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to CreateZone in importZone: %s\n", err),
				UserMessage: "Sorry, we are unable to create your zone.",
			}
		}

		domain.ZoneHistory = append(
			[]happydns.Identifier{newZone.Id}, domain.ZoneHistory...)

		err = uc.domainUpdater.Update(domain.Id, user, func(dn *happydns.Domain) {
			dn.ZoneHistory = domain.ZoneHistory
		})
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to UpdateDomain in importZone: %s\n", err),
				UserMessage: "Sorry, we are unable to create your zone.",
			}
		}
	}

	err = act(newZone)
	if err != nil {
		return nil, err
	}

	return newZone, nil
}
