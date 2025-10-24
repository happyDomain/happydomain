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

package domain

import (
	domainLogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	CreateDomainUC *CreateDomainUsecase
	DeleteDomainUC *DeleteDomainUsecase
	GetDomainUC    *GetDomainUsecase
	ListDomainsUC  *ListDomainsUsecase
	UpdateDomainUC *UpdateDomainUsecase
}

func NewDomainUsecases(
	store DomainStorage,
	getProviderUC *providerUC.GetProviderUsecase,
	getZoneUC *zoneUC.GetZoneUsecase,
	domainExistenceUC *providerUC.DomainExistenceUsecase,
	domainLogAppenderUC domainLogUC.DomainLogAppender,
) *Service {
	getDomainUC := NewGetDomainUsecase(store, getZoneUC)

	return &Service{
		CreateDomainUC: NewCreateDomainUsecase(store, getProviderUC, domainExistenceUC, domainLogAppenderUC),
		DeleteDomainUC: NewDeleteDomainUsecase(store),
		GetDomainUC:    getDomainUC,
		ListDomainsUC:  NewListDomainsUsecase(store),
		UpdateDomainUC: NewUpdateDomainUsecase(store, getDomainUC, domainLogAppenderUC),
	}
}

func (s *Service) CreateDomain(user *happydns.User, uz *happydns.Domain) error {
	return s.CreateDomainUC.Create(user, uz)
}

func (s *Service) DeleteDomain(domainid happydns.Identifier) error {
	return s.DeleteDomainUC.Delete(domainid)
}

func (s *Service) ExtendsDomainWithZoneMeta(domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	return s.GetDomainUC.ExtendsDomainWithZoneMeta(domain)
}

func (s *Service) GetUserDomain(user *happydns.User, domainID happydns.Identifier) (*happydns.Domain, error) {
	return s.GetDomainUC.ByID(user, domainID)
}

func (s *Service) GetUserDomainByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return s.GetDomainUC.ByFQDN(user, fqdn)
}

func (s *Service) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	return s.ListDomainsUC.List(user)
}

func (s *Service) UpdateDomain(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	return s.UpdateDomainUC.Update(domainID, user, updateFn)
}
