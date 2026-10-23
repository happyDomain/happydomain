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
	"slices"
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

// GuardedProviderBody is implemented by provider bodies that dial
// destinations they learn on their own, rather than the endpoints of their
// settings, which checkEndpoints vets beforehand: they are handed the guard
// to vet those destinations as they find them, and the default resolvers
// to learn them from.
type GuardedProviderBody interface {
	InstantiateGuardedProvider(guard *netguard.Guard, resolvers []string) (happydns.ProviderActuator, error)
}

// instantiator is the only place a provider's secrets are opened: right
// before happyDomain connects to it on its owner's behalf.
type instantiator struct {
	// guard decides which endpoints a provider may be pointed at. A nil guard
	// still refuses non-public destinations: see netguard.Guard.
	guard *netguard.Guard

	// resolvers are the -default-resolver ones, handed to the providers
	// learning on their own where to connect.
	resolvers []string

	secrets *secret.Manager
}

// instantiate checks where the provider points, opens its secrets in a copy,
// then instantiates it. Every dial made on a provider's behalf, the
// validator's included, starts here, so this is the one place the endpoint
// check has to be made. It comes first: several backends dial while being
// constructed, carrying the opened credentials to the endpoint.
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

	var instance happydns.ProviderActuator
	if guarded, ok := body.(GuardedProviderBody); ok {
		instance, err = guarded.InstantiateGuardedProvider(i.guard, i.resolvers)
	} else {
		instance, err = body.InstantiateProvider()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}
	return instance, nil
}

// seal seals the secrets of p before it is stored, into keystore: moving
// there those sealed elsewhere, or, empty, where secret.Destination tells.
// stored, the stored version, tells the safes it used, whose secrets an
// update may all replace, and the references it kept, which a client sends
// back as shown.
func (i instantiator) seal(ctx context.Context, p *happydns.Provider, keystore string, stored storedSecrets) error {
	err := i.secrets.SealObjectInto(ctx, SecretContext(p), p.Provider, secret.Destination{Keystore: keystore, Current: stored.safes, References: stored.references})
	if errors.Is(err, secret.ErrUnknownKeystore) || secret.IsReferenceError(err) {
		return happydns.ValidationError{Msg: err.Error()}
	}
	if errors.Is(err, happydns.ErrKeystoreLocked) {
		// The owner unlocks it, and tries again.
		return err
	}
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to seal provider credentials: %w", err),
			UserMessage: "Sorry, we are currently unable to store your provider credentials. Please try again later.",
		}
	}
	return nil
}

// storedSecrets tells where the secrets of the stored version of a provider
// are, before an update replaces them.
type storedSecrets struct {
	safes      []happydns.Identifier
	references map[string]string
}

// storedSecretsOf reads, from p as stored, the safes and references seal needs
// once an update has replaced them.
func (i instantiator) storedSecretsOf(ctx context.Context, p *happydns.Provider) (storedSecrets, error) {
	safes, err := secret.SafesOf(p.Provider)
	if err != nil {
		return storedSecrets{}, err
	}
	references, err := i.secrets.ReferencesOf(ctx, SecretContext(p), p.Provider)
	if err != nil {
		return storedSecrets{}, err
	}
	return storedSecrets{safes: safes, references: references}, nil
}

// ShowReferences replaces each secret of p, as stored, that a keystore of
// references keeps by that reference, which is not secret: see
// secret.Manager.ShowReferences. Only on a provider answering its owner,
// before the other secrets are withheld.
func (s *Service) ShowReferences(ctx context.Context, p *happydns.Provider) error {
	return s.secrets.ShowReferences(ctx, SecretContext(p), p.Provider)
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
// Entries are never removed: deleting one could hand a second mutex to a
// goroutine still waiting on the first, and one mutex per provider is cheap.
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

// OwnerObjects lists the providers of owner holding sealed secrets, with the
// safes each uses.
func (s *Service) OwnerObjects(ctx context.Context, owner happydns.Identifier) ([]secret.ObjectSafes, error) {
	msgs, err := s.store.ListProviders(&happydns.User{Id: owner})
	if err != nil {
		return nil, err
	}

	var objects []secret.ObjectSafes
	for _, meta := range msgs {
		msg, err := s.store.GetProvider(meta.Id)
		if err != nil {
			continue
		}
		p, err := ParseProvider(msg)
		if err != nil {
			continue
		}
		ids, err := secret.SafesOf(p.Provider)
		if err != nil || len(ids) == 0 {
			continue
		}
		objects = append(objects, secret.ObjectSafes{ObjectType: SecretObjectType, ObjectId: p.Id.String(), Label: p.Comment, Safes: ids})
	}
	return objects, nil
}

// MoveOwnerSecrets moves into the keystore to the secrets of the providers of
// owner sealed in the safe from. A provider that fails is reported and
// skipped.
func (s *Service) MoveOwnerSecrets(ctx context.Context, owner, from happydns.Identifier, to string) (secret.ResealReport, error) {
	report := secret.ResealReport{ObjectType: SecretObjectType}

	objects, err := s.OwnerObjects(ctx, owner)
	if err != nil {
		return report, err
	}
	for _, o := range objects {
		if !slices.ContainsFunc(o.Safes, from.Equals) {
			continue
		}
		report.Processed++
		id, err := happydns.NewIdentifierFromString(o.ObjectId)
		if err == nil {
			err = s.moveProvider(ctx, id, owner, to)
		}
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("provider %s: %s", o.ObjectId, err.Error()))
			continue
		}
		report.Changed++
	}
	return report, nil
}

func (s *Service) moveProvider(ctx context.Context, id, owner happydns.Identifier, to string) error {
	defer lockProvider(id)()

	msg, err := s.store.GetProvider(id)
	if err != nil {
		return err
	}
	p, err := ParseProvider(msg)
	if err != nil {
		return err
	}
	if !p.Owner.Equals(owner) {
		return errors.New("provider of another owner")
	}

	if err := s.secrets.SealObjectInto(ctx, SecretContext(p), p.Provider, secret.Destination{Keystore: to}); err != nil {
		return err
	}
	return s.store.UpdateProvider(p)
}
