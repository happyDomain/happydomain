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

package usecase

import (
	"encoding/json"
	"fmt"

	"git.happydns.org/happyDomain/internal/config"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

type providerUsecase struct {
	happydns.ProviderUsecase
	config *config.Options
}

func NewProviderUsecase(cfg *config.Options, store storage.ProviderAndDomainStorage) happydns.ProviderUsecase {
	return &providerUsecase{
		ProviderUsecase: NewAdminProviderUsecase(store),
		config:          cfg,
	}
}

func (pu *providerUsecase) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	if pu.config.DisableProviders {
		return nil, happydns.ForbiddenError{"Cannot add provider as DisableProviders parameter is set."}
	}

	return pu.ProviderUsecase.CreateProvider(user, msg)
}

func (pu *providerUsecase) DeleteProvider(user *happydns.User, providerid happydns.Identifier) error {
	if pu.config.DisableProviders {
		return happydns.ForbiddenError{"Cannot delete provider as DisableProviders parameter is set."}
	}

	return pu.ProviderUsecase.DeleteProvider(user, providerid)
}

func (pu *providerUsecase) UpdateProvider(providerid happydns.Identifier, user *happydns.User, upd func(*happydns.Provider)) error {
	if pu.config.DisableProviders {
		return happydns.ForbiddenError{"Cannot update provider as DisableProviders parameter is set."}
	}

	return pu.ProviderUsecase.UpdateProvider(providerid, user, upd)
}

func (pu *providerUsecase) UpdateProviderFromMessage(providerid happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	if pu.config.DisableProviders {
		return happydns.ForbiddenError{"Cannot update provider as DisableProviders parameter is set."}
	}

	return pu.ProviderUsecase.UpdateProviderFromMessage(providerid, user, p)
}

type adminProviderUsecase struct {
	store storage.ProviderAndDomainStorage
}

func NewAdminProviderUsecase(store storage.ProviderAndDomainStorage) happydns.ProviderUsecase {
	return &adminProviderUsecase{
		store: store,
	}
}

func (pu *adminProviderUsecase) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse provider attributes: %w", err)
	}

	err = pu.ValidateProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("unable to validate provider attributes: %w", err)
	}

	provider.Owner = user.Id

	err = pu.store.CreateProvider(provider)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
		}
	}

	return provider, nil
}

func (pu *adminProviderUsecase) DeleteProvider(user *happydns.User, providerid happydns.Identifier) error {
	// Check if the provider has no more domain associated
	domains, err := pu.store.ListDomains(user)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to GetDomains for user id=%x email=%s: %w", user.Id, user.Email, err),
			UserMessage: "Sorry, we are currently unable to perform this action. Please try again later.",
		}
	}

	for _, domain := range domains {
		if domain.ProviderId.Equals(providerid) {
			return fmt.Errorf("You cannot delete this provider because there is still some domains associated with it. For example: %s", domain.DomainName)
		}
	}

	if err := pu.store.DeleteProvider(providerid); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteProvider %s for user id=%x email=%s: %w", providerid.String(), user.Id.String(), user.Email, err),
			UserMessage: "Sorry, we are currently unable to delete your provider. Please try again later.",
		}
	}

	return nil
}

func (pu *adminProviderUsecase) getUserProvider(user *happydns.User, pid happydns.Identifier) (*happydns.ProviderMessage, error) {
	p, err := pu.store.GetProvider(pid)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(p.ProviderMeta.Owner) {
		return nil, fmt.Errorf("Provider not found")
	}

	return p, err
}

func (pu *adminProviderUsecase) GetUserProvider(user *happydns.User, pid happydns.Identifier) (*happydns.Provider, error) {
	p, err := pu.getUserProvider(user, pid)
	if err != nil {
		return nil, err
	}

	return ParseProvider(p)
}

func (pu *adminProviderUsecase) GetUserProviderMeta(user *happydns.User, pid happydns.Identifier) (*happydns.ProviderMeta, error) {
	p, err := pu.getUserProvider(user, pid)
	if err != nil {
		return nil, err
	}

	return p.Meta(), nil
}

func (pu *adminProviderUsecase) GetZoneCorrections(provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, error) {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}

	return instance.GetZoneCorrections(domain.DomainName, records)
}

func (pu *adminProviderUsecase) ListUserProviders(user *happydns.User) ([]*happydns.ProviderMeta, error) {
	providers, err := pu.store.ListProviders(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserProviders: %s", err.Error())
	}

	if len(providers) == 0 {
		return []*happydns.ProviderMeta{}, nil
	}

	var ret []*happydns.ProviderMeta

	for _, p := range providers {
		ret = append(ret, &p.ProviderMeta)
	}

	return ret, nil
}

func ParseProvider(msg *happydns.ProviderMessage) (p *happydns.Provider, err error) {
	p = &happydns.Provider{}

	p.ProviderMeta = msg.ProviderMeta
	p.Provider, err = providers.FindProvider(msg.Type)
	if err != nil {
		return
	}

	err = json.Unmarshal(msg.Provider, &p.Provider)
	return
}

func (pu *adminProviderUsecase) RetrieveZone(provider *happydns.Provider, domain string) ([]happydns.Record, error) {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}

	return instance.GetZoneRecords(domain)
}

func (pu *adminProviderUsecase) TestDomainExistence(provider *happydns.Provider, domain string) error {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return fmt.Errorf("unable to instantiate provider: %w", err)
	}

	_, err = instance.GetZoneRecords(domain)
	return err
}

func (pu *adminProviderUsecase) UpdateProvider(providerid happydns.Identifier, user *happydns.User, upd func(*happydns.Provider)) error {
	provider, err := pu.GetUserProvider(user, providerid)
	if err != nil {
		return err
	}

	upd(provider)

	if !provider.Id.Equals(providerid) {
		return happydns.ValidationError{"you cannot change the provider identifier"}
	}

	err = pu.ValidateProvider(provider)
	if err != nil {
		return happydns.ValidationError{fmt.Sprintf("unable to validate provider attributes: %s", err.Error())}
	}

	err = pu.store.UpdateProvider(provider)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateProvider in UpdateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to update your provider. Please retry later.",
		}
	}

	return nil
}

func (pu *adminProviderUsecase) UpdateProviderFromMessage(providerid happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	newprovider, err := ParseProvider(p)
	if err != nil {
		return err
	}

	return pu.UpdateProvider(providerid, user, func(provider *happydns.Provider) {
		*provider = *newprovider
	})
}

func (pu *adminProviderUsecase) ValidateProvider(provider *happydns.Provider) error {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return fmt.Errorf("unable to instantiate provider: %w", err)
	}

	if sr, ok := instance.(happydns.ZoneLister); ok {
		_, err = sr.ListZones()
	}

	return err
}
