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
	if err != nil {
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
	if err := i.secrets.SealObject(ctx, SecretContext(p), p.Provider); err != nil {
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
