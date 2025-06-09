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

package provider

import (
	"git.happydns.org/happyDomain/model"
)

type Service struct {
	CreateProviderUC         *CreateProviderUsecase
	CreateDomainOnProviderUC *CreateDomainOnProviderUsecase
	DeleteProviderUC         *DeleteProviderUsecase
	UpdateProviderUC         *UpdateProviderUsecase
	ListHostedDomainsUC      *ListHostedDomainsUsecase
	GetProviderUC            *GetProviderUsecase
	ListProvidersUC          *ListProvidersUsecase
	RetrieveZoneUC           *ZoneRetrieverUsecase
	ZoneCorrectionsUC        *ZoneCorrectorUsecase
	DomainExistenceUC        *DomainExistenceUsecase
}

func NewProviderUsecases(store ProviderStorage) *Service {
	getProvider := NewGetProviderUsecase(store)
	validator := &DefaultProviderValidator{}

	return &Service{
		CreateProviderUC:         NewCreateProviderUsecase(store, validator),
		CreateDomainOnProviderUC: NewCreateDomainOnProviderUsecase(),
		DeleteProviderUC:         NewDeleteProviderUsecase(store),
		UpdateProviderUC:         NewUpdateProviderUsecase(store, getProvider, validator),
		ListHostedDomainsUC:      NewListHostedDomainsUsecase(),
		GetProviderUC:            getProvider,
		ListProvidersUC:          NewListProvidersUsecase(store),
		RetrieveZoneUC:           NewZoneRetrieverUsecase(),
		ZoneCorrectionsUC:        NewZoneCorrectorUsecase(),
		DomainExistenceUC:        NewDomainExistenceUsecase(),
	}
}

func (s *Service) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	return s.CreateProviderUC.Create(user, msg)
}

func (s *Service) CreateDomainOnProvider(provider *happydns.Provider, fqdn string) error {
	return s.CreateDomainOnProviderUC.Create(provider, fqdn)
}

func (s *Service) DeleteProvider(user *happydns.User, providerID happydns.Identifier) error {
	return s.DeleteProviderUC.Delete(user, providerID)
}

func (s *Service) GetUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	return s.GetProviderUC.Get(user, providerID)
}

func (s *Service) GetUserProviderMeta(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	return s.GetProviderUC.GetMeta(user, providerID)
}

func (s *Service) ListHostedDomains(provider *happydns.Provider) ([]string, error) {
	return s.ListHostedDomainsUC.List(provider)
}

func (s *Service) ListZoneCorrections(provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, error) {
	return s.ZoneCorrectionsUC.List(provider, domain, records)
}

func (s *Service) ListUserProviders(user *happydns.User) ([]*happydns.ProviderMeta, error) {
	return s.ListProvidersUC.List(user)
}

func (s *Service) RetrieveZone(provider *happydns.Provider, name string) ([]happydns.Record, error) {
	return s.RetrieveZoneUC.RetrieveCurrentZone(provider, name)
}

func (s *Service) TestDomainExistence(provider *happydns.Provider, name string) error {
	return s.DomainExistenceUC.TestDomainExistence(provider, name)
}

func (s *Service) UpdateProvider(providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	return s.UpdateProviderUC.Update(providerID, user, updateFn)
}

func (s *Service) UpdateProviderFromMessage(providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	return s.UpdateProviderUC.FromMessage(providerID, user, p)
}

type RestrictedService struct {
	Service
	config *happydns.Options
}

func NewRestrictedProviderUsecases(cfg *happydns.Options, store ProviderStorage) *RestrictedService {
	s := NewProviderUsecases(store)
	return &RestrictedService{
		*s,
		cfg,
	}
}

func (s *RestrictedService) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	if s.config.DisableProviders {
		return nil, happydns.ForbiddenError{Msg: "cannot add provider as DisableProviders parameter is set."}
	}

	return s.Service.CreateProvider(user, msg)
}

func (s *RestrictedService) CreateDomainOnProvider(provider *happydns.Provider, fqdn string) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot create domain on provider as DisableProviders parameter is set."}
	}

	return s.Service.CreateDomainOnProvider(provider, fqdn)
}

func (s *RestrictedService) DeleteProvider(user *happydns.User, providerID happydns.Identifier) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot delete provider as DisableProviders parameter is set."}
	}

	return s.Service.DeleteProvider(user, providerID)
}

func (s *RestrictedService) UpdateProvider(providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.Service.UpdateProvider(providerID, user, updateFn)
}

func (s *RestrictedService) UpdateProviderFromMessage(providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.Service.UpdateProviderFromMessage(providerID, user, p)
}
