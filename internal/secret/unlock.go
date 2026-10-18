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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

	"git.happydns.org/happyDomain/model"
)

// unlockCache keeps the safes of Interactive stores their owners unlocked,
// by scope, the session that unlocked them, then by safe. It is never
// persisted: a restart locks them all.
type unlockCache struct {
	mu     sync.Mutex
	scopes map[string]map[string]unlockedEntry
}

type unlockedEntry struct {
	owner    happydns.Identifier
	unlocked UnlockedSafe
}

func newUnlockCache() *unlockCache {
	return &unlockCache{scopes: map[string]map[string]unlockedEntry{}}
}

func (c *unlockCache) get(scope string, safe *happydns.Safe) (UnlockedSafe, bool) {
	if scope == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.scopes[scope][safe.Id.String()]
	// A safe id stored again for another owner, as by a restore, does not
	// open with what the previous owner unlocked.
	if !ok || !e.owner.Equals(safe.Owner) {
		return nil, false
	}
	return e.unlocked, true
}

func (c *unlockCache) put(scope string, safe *happydns.Safe, u UnlockedSafe) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scopes[scope] == nil {
		c.scopes[scope] = map[string]unlockedEntry{}
	}
	c.scopes[scope][safe.Id.String()] = unlockedEntry{owner: safe.Owner, unlocked: u}
}

// drop removes the entries keep refuses.
func (c *unlockCache) drop(keep func(scope, safeId string, e unlockedEntry) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for scope, safes := range c.scopes {
		for id, e := range safes {
			if !keep(scope, id, e) {
				delete(safes, id)
			}
		}
		if len(safes) == 0 {
			delete(c.scopes, scope)
		}
	}
}

// lockedError tells the owner of safe what unlocks it.
func lockedError(safe *happydns.Safe, store Interactive) error {
	ref := safe.Kind
	if isPersonal(safe) {
		ref = safe.Id.String()
	}
	return happydns.KeystoreLockedError{KeystoreLock: happydns.KeystoreLock{
		Ref:  ref,
		Kind: safe.Kind,
		Name: safe.Name,
		Form: store.UnlockForm(),
	}}
}

// interactiveSafe returns the safe behind ref, a keystore of owner, and its
// store, which has to be Interactive.
func (m *Manager) interactiveSafe(owner happydns.Identifier, ref string) (*happydns.Safe, Interactive, error) {
	safe, err := m.keystoreSafe(owner, ref)
	if err != nil {
		return nil, nil, err
	}
	if safe == nil {
		return nil, nil, fmt.Errorf("%w: keystore %q does not need unlocking", ErrInvalidKeystore, ref)
	}
	store, ok := m.stores[safe.Kind].(Interactive)
	if !ok {
		return nil, nil, fmt.Errorf("%w: keystore %q does not need unlocking", ErrInvalidKeystore, ref)
	}
	return safe, store, nil
}

// UnlockKeystore opens ref, a keystore of owner, with material, and keeps it
// open for the scope of ctx, the session of the request.
func (m *Manager) UnlockKeystore(ctx context.Context, owner happydns.Identifier, ref string, material json.RawMessage) error {
	if m == nil {
		return errNoManager
	}
	scope := UnlockScopeFromContext(ctx)
	if scope == "" {
		return errors.New("a keystore is unlocked for a session: this request has none")
	}
	safe, store, err := m.interactiveSafe(owner, ref)
	if err != nil {
		return err
	}
	u, err := store.UnlockWith(ctx, safe, material)
	if err != nil {
		return err
	}
	m.unlocked.put(scope, safe, u)
	return nil
}

// LockKeystore closes ref, a keystore of owner, for the scope of ctx.
func (m *Manager) LockKeystore(ctx context.Context, owner happydns.Identifier, ref string) error {
	if m == nil {
		return errNoManager
	}
	safe, _, err := m.interactiveSafe(owner, ref)
	if err != nil {
		return err
	}
	scope := UnlockScopeFromContext(ctx)
	m.unlocked.drop(func(s, id string, _ unlockedEntry) bool {
		return s != scope || id != safe.Id.String()
	})
	return nil
}

// LockScope closes every keystore scope, a session ending, unlocked.
func (m *Manager) LockScope(scope string) {
	if m == nil {
		return
	}
	m.unlocked.drop(func(s, _ string, _ unlockedEntry) bool { return s != scope })
}

// LockOwner closes every keystore of owner, in every session.
func (m *Manager) LockOwner(owner happydns.Identifier) {
	if m == nil {
		return
	}
	m.unlocked.drop(func(_, _ string, e unlockedEntry) bool { return !e.owner.Equals(owner) })
}

// PurgeScopes closes the keystores of the scopes alive refuses, sessions gone.
func (m *Manager) PurgeScopes(alive func(scope string) bool) {
	if m == nil {
		return
	}
	m.unlocked.drop(func(s, _ string, _ unlockedEntry) bool { return alive(s) })
}

// passwordSafes returns the safes of owner whose store is PasswordBound.
func (m *Manager) passwordSafes(owner happydns.Identifier) ([]*happydns.Safe, error) {
	if m.safes == nil {
		return nil, nil
	}
	safes, err := m.safes.ListOwnerSafes(owner)
	if err != nil {
		return nil, err
	}
	var bound []*happydns.Safe
	for _, safe := range safes {
		if _, ok := m.stores[safe.Kind].(PasswordBound); ok {
			bound = append(bound, safe)
		}
	}
	return bound, nil
}

// UnlockAtLogin opens, for the scope of ctx, the new session, the keystores
// of owner bound to their login password. One that does not open, as after a
// password reset, stays locked: its owner unlocks it with the password it is
// bound to.
func (m *Manager) UnlockAtLogin(ctx context.Context, owner happydns.Identifier, password string) {
	if m == nil {
		return
	}
	scope := UnlockScopeFromContext(ctx)
	if scope == "" {
		return
	}
	safes, err := m.passwordSafes(owner)
	if err != nil {
		log.Printf("secret: unable to list the safes of %s to unlock them at login: %s", owner.String(), err)
		return
	}
	for _, safe := range safes {
		u, err := m.stores[safe.Kind].(PasswordBound).UnlockWithPassword(ctx, safe, password)
		if err != nil {
			if !errors.Is(err, ErrWrongUnlockMaterial) {
				log.Printf("secret: unable to unlock safe %s at login: %s", safe.Id.String(), err)
			}
			continue
		}
		m.unlocked.put(scope, safe, u)
	}
}

// PasswordChanged binds the keystores of owner bound to old, their login
// password, to new. Their key does not change: what is unlocked stays so.
func (m *Manager) PasswordChanged(ctx context.Context, owner happydns.Identifier, old, new string) error {
	if m == nil {
		return errNoManager
	}
	safes, err := m.passwordSafes(owner)
	if err != nil {
		return err
	}
	var errs []error
	for _, safe := range safes {
		if err := m.stores[safe.Kind].(PasswordBound).Rebind(ctx, safe, old, new); err != nil {
			errs = append(errs, fmt.Errorf("safe %s: %w", safe.Id.String(), err))
			continue
		}
		if err := m.safes.UpdateSafe(safe); err != nil {
			errs = append(errs, fmt.Errorf("safe %s: %w", safe.Id.String(), err))
		}
	}
	return errors.Join(errs...)
}
