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
	mu := m.ownerLock(kind + ":" + owner.String())
	mu.Lock()
	defer mu.Unlock()

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

// ownerLock returns the mutex of key, an owner and what is locked of theirs.
func (m *Manager) ownerLock(key string) *sync.Mutex {
	mu, _ := m.locks.LoadOrStore(key, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// unlock returns what seals and opens the secrets of safe: through the store
// of its kind, or else through a copy of its key, an entry of its keyring a
// KEK here unwraps. A safe keeping a secret opens it first, among the safes
// already unlocked, which may be nil.
func (m *Manager) unlock(ctx context.Context, safe *happydns.Safe, unlocked unlockedSafes) (UnlockedSafe, error) {
	// Only what its owner gives opens it: a copy of its key would make
	// that of no use.
	if store, ok := m.stores[safe.Kind].(Interactive); ok {
		if u, ok := m.unlocked.get(UnlockScopeFromContext(ctx), safe); ok {
			return u, nil
		}
		return nil, lockedError(safe, store)
	}
	if store, ok := m.stores[safe.Kind].(SecretKeeping); ok {
		secret, err := m.openSafeSecret(ctx, safe, unlocked)
		if err != nil {
			return nil, err
		}
		return store.UnlockWithSecret(ctx, safe, secret)
	}

	var errs []error
	if store, ok := m.stores[safe.Kind]; ok {
		u, err := store.Unlock(ctx, safe)
		if err == nil {
			return u, nil
		}
		errs = append(errs, err)
	} else {
		errs = append(errs, fmt.Errorf("safe %s needs the %q secret store, which is not configured", safe.Id.String(), safe.Kind))
	}

	// The store of its kind missing or failing, a copy of its key kept by
	// another store opens it all the same: that is what the copy is for.
	for _, w := range safe.Keyring {
		kek, ok := m.stores[w.KEK].(KEK)
		if !ok || w.KEK == safe.Kind {
			continue
		}
		dek, err := kek.Unwrap(ctx, safe, w)
		if err != nil {
			errs = append(errs, fmt.Errorf("through its copy under %q: %w", w.KEK, err))
			continue
		}
		return newAEADSafe(safe.Id, dek)
	}

	return nil, errors.Join(errs...)
}

// hasCopy tells whether a store here opens safe through a copy of its key.
func (m *Manager) hasCopy(safe *happydns.Safe) bool {
	return slices.ContainsFunc(safe.Keyring, func(w happydns.WrappedKeyset) bool {
		_, ok := m.stores[w.KEK].(KEK)
		return ok && w.KEK != safe.Kind
	})
}

// checkSafeKinds returns an error when safes are stored whose kind has no
// store here, not enabled or not configured, and that no copy of their key
// opens: the secrets they hold would fail one at a time. A safe record that
// does not decode is left to the reports.
func (m *Manager) checkSafeKinds() error {
	if m == nil {
		return errNoManager
	}
	if m.safes == nil {
		return nil
	}

	missing := map[string]int{}
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		if _, ok := m.stores[safe.Kind]; !ok && !m.hasCopy(safe) {
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

// CheckSafe returns an error when this instance could not keep safe, one
// coming from elsewhere such as a backup: stored, it would keep happyDomain
// from starting, or only hold values that never open here.
//
// It asks for what startup asks: a store of its kind, or a copy of its key a
// store here opens. Beyond that, only the instance keyset, which is held here,
// is tried: the services of the other stores may not answer at the time of a
// restore, and the safe only waits for them.
func (m *Manager) CheckSafe(ctx context.Context, safe *happydns.Safe) error {
	if m == nil {
		return errNoManager
	}
	if m.safes == nil {
		return fmt.Errorf("safe %s cannot be kept: no safe storage is configured", safe.Id.String())
	}

	store, ok := m.stores[safe.Kind]
	if !ok {
		if m.hasCopy(safe) {
			return nil
		}
		err := fmt.Errorf("safe %s needs the %q secret store, which is not enabled or not configured", safe.Id.String(), safe.Kind)
		if why, ok := m.unavailable[safe.Kind]; ok {
			err = fmt.Errorf("%w: %w", err, why)
		}
		return err
	}

	if _, ok := store.(*instanceStore); !ok {
		return nil
	}
	_, err := m.unlock(ctx, safe, nil)
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

	// What they unlocked would no longer open anyway.
	m.LockOwner(owner)

	if m.defaults != nil {
		if err := m.defaults.DeleteDefaultKeystore(owner); err != nil && !errors.Is(err, happydns.ErrNotFound) {
			return fmt.Errorf("unable to forget the default keystore of %s: %w", owner.String(), err)
		}
	}
	return nil
}
