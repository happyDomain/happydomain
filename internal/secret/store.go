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
	"maps"
	"slices"

	"git.happydns.org/happyDomain/model"
)

// Store protects the secrets of every safe of one kind: the instance keyset,
// a passphrase, an external secret manager...
//
// To add a kind of safe, implement Store in a file of its own and list its
// factory in builtinStores.
type Store interface {
	// Unlock returns what seals and opens the secrets of safe. It is
	// called once per safe and object, so that what is costly, such as
	// unwrapping a key, is not done for each secret.
	Unlock(ctx context.Context, safe *happydns.Safe) (UnlockedSafe, error)
}

// UnlockedSafe seals and opens the secrets of one safe.
type UnlockedSafe interface {
	// Seal returns the payload the sealed value of clear carries. sc tells
	// where the secret belongs: a store encrypting it binds it there.
	Seal(ctx context.Context, sc SecretContext, clear []byte) ([]byte, error)

	// Open returns the value payload, sealed for sc, stands for.
	Open(ctx context.Context, sc SecretContext, payload []byte) ([]byte, error)
}

// Provisioner is a Store creating the safe of an owner the first time a
// secret of theirs is sealed, without them doing anything. Only such a store
// can be the policy.
type Provisioner interface {
	Store

	// Provision fills the keyring of safe, new and not stored yet.
	Provision(ctx context.Context, safe *happydns.Safe) error
}

// StartupChecker is a Store that checks, before happyDomain starts, that what
// it holds can still be opened: refusing to start beats failing one
// credential at a time.
type StartupChecker interface {
	Store

	// CheckStartup returns an error when happyDomain must not start with
	// this store as configured.
	CheckStartup(ctx context.Context) error
}

// Rewrapper is a Store whose key changes, as the instance keyset rotates: it
// wraps again the keyring entries it made under an older key.
type Rewrapper interface {
	Store

	// Rewrap wraps w again, an entry of safe made by this store, under its
	// current key, and reports whether w changed.
	Rewrap(ctx context.Context, safe *happydns.Safe, w *happydns.WrappedKeyset) (bool, error)
}

// KeyReporter is a Store that lists its keys, and tells which of them made
// each of its keyring entries: a key no entry needs any more can be removed.
type KeyReporter interface {
	Store

	// Keys lists the keys of the store, without their material.
	Keys() []KeyInfo

	// WrappingKey returns the identifier of the key that made w, an entry
	// of this store.
	WrappingKey(w happydns.WrappedKeyset) (uint32, error)
}

// StoreFactory returns the store of a kind for cfg, or ErrStoreUnavailable
// when cfg lacks what it needs, wrapped with how to configure it.
type StoreFactory func(cfg Config) (Store, error)

// ErrStoreUnavailable is returned by a StoreFactory that is not configured:
// the safes of its kind do not open, nor are new ones created.
var ErrStoreUnavailable = errors.New("secret store not configured")

// builtinStores holds the factory of each kind of safe happyDomain knows. A
// kind is persisted in each safe: it cannot change once safes of that kind
// exist. It is never empty nor plaintext, which names no store.
var builtinStores = map[string]StoreFactory{
	KindInstance: newInstanceStore,
}

// KnownStores returns the kinds of safe happyDomain knows, sorted.
func KnownStores() []string {
	return slices.Sorted(maps.Keys(builtinStores))
}

// DefaultStores returns the stores enabled when the administrator lists none:
// plaintext, as happyDomain always did, and every kind of safe. A kind not
// configured is enabled all the same: it is offered to nobody, see
// ErrStoreUnavailable.
func DefaultStores() []string {
	return append([]string{string(PolicyPlaintext)}, KnownStores()...)
}

// storeFactories returns the factories cfg builds its stores from: the
// builtin ones, unless a test gave others.
func (cfg Config) storeFactories() map[string]StoreFactory {
	if cfg.factories != nil {
		return cfg.factories
	}
	return builtinStores
}
