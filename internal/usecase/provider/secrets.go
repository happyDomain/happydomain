// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/internal/netguard"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// SecretObjectType names providers in the context their secrets are bound to.
const SecretObjectType = "provider"

// SecretContext returns the context the secrets of p are bound to.
func SecretContext(p *happydns.Provider) secret.SecretContext {
	return secret.SecretContext{
		Owner:      p.Owner,
		ObjectType: SecretObjectType,
		ObjectId:   p.Id.String(),
	}
}

// instantiator is the only place a provider's secrets are opened: right
// before happyDomain connects to it on its owner's behalf.
type instantiator struct {
	// guard decides which endpoints a provider may be pointed at. A nil guard
	// still refuses non-public destinations: see netguard.Guard.
	guard *netguard.Guard

	secrets *secret.Manager
}

func (i instantiator) instantiate(ctx context.Context, p *happydns.Provider) (happydns.ProviderActuator, error) {
	if err := checkEndpoints(ctx, i.guard, p.Provider); err != nil {
		return nil, err
	}

	// Opened in a copy: p goes on carrying sealed values only.
	opened, err := i.secrets.OpenCopy(ctx, SecretContext(p), p.Provider)
	if uerr := secret.UserError(err, "provider"); uerr != nil {
		return nil, uerr
	} else if err != nil {
		return nil, fmt.Errorf("unable to open provider credentials: %w", err)
	}

	body, ok := opened.(happydns.ProviderBody)
	if !ok {
		return nil, fmt.Errorf("unable to open provider credentials: unexpected body %T", opened)
	}

	instance, err := body.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}
	return instance, nil
}

// seal seals the secrets of p before it is stored.
func (i instantiator) seal(ctx context.Context, p *happydns.Provider) error {
	err := i.secrets.SealObject(ctx, SecretContext(p), p.Provider)
	if uerr := secret.UserError(err, "provider"); uerr != nil {
		// A stored credential carried forward that does not open.
		return uerr
	} else if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to seal provider credentials: %w", err),
			UserMessage: "Sorry, we are currently unable to store your provider credentials. Please try again later.",
		}
	}
	return nil
}

// checkIncoming refuses a provider sent by a client holding a sealed value.
func checkIncoming(p *happydns.Provider) error {
	if err := secret.CheckIncoming(p.Provider); err != nil {
		if errors.Is(err, secret.ErrSealedFromClient) {
			return happydns.ValidationError{Msg: fmt.Sprintf("invalid provider: %s", err.Error())}
		}
		return err
	}
	return nil
}

// providerName names a provider in the reports.
func providerName(msg *happydns.ProviderMessage) string {
	return "provider " + msg.Id.String()
}

// InspectSecrets tells how the secrets of every provider are stored. A
// provider that cannot be looked at, such as one of a type this build does
// not know, is counted as undecodable.
func (s *Service) InspectSecrets(ctx context.Context) (secret.Counts, error) {
	iter, err := s.store.ListAllProviders()
	if err != nil {
		return secret.Counts{}, err
	}

	return secret.InspectAll(iter, providerName, func(msg *happydns.ProviderMessage, c *secret.Counts) error {
		p, err := ParseProvider(msg)
		if err != nil {
			return err
		}
		return s.secrets.Inspect(ctx, SecretContext(p), p.Provider, c)
	})
}

// ResealSecrets stores the secrets of every provider the way the current
// policy stores new ones. A provider that fails is reported and skipped; run
// it again to resume.
func (s *Service) ResealSecrets(ctx context.Context) (secret.ResealReport, error) {
	iter, err := s.store.ListAllProviders()
	if err != nil {
		return secret.ResealReport{ObjectType: SecretObjectType}, err
	}

	return secret.ResealAll(SecretObjectType, iter, providerName, func(msg *happydns.ProviderMessage) (bool, error) {
		return s.resealProvider(ctx, msg.Id)
	})
}

// resealProvider reseals the provider id as stored now. The write is
// conditional: whatever another writer did in between (a user's update, a
// restore, a deletion) wins, and the provider is left for the next run.
func (s *Service) resealProvider(ctx context.Context, id happydns.Identifier) (bool, error) {
	changed := false
	err := s.store.ReplaceProvider(id, func(msg *happydns.ProviderMessage) (*happydns.Provider, error) {
		p, err := ParseProvider(msg)
		if err != nil {
			return nil, err
		}

		c, err := s.secrets.ResealObject(ctx, SecretContext(p), p.Provider)
		if err != nil || !c {
			return nil, err
		}

		changed = true
		return p, nil
	})
	if errors.Is(err, happydns.ErrProviderNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return changed, nil
}
