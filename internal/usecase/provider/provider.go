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
	"context"
	"encoding/json"
	"fmt"

	providerReg "git.happydns.org/happyDomain/internal/provider"
	"git.happydns.org/happyDomain/model"
)

// Service handles CRUD operations on DNS providers, with ownership enforcement.
type Service struct {
	store     ProviderStorage
	validator ProviderValidator
}

// NewService creates a new provider Service. If validator is nil,
// the DefaultProviderValidator is used.
func NewService(store ProviderStorage, validator ProviderValidator) *Service {
	if validator == nil {
		validator = &DefaultProviderValidator{}
	}
	return &Service{
		store:     store,
		validator: validator,
	}
}

// ParseProvider converts a ProviderMessage to a Provider.
func ParseProvider(msg *happydns.ProviderMessage) (p *happydns.Provider, err error) {
	p = &happydns.Provider{}

	p.ProviderMeta = msg.ProviderMeta
	p.Provider, err = providerReg.FindProvider(msg.Type)
	if err != nil {
		return
	}

	err = json.Unmarshal(msg.Provider, &p.Provider)
	return
}

// instantiate is a helper that instantiates a provider and wraps errors consistently.
func instantiate(p *happydns.Provider) (happydns.ProviderActuator, error) {
	instance, err := p.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}
	return instance, nil
}

// CreateProvider creates a new provider for the given user.
func (s *Service) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse provider: %w", err)
	}

	if err := s.validator.Validate(provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	provider.Owner = user.Id

	if err := s.store.CreateProvider(provider); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("failed to save provider: %w", err),
			UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
		}
	}

	return provider, nil
}

// getUserProvider retrieves a provider and verifies ownership.
func (s *Service) getUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMessage, error) {
	p, err := s.store.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(p.ProviderMeta.Owner) {
		return nil, happydns.ErrProviderNotFound
	}

	return p, err
}

// GetUserProvider retrieves a provider for the given user.
func (s *Service) GetUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	p, err := s.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	return ParseProvider(p)
}

// GetUserProviderMeta retrieves provider metadata for the given user.
func (s *Service) GetUserProviderMeta(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	p, err := s.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	return p.Meta(), nil
}

// ListUserProviders retrieves all providers for the given user.
func (s *Service) ListUserProviders(user *happydns.User) ([]*happydns.ProviderMeta, error) {
	items, err := s.store.ListProviders(user)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("failed to list providers: %w", err),
			UserMessage: "Sorry, we are currently unable to list your providers. Please try again later.",
		}
	}

	metas := make([]*happydns.ProviderMeta, 0, len(items))
	for _, p := range items {
		metas = append(metas, &p.ProviderMeta)
	}

	return metas, nil
}

// UpdateProvider updates a provider using the provided update function.
func (s *Service) UpdateProvider(providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	provider, err := s.GetUserProvider(user, providerID)
	if err != nil {
		return err
	}

	updateFn(provider)

	if !provider.Id.Equals(providerID) {
		return happydns.ValidationError{Msg: "you cannot change the provider identifier"}
	}

	err = s.validator.Validate(provider)
	if err != nil {
		return happydns.ValidationError{Msg: fmt.Sprintf("unable to validate provider attributes: %s", err.Error())}
	}

	err = s.store.UpdateProvider(provider)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateProvider in UpdateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to update your provider. Please retry later.",
		}
	}

	return nil
}

// UpdateProviderFromMessage updates a provider from a ProviderMessage.
func (s *Service) UpdateProviderFromMessage(providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	newprovider, err := ParseProvider(p)
	if err != nil {
		return err
	}

	return s.UpdateProvider(providerID, user, func(provider *happydns.Provider) {
		provider.Type = newprovider.Type
		provider.Comment = newprovider.Comment
		provider.Provider = newprovider.Provider
	})
}

// DeleteProvider deletes a provider for the given user.
func (s *Service) DeleteProvider(user *happydns.User, providerID happydns.Identifier) error {
	// Verify ownership before deleting
	if _, err := s.getUserProvider(user, providerID); err != nil {
		return err
	}

	if err := s.store.DeleteProvider(providerID); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("failed to delete provider %s: %w", providerID.String(), err),
			UserMessage: "Sorry, we are currently unable to delete your provider. Please try again later.",
		}
	}

	return nil
}

// RestrictedService wraps a ProviderUsecase with configuration-based restrictions.
type RestrictedService struct {
	inner  happydns.ProviderUsecase
	config *happydns.Options
}

// NewRestrictedService creates a RestrictedService backed by the given configuration and storage.
func NewRestrictedService(cfg *happydns.Options, store ProviderStorage) *RestrictedService {
	return &RestrictedService{
		inner:  NewService(store, nil),
		config: cfg,
	}
}

// CreateProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) CreateProvider(user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	if s.config.DisableProviders {
		return nil, happydns.ForbiddenError{Msg: "cannot add provider as DisableProviders parameter is set."}
	}

	return s.inner.CreateProvider(user, msg)
}

// DeleteProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) DeleteProvider(user *happydns.User, providerID happydns.Identifier) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot delete provider as DisableProviders parameter is set."}
	}

	return s.inner.DeleteProvider(user, providerID)
}

// UpdateProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) UpdateProvider(providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.inner.UpdateProvider(providerID, user, updateFn)
}

// UpdateProviderFromMessage refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) UpdateProviderFromMessage(providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.inner.UpdateProviderFromMessage(providerID, user, p)
}

func (s *RestrictedService) CreateDomainOnProvider(provider *happydns.Provider, fqdn string) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot create domain on provider as DisableProviders parameter is set."}
	}

	return s.inner.CreateDomainOnProvider(provider, fqdn)
}

// Read-only operations delegate directly.

func (s *RestrictedService) GetUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	return s.inner.GetUserProvider(user, providerID)
}

func (s *RestrictedService) GetUserProviderMeta(user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	return s.inner.GetUserProviderMeta(user, providerID)
}

func (s *RestrictedService) ListUserProviders(user *happydns.User) ([]*happydns.ProviderMeta, error) {
	return s.inner.ListUserProviders(user)
}

func (s *RestrictedService) ListHostedDomains(provider *happydns.Provider) ([]string, error) {
	return s.inner.ListHostedDomains(provider)
}

func (s *RestrictedService) ListZoneCorrections(ctx context.Context, provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, int, error) {
	return s.inner.ListZoneCorrections(ctx, provider, domain, records)
}

func (s *RestrictedService) RetrieveZone(ctx context.Context, provider *happydns.Provider, name string) ([]happydns.Record, error) {
	return s.inner.RetrieveZone(ctx, provider, name)
}

func (s *RestrictedService) TestDomainExistence(provider *happydns.Provider, name string) error {
	return s.inner.TestDomainExistence(provider, name)
}
