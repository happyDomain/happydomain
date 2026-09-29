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
	"sync"

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

// providerLocks serializes the read-modify-write of one provider across every
// Service of the process: the user API and the admin API each have their own,
// and a reseal must not write back a body read before a user's update.
var providerLocks sync.Map

// lockProvider locks the provider id and returns the function unlocking it.
func lockProvider(id happydns.Identifier) func() {
	mu, _ := providerLocks.LoadOrStore(id.String(), &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}

// InspectSecrets tells how the secrets of every provider are stored.
func (s *Service) InspectSecrets(ctx context.Context) (secret.Counts, error) {
	var counts secret.Counts

	iter, err := s.store.ListAllProviders()
	if err != nil {
		return counts, err
	}
	defer iter.Close()

	// One provider that cannot be looked at must not hide the others.
	for iter.NextWithError() {
		if iter.Err() != nil {
			counts.Undecodable++
			continue
		}
		p, err := ParseProvider(iter.Item())
		if err != nil {
			counts.Undecodable++
			continue
		}
		if err := s.secrets.Inspect(ctx, SecretContext(p), p.Provider, &counts); err != nil {
			counts.Undecodable++
		}
	}

	return counts, iter.Err()
}

// ResealSecrets stores the secrets of every provider the way the current
// policy stores new ones. A provider that fails is reported and skipped; run
// it again to resume.
func (s *Service) ResealSecrets(ctx context.Context) (secret.ResealReport, error) {
	report := secret.ResealReport{ObjectType: SecretObjectType}

	iter, err := s.store.ListAllProviders()
	if err != nil {
		return report, err
	}

	// Collected first: writing while iterating is not safe on every storage.
	// A record that does not decode is reported, and the others resealed.
	var ids []happydns.Identifier
	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			report.Processed++
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("record %s: %s", iter.Key(), err.Error()))
			continue
		}
		ids = append(ids, iter.Item().Id)
	}
	err = iter.Err()
	iter.Close()
	if err != nil {
		return report, err
	}

	for _, id := range ids {
		report.Processed++

		changed, err := s.resealProvider(ctx, id)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("provider %s: %s", id.String(), err.Error()))
			continue
		}
		if changed {
			report.Changed++
		}
	}

	return report, nil
}

func (s *Service) resealProvider(ctx context.Context, id happydns.Identifier) (bool, error) {
	defer lockProvider(id)()

	msg, err := s.store.GetProvider(id)
	if errors.Is(err, happydns.ErrProviderNotFound) {
		// Deleted since it was listed.
		return false, nil
	}
	if err != nil {
		return false, err
	}

	p, err := ParseProvider(msg)
	if err != nil {
		return false, err
	}

	changed, err := s.secrets.ResealObject(ctx, SecretContext(p), p.Provider)
	if err != nil || !changed {
		return false, err
	}

	return true, s.store.UpdateProvider(p)
}
