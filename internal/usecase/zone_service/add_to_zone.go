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

	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	"git.happydns.org/happyDomain/model"
)

type AddToZoneUsecase struct {
	store           serviceUC.ZoneUpdaterStorage
	validateService *serviceUC.ValidateServiceUsecase
}

func NewAddToZoneUsecase(store serviceUC.ZoneUpdaterStorage, validateService *serviceUC.ValidateServiceUsecase) *AddToZoneUsecase {
	return &AddToZoneUsecase{
		store:           store,
		validateService: validateService,
	}
}

func (uc *AddToZoneUsecase) AddService(zone *happydns.Zone, subdomain happydns.Subdomain, origin happydns.Origin, service *happydns.Service) error {
	if service.Service == nil {
		return happydns.ValidationError{Msg: "Unable to parse the given service."}
	}

	hash, err := uc.validateService.Validate(service.Service, subdomain, origin)
	if err != nil {
		return err
	}

	service.Id = hash
	service.Domain = string(subdomain)
	service.NbResources = service.Service.GetNbResources()
	service.Comment = service.Service.GenComment()

	zone.Services[subdomain] = append(zone.Services[subdomain], service)

	err = uc.store.UpdateZone(zone)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("Unable to UpdateZone in AppendService: %w", err),
			UserMessage: "Sorry, we are currently unable to update your zone. Please retry later.",
		}
	}

	return nil
}
