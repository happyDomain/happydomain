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

package secret

import (
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"slices"
	"sync"
	"time"

	"git.happydns.org/happyDomain/model"
)

// instanceOwner owns the secrets of the instance itself. It is persisted, in
// safes and in the associated data of what they seal: it cannot change.
const instanceOwner = "instance"

// InstanceOwner returns the owner of the secrets of the instance itself, such
// as the checker options an administrator sets for every user. Each call
// returns a new copy, which the caller may change.
//
// Generated user identifiers are 16 random bytes and do not collide with it,
// but an administrator or a restore can store a user under any identifier:
// code acting on the safes of a user checks IsInstanceOwner first.
func InstanceOwner() happydns.Identifier {
	return happydns.Identifier(instanceOwner)
}

// IsInstanceOwner tells whether owner is the instance itself rather than a
// user.
func IsInstanceOwner(owner happydns.Identifier) bool {
	return string(owner) == instanceOwner
}

// ownerSafe returns the safe of kind owned by owner, having store create it
// on first use.
func (m *Manager) ownerSafe(ctx context.Context, owner happydns.Identifier, kind string, store Provisioner) (*happydns.Safe, error) {
	// One mutex per owner and kind, so that two requests of the same user
	// do not both create their safe.
	mu, _ := m.locks.LoadOrStore(kind+":"+owner.String(), &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	safe, err := m.safes.GetSafeByOwner(owner, kind)
	if err == nil {
		return safe, nil
	}
	if !errors.Is(err, happydns.ErrSafeNotFound) {
		return nil, err
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return nil, err
	}
	safe = &happydns.Safe{
		Id:        id,
		Owner:     owner,
		Kind:      kind,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Provision(ctx, safe); err != nil {
		return nil, err
	}

	if err := m.safes.CreateSafe(safe); err != nil {
		return nil, fmt.Errorf("unable to create the safe of %s: %w", owner.String(), err)
	}
	return safe, nil
}

// unlock returns what seals and opens the secrets of safe, through the store
// of its kind.
func (m *Manager) unlock(ctx context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	store, ok := m.stores[safe.Kind]
	if !ok {
		return nil, fmt.Errorf("safe %s needs the %q secret store, which is not configured", safe.Id.String(), safe.Kind)
	}
	return store.Unlock(ctx, safe)
}

// checkSafeKinds returns an error when safes are stored whose kind has no
// store here, not enabled or not configured: the secrets they hold would fail
// one at a time. A safe record that does not decode is left to the reports.
func (m *Manager) checkSafeKinds() error {
	if m == nil {
		return errNoManager
	}
	if m.safes == nil {
		return nil
	}

	missing := map[string]int{}
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		if _, ok := m.stores[safe.Kind]; !ok {
			missing[safe.Kind]++
		}
	}, nil)
	if err != nil {
		return fmt.Errorf("unable to list safes: %w", err)
	}

	var errs []error
	for _, kind := range slices.Sorted(maps.Keys(missing)) {
		err := fmt.Errorf("%d safes of kind %q are stored, but its secret store is not enabled or not configured", missing[kind], kind)
		if why, ok := m.unavailable[kind]; ok {
			err = fmt.Errorf("%w: %w", err, why)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CheckSafe returns an error when this instance cannot open safe, one coming
// from elsewhere such as a backup: stored, it would only hold values that
// never open here.
func (m *Manager) CheckSafe(ctx context.Context, safe *happydns.Safe) error {
	if m == nil {
		return errNoManager
	}
	if m.safes == nil {
		return fmt.Errorf("safe %s cannot be kept: no safe storage is configured", safe.Id.String())
	}
	_, err := m.unlock(ctx, safe)
	return err
}

// DeleteOwnerSafes deletes every safe owned by owner: whatever was sealed in
// them no longer opens, copies and backups included.
func (m *Manager) DeleteOwnerSafes(owner happydns.Identifier) error {
	if m == nil || m.safes == nil {
		return errNoManager
	}
	if IsInstanceOwner(owner) {
		return ErrInstanceOwner
	}

	// A record that does not decode tells no owner: it is left as it is,
	// rather than keep every account from being deleted.
	var ids []happydns.Identifier
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		if safe.Owner.Equals(owner) {
			ids = append(ids, safe.Id)
		}
	}, func(key string, err error) {
		log.Printf("secret: safe record %q does not decode, left as it is while deleting the safes of %s: %s", key, owner.String(), err)
	})
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := m.safes.DeleteSafe(id); err != nil {
			return fmt.Errorf("unable to delete safe %s: %w", id.String(), err)
		}
	}
	return nil
}
