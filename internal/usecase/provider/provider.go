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
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// Service handles CRUD operations on DNS providers, with ownership enforcement.
type Service struct {
	cfg       *happydns.Options
	secrets   *secret.Manager
	store     ProviderStorage
	validator ProviderValidator
}

// NewService creates a new provider Service. If validator is nil,
// the DefaultProviderValidator is used.
func NewService(cfg *happydns.Options, secrets *secret.Manager, store ProviderStorage, validator ProviderValidator) *Service {
	if validator == nil {
		validator = &DefaultProviderValidator{}
	}
	return &Service{
		cfg:       cfg,
		secrets:   secrets,
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

// resolveSecretMethod determines which secret method to use for the given user.
func (s *Service) resolveSecretMethod(user *happydns.User) string {
	if s.cfg != nil && !s.cfg.DisableUserSecretMethod {
		if m := user.Settings.SecretMethod; m != "" {
			return m
		}
	}
	if s.cfg != nil && s.cfg.SecretMethod != "" {
		return s.cfg.SecretMethod
	}
	return ""
}

// openProviderSecret decrypts a provider's raw JSON if it's a SecretEnvelope,
// otherwise returns it unchanged (legacy plaintext).
func (s *Service) openProviderSecret(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	if s.secrets == nil {
		return raw, nil
	}
	if envelope, ok := secret.TryParseEnvelope(raw); ok {
		return s.secrets.Open(ctx, envelope)
	}
	return raw, nil
}

// sealProviderMessage encrypts the Provider field of a ProviderMessage.
func (s *Service) sealProviderMessage(ctx context.Context, user *happydns.User, msg *happydns.ProviderMessage) error {
	if s.secrets == nil {
		return nil
	}
	method := s.resolveSecretMethod(user)
	envelope, err := s.secrets.Seal(ctx, method, user.Id, msg.Id, msg.Provider)
	if err != nil {
		return fmt.Errorf("failed to seal provider secret: %w", err)
	}
	msg.Provider, err = json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal secret envelope: %w", err)
	}
	return nil
}

// CreateProvider creates a new provider for the given user.
func (s *Service) CreateProvider(ctx context.Context, user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse provider: %w", err)
	}

	if err := s.validator.Validate(provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	provider.Owner = user.Id

	if s.secrets != nil {
		// Seal and store as message
		sealedMsg, err := provider.ToMessage()
		if err != nil {
			return nil, fmt.Errorf("unable to serialize provider: %w", err)
		}
		sealedMsg.Owner = user.Id

		if err := s.sealProviderMessage(ctx, user, &sealedMsg); err != nil {
			return nil, err
		}

		if err := s.store.CreateProviderFromMessage(&sealedMsg); err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("failed to save provider: %w", err),
				UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
			}
		}
		provider.Id = sealedMsg.Id
	} else {
		if err := s.store.CreateProvider(provider); err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("failed to save provider: %w", err),
				UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
			}
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
func (s *Service) GetUserProvider(ctx context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	p, err := s.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	p.Provider, err = s.openProviderSecret(ctx, p.Provider)
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt provider secret: %w", err)
	}

	return ParseProvider(p)
}

// GetUserProviderMeta retrieves provider metadata for the given user.
func (s *Service) GetUserProviderMeta(_ context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	p, err := s.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
	}

	return p.Meta(), nil
}

// ListUserProviders retrieves all providers for the given user.
func (s *Service) ListUserProviders(_ context.Context, user *happydns.User) ([]*happydns.ProviderMeta, error) {
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
func (s *Service) UpdateProvider(ctx context.Context, providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	provider, err := s.GetUserProvider(ctx, user, providerID)
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

	if s.secrets != nil {
		sealedMsg, err := provider.ToMessage()
		if err != nil {
			return fmt.Errorf("unable to serialize provider: %w", err)
		}

		if err := s.sealProviderMessage(ctx, user, &sealedMsg); err != nil {
			return err
		}

		err = s.store.UpdateProviderFromRawMessage(&sealedMsg)
	} else {
		err = s.store.UpdateProvider(provider)
	}
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateProvider in UpdateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to update your provider. Please retry later.",
		}
	}

	return nil
}

// UpdateProviderFromMessage updates a provider from a ProviderMessage.
func (s *Service) UpdateProviderFromMessage(ctx context.Context, providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	newprovider, err := ParseProvider(p)
	if err != nil {
		return err
	}

	return s.UpdateProvider(ctx, providerID, user, func(provider *happydns.Provider) {
		provider.Type = newprovider.Type
		provider.Comment = newprovider.Comment
		provider.Provider = newprovider.Provider
	})
}

// DeleteProvider deletes a provider for the given user.
func (s *Service) DeleteProvider(_ context.Context, user *happydns.User, providerID happydns.Identifier) error {
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
func NewRestrictedService(cfg *happydns.Options, store ProviderStorage, secrets *secret.Manager) *RestrictedService {
	return &RestrictedService{
		inner:  NewService(cfg, secrets, store, nil),
		config: cfg,
	}
}

// CreateProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) CreateProvider(ctx context.Context, user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	if s.config.DisableProviders {
		return nil, happydns.ForbiddenError{Msg: "cannot add provider as DisableProviders parameter is set."}
	}

	return s.inner.CreateProvider(ctx, user, msg)
}

// DeleteProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) DeleteProvider(ctx context.Context, user *happydns.User, providerID happydns.Identifier) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot delete provider as DisableProviders parameter is set."}
	}

	return s.inner.DeleteProvider(ctx, user, providerID)
}

// UpdateProvider refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) UpdateProvider(ctx context.Context, providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.inner.UpdateProvider(ctx, providerID, user, updateFn)
}

// UpdateProviderFromMessage refuses the operation when DisableProviders is set, otherwise delegates to Service.
func (s *RestrictedService) UpdateProviderFromMessage(ctx context.Context, providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot update provider as DisableProviders parameter is set."}
	}

	return s.inner.UpdateProviderFromMessage(ctx, providerID, user, p)
}

func (s *RestrictedService) CreateDomainOnProvider(ctx context.Context, provider *happydns.Provider, fqdn string) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot create domain on provider as DisableProviders parameter is set."}
	}

	return s.inner.CreateDomainOnProvider(ctx, provider, fqdn)
}

// Read-only operations delegate directly.

func (s *RestrictedService) GetUserProvider(ctx context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	return s.inner.GetUserProvider(ctx, user, providerID)
}

func (s *RestrictedService) GetUserProviderMeta(ctx context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.ProviderMeta, error) {
	return s.inner.GetUserProviderMeta(ctx, user, providerID)
}

func (s *RestrictedService) ListUserProviders(ctx context.Context, user *happydns.User) ([]*happydns.ProviderMeta, error) {
	return s.inner.ListUserProviders(ctx, user)
}

func (s *RestrictedService) ListHostedDomains(ctx context.Context, provider *happydns.Provider) ([]string, error) {
	return s.inner.ListHostedDomains(ctx, provider)
}

func (s *RestrictedService) ListZoneCorrections(ctx context.Context, provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, int, error) {
	return s.inner.ListZoneCorrections(ctx, provider, domain, records)
}

func (s *RestrictedService) RetrieveZone(ctx context.Context, provider *happydns.Provider, name string) ([]happydns.Record, error) {
	return s.inner.RetrieveZone(ctx, provider, name)
}

func (s *RestrictedService) TestDomainExistence(ctx context.Context, provider *happydns.Provider, name string) error {
	return s.inner.TestDomainExistence(ctx, provider, name)
}
