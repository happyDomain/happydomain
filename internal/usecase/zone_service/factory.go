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
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	ActionOnDomainUC *ActionOnDomainUsecase
	AddToZoneUC      *AddToZoneUsecase
	DeleteFromZoneUC *DeleteFromZoneUsecase
	UpdateServiceUC  *UpdateServiceUsecase
}

func NewZoneServiceUsecases(
	domainUpdater DomainUpdater,
	zoneCreator *zoneUC.CreateZoneUsecase,
	validateService *serviceUC.ValidateServiceUsecase,
	store serviceUC.ZoneUpdaterStorage,
) *Service {
	return &Service{
		ActionOnDomainUC: NewActionOnDomainUsecase(domainUpdater, zoneCreator),
		AddToZoneUC:      NewAddToZoneUsecase(store, validateService),
		DeleteFromZoneUC: NewDeleteFromZoneUsecase(store),
		UpdateServiceUC:  NewUpdateServiceUsecase(store),
	}
}

func (s *Service) ActionOnEditableZone(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, act func(zone *happydns.Zone) error) (*happydns.Zone, error) {
	return s.ActionOnDomainUC.ActionOnEditableZone(user, domain, zone, act)
}

func (s *Service) AddServiceToZone(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, origin happydns.Origin, service *happydns.Service) (*happydns.Zone, error) {
	return zone, s.AddToZoneUC.AddService(zone, subdomain, origin, service)
}

func (s *Service) RemoveServiceFromZone(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, serviceID happydns.Identifier) (*happydns.Zone, error) {
	return zone, s.DeleteFromZoneUC.DeleteService(zone, subdomain, serviceID)
}

func (s *Service) UpdateZoneService(user *happydns.User, domain *happydns.Domain, zone *happydns.Zone, subdomain string, serviceID happydns.Identifier, service *happydns.Service) (*happydns.Zone, error) {
	return zone, s.UpdateServiceUC.Update(zone, subdomain, serviceID, service)
}
