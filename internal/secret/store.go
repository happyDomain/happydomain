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
	"maps"
	"slices"

	"github.com/tink-crypto/tink-go/v2/keyset"

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

// Creator is a Store whose safes their owner creates, giving what the store
// needs, such as a passphrase: a personal keystore. Several safes of its kind
// may belong to one owner, told apart by their name.
type Creator interface {
	Store

	// CreateForm describes the parameters Create takes, for the owner to
	// fill.
	CreateForm() []happydns.Field

	// Create fills the keyring of safe, new and not stored yet, from
	// params, a JSON object following CreateForm.
	Create(ctx context.Context, safe *happydns.Safe, params json.RawMessage) error
}

// Updater is a Creator whose owner may replace a credential the store keeps,
// such as a token, while the secrets of the safe stay as they are. A store
// whose update would change the key, such as a passphrase one, must not be
// an Updater: what its safe holds would no longer open.
type Updater interface {
	Creator

	// UpdateForm describes the parameters Update takes, for the owner to
	// fill.
	UpdateForm() []happydns.Field

	// Update changes the keyring of safe, as stored, from params, a JSON
	// object following UpdateForm. It leaves safe as it was on error.
	Update(ctx context.Context, safe *happydns.Safe, params json.RawMessage) error
}

// Interactive is a Store whose safes only open with what their owner gives,
// such as a passphrase. Once unlocked, a safe stays open for the session of
// the request that unlocked it, see Manager.UnlockKeystore; it never opens in
// the background, nor through a copy of its key under another store.
type Interactive interface {
	Store

	// UnlockForm describes what UnlockWith takes, for the owner to fill.
	UnlockForm() []happydns.Field

	// UnlockWith opens safe with material, a JSON object following
	// UnlockForm, or fails with ErrWrongUnlockMaterial.
	UnlockWith(ctx context.Context, safe *happydns.Safe, material json.RawMessage) (UnlockedSafe, error)
}

// PasswordBound is an Interactive store whose safes open with the login
// password of their owner: they are unlocked when the owner logs in, and
// bound to the new password when they change it. Its keystores are created
// from the login password alone, given as PasswordParam.
type PasswordBound interface {
	Interactive

	// Bind fills the keyring of safe, new and not stored yet, so that it
	// opens with password.
	Bind(ctx context.Context, safe *happydns.Safe, password string) error

	// UnlockWithPassword opens safe with the login password of its owner,
	// or fails with ErrWrongUnlockMaterial.
	UnlockWithPassword(ctx context.Context, safe *happydns.Safe, password string) (UnlockedSafe, error)

	// Rebind binds safe, opening with old, to new instead.
	Rebind(ctx context.Context, safe *happydns.Safe, old, new string) error
}

// PasswordParam names the login password in the parameters creating a
// keystore of a PasswordBound store.
const PasswordParam = "password"

// passwordForm is the creation form of a PasswordBound store.
func passwordForm() []happydns.Field {
	return []happydns.Field{{Id: PasswordParam, Type: "string", Label: "Login password", Secret: true, Required: true}}
}

// ErrWrongUnlockMaterial is returned by an Interactive store given what does
// not open the safe.
var ErrWrongUnlockMaterial = errors.New("this does not unlock the keystore")

// Referencing is a Store whose safes keep no secret: what the owner enters in
// a secret field is a reference, such as the key of the secret in a file of
// theirs, which Seal checks and keeps as the payload, and Open resolves to the
// secret, kept outside happyDomain.
//
// The Manager never moves a secret into such a safe, where it would be taken
// for a reference, and shows the references to their owner: they are not
// secret.
type Referencing interface {
	Store

	// ReferenceHint tells the owner what to enter in place of a secret.
	ReferenceHint() string

	// Reference returns the reference payload, made by Seal, carries, as
	// the owner entered it. It never resolves it.
	Reference(payload []byte) (string, error)
}

// Unattended is a Store whose safes open without their owner, as for a
// notification or a scheduled check run in the background. A store that is
// not an Unattended one is taken as needing its owner.
type Unattended interface {
	Store

	// OpensUnattended reports whether the safes of the store open without
	// their owner.
	OpensUnattended() bool
}

// opensUnattended tells whether the safes of store open without their owner.
func opensUnattended(store Store) bool {
	u, ok := store.(Unattended)
	return ok && u.OpensUnattended()
}

// KEK is a Store that wraps the key of a safe in a keyring entry named after
// its kind. Through such an entry, a copy, it opens any safe holding one,
// whatever the kind of the safe: the copy is kept for when the store of that
// kind is lost. The key copied is the Tink AEAD keyset sealing the values of
// the safe, as aeadSafe does.
type KEK interface {
	Store

	// Unwrap returns the key of safe from w, an entry made by this store.
	Unwrap(ctx context.Context, safe *happydns.Safe, w happydns.WrappedKeyset) (*keyset.Handle, error)
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

// KeyBorrower is a Store whose keyring entries are sealed by the keys of
// another, its lender, such as a token sealed under the instance keyset: the
// key status counts them under the keys of the lender, which they keep from
// being removed. It wraps them again itself, as a Rewrapper.
type KeyBorrower interface {
	Store

	// Lender names the store whose keys seal the entries of this one.
	Lender() string

	// WrappingKey returns the identifier of the key of the lender that
	// sealed w, an entry of this store.
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
	KindInstance:      newInstanceStore,
	KindPassphrase:    newPassphraseStore,
	KindLoginPassword: newLoginPasswordStore,
	KindFile:          newFileStore,
	KindKeyFile:       newKeyFileStore,
	KindProgram:       newProgramStore,
	KindVault:         newVaultStore,
	KindVaultUser:     newVaultUserStore,
}

// KnownStores returns the kinds of safe happyDomain knows, sorted.
func KnownStores() []string {
	return slices.Sorted(maps.Keys(builtinStores))
}

// optInStores lists the kinds of safe left out by default: they reach the
// files or programs of the server, and are only ever enabled on purpose.
var optInStores = []string{KindFile, KindKeyFile, KindProgram}

// DefaultStores returns the stores enabled when the administrator lists none:
// plaintext, as happyDomain always did, and every kind of safe but those of
// optInStores. A kind not configured is enabled all the same: it is offered
// to nobody, see ErrStoreUnavailable.
func DefaultStores() []string {
	stores := []string{string(PolicyPlaintext)}
	for _, kind := range KnownStores() {
		if !slices.Contains(optInStores, kind) {
			stores = append(stores, kind)
		}
	}
	return stores
}

// storeFactories returns the factories cfg builds its stores from: the
// builtin ones, unless a test gave others.
func (cfg Config) storeFactories() map[string]StoreFactory {
	if cfg.factories != nil {
		return cfg.factories
	}
	return builtinStores
}

// WithStores returns cfg knowing, besides its stores, those of factories, by
// kind, and enabling them when cfg lists the stores enabled. For tests
// outside this package: no other Manager knows of them.
func (cfg Config) WithStores(factories map[string]StoreFactory) Config {
	all := maps.Clone(cfg.storeFactories())
	maps.Copy(all, factories)
	cfg.factories = all

	if cfg.Stores != nil {
		// Appending must not reach the backing array of the caller.
		cfg.Stores = slices.Clone(cfg.Stores)
		for _, kind := range slices.Sorted(maps.Keys(factories)) {
			if !slices.Contains(cfg.Stores, kind) {
				cfg.Stores = append(cfg.Stores, kind)
			}
		}
	}
	return cfg
}
