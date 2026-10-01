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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/internal/forms"
	"git.happydns.org/happyDomain/internal/netguard"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// Service handles CRUD operations on DNS providers, with ownership enforcement.
type Service struct {
	instantiator
	store     ProviderStorage
	validator ProviderValidator
}

func NewService(store ProviderStorage, validator ProviderValidator, guard *netguard.Guard, secrets *secret.Manager) *Service {
	if validator == nil {
		validator = NewValidator(guard, secrets)
	}
	return &Service{
		instantiator: instantiator{guard: guard, secrets: secrets},
		store:        store,
		validator:    validator,
	}
}

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

// instantiate checks where the provider points, then instantiates it, wrapping
// errors consistently.
//
// Every dial happyDomain makes on a provider's behalf starts here or in
// DefaultProviderValidator, so these two are the only places the endpoint check
// has to be made: creating a provider, editing it, and every later apply all
// funnel through one of them.
func (s *Service) CreateProvider(ctx context.Context, user *happydns.User, msg *happydns.ProviderMessage) (*happydns.Provider, error) {
	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse provider: %w", err)
	}

	if err := checkIncoming(provider); err != nil {
		return nil, err
	}

	provider.Owner = user.Id

	// Chosen here rather than by the storage, whatever the client sent: the
	// secrets are sealed bound to this identifier before being stored.
	provider.Id, err = happydns.NewRandomIdentifier()
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to generate provider identifier: %w", err),
			UserMessage: "Sorry, we are currently unable to create the given provider. Please try again later.",
		}
	}

	// Nothing is stored yet, so a secret still holding happydns.RedactedSecret
	// has no value behind it. Clear it rather than let the placeholder reach a
	// provider API as if it were a credential: Validate below dials for real,
	// and `required` should report the field as empty.
	forms.MergeSecrets(nil, provider.Provider)

	if err := s.validator.Validate(ctx, provider); err != nil {
		return nil, fmt.Errorf("invalid provider: %w", err)
	}

	if err := s.seal(ctx, provider); err != nil {
		return nil, err
	}

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
func (s *Service) GetUserProvider(_ context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error) {
	p, err := s.getUserProvider(user, providerID)
	if err != nil {
		return nil, err
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
//
// The provider is written only if it is still what updateFn was given: what
// updateFn carries forward, sealed values included, is never written over
// what another writer, such as a reseal, stored in between. Such an update
// fails with a happydns.ConflictError; retrying reads the provider again.
//
// This only guards against a write landing while the update is handled. A
// client submitting a form read before someone else's change is not detected:
// the update is applied to what is stored when writing.
func (s *Service) UpdateProvider(ctx context.Context, providerID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Provider)) error {
	// What updating the stored provider failed with, told apart from what
	// the storage fails with.
	var updateErr error
	err := s.store.ReplaceProvider(providerID, func(msg *happydns.ProviderMessage) (*happydns.Provider, error) {
		var provider *happydns.Provider
		provider, updateErr = s.updated(ctx, providerID, user, msg, updateFn)
		return provider, updateErr
	})

	switch {
	case updateErr != nil:
		return updateErr
	case errors.Is(err, happydns.ErrChangedMeanwhile):
		return happydns.ConflictError{
			Msg: "This provider was written by another operation at the same moment. Please try again.",
			Err: err,
		}
	case errors.Is(err, happydns.ErrProviderNotFound):
		return err
	case err != nil:
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateProvider in UpdateProvider: %w", err),
			UserMessage: "Sorry, we are currently unable to update your provider. Please retry later.",
		}
	}

	return nil
}

// updated returns msg, the stored provider providerID, as updateFn changes it,
// validated and sealed.
func (s *Service) updated(ctx context.Context, providerID happydns.Identifier, user *happydns.User, msg *happydns.ProviderMessage, updateFn func(*happydns.Provider)) (*happydns.Provider, error) {
	if !user.Id.Equals(msg.Owner) {
		return nil, happydns.ErrProviderNotFound
	}

	provider, err := ParseProvider(msg)
	if err != nil {
		return nil, err
	}

	updateFn(provider)

	if !provider.Id.Equals(providerID) {
		return nil, happydns.ValidationError{Msg: "you cannot change the provider identifier"}
	}
	if !provider.Owner.Equals(user.Id) {
		// Secrets are bound to their owner: those carried forward would no
		// longer open.
		return nil, happydns.ValidationError{Msg: "you cannot change the provider owner"}
	}

	if err := s.validator.Validate(ctx, provider); err != nil {
		// The validator opens the stored credentials first: one that does
		// not open is not a fault of the attributes.
		if uerr := secret.UserError(err, "provider"); uerr != nil {
			return nil, uerr
		}
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to validate provider attributes: %s", err.Error())}
	}

	if err := s.seal(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// UpdateProviderFromMessage updates a provider from a ProviderMessage.
func (s *Service) UpdateProviderFromMessage(ctx context.Context, providerID happydns.Identifier, user *happydns.User, p *happydns.ProviderMessage) error {
	newprovider, err := ParseProvider(p)
	if err != nil {
		return err
	}

	// Before merging: the stored values carried forward are sealed, and
	// legitimately so.
	if err := checkIncoming(newprovider); err != nil {
		return err
	}

	return s.UpdateProvider(ctx, providerID, user, func(provider *happydns.Provider) {
		// provider is what is stored; a secret field still holding
		// happydns.RedactedSecret means the client is echoing back what the
		// user API withheld from it, so carry the stored value forward instead
		// of writing the placeholder. Skipped when the type changed: the two
		// bodies are then unrelated structs.
		//
		// This runs before UpdateProvider validates, which matters: the
		// validator dials the provider with these credentials.
		if provider.Type == newprovider.Type {
			forms.MergeSecrets(provider.Provider, newprovider.Provider)
		} else {
			forms.MergeSecrets(nil, newprovider.Provider)
		}

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
func NewRestrictedService(cfg *happydns.Options, store ProviderStorage, guard *netguard.Guard, secrets *secret.Manager) *RestrictedService {
	return &RestrictedService{
		inner:  NewService(store, nil, guard, secrets),
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
