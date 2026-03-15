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

// Package zoneService implements the use cases for managing DNS services
// within a zone.  A "service" in happyDomain is a logical grouping of related
// DNS records under a subdomain.  The package provides:
//
//   - AddToZoneUsecase – validates and appends a new service to a zone.
//   - DeleteFromZoneUsecase – removes a service from a zone by ID.
//   - UpdateServiceUsecase – replaces a service in-place within a zone.
//   - ActionOnDomainUsecase – ensures mutations always target an editable
//     (non-committed) zone snapshot, creating a derivative zone when needed.
//
// The Service facade exposes all of the above through a unified API consumed
// by the HTTP handler layer.
package zoneService

import (
	serviceUC "git.happydns.org/happyDomain/internal/usecase/service"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// Service is the facade that composes all zone-service use cases and exposes
// them through a unified API consumed by the HTTP handler layer.
type Service struct {
	ActionOnDomainUC *ActionOnDomainUsecase
	AddToZoneUC      *AddToZoneUsecase
	DeleteFromZoneUC *DeleteFromZoneUsecase
	UpdateServiceUC  *UpdateServiceUsecase
}

// NewZoneServiceUsecases wires up and returns a fully initialized Service.
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

// ActionOnEditableZone delegates to ActionOnDomainUsecase.ActionOnEditableZone.
func (s *Service) ActionOnEditableZone(
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	act func(zone *happydns.Zone) error,
) (*happydns.Zone, error) {
	return s.ActionOnDomainUC.ActionOnEditableZone(user, domain, zone, act)
}

// AddServiceToZone delegates to AddToZoneUsecase.AddService.
func (s *Service) AddServiceToZone(
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	subdomain happydns.Subdomain,
	origin happydns.Origin,
	service *happydns.Service,
) (*happydns.Zone, error) {
	return zone, s.AddToZoneUC.AddService(zone, subdomain, origin, service)
}

// RemoveServiceFromZone delegates to DeleteFromZoneUsecase.DeleteService.
func (s *Service) RemoveServiceFromZone(
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	subdomain happydns.Subdomain,
	serviceID happydns.Identifier,
) (*happydns.Zone, error) {
	return zone, s.DeleteFromZoneUC.DeleteService(zone, subdomain, serviceID)
}

// UpdateZoneService delegates to UpdateServiceUsecase.Update.
func (s *Service) UpdateZoneService(
	user *happydns.User,
	domain *happydns.Domain,
	zone *happydns.Zone,
	subdomain happydns.Subdomain,
	serviceID happydns.Identifier,
	service *happydns.Service,
) (*happydns.Zone, error) {
	return zone, s.UpdateServiceUC.Update(zone, subdomain, serviceID, service)
}
