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
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindInstance is a safe whose key is wrapped by the instance keyset.
	KindInstance = "instance"

	// KEKInstance names the instance keyset in a keyring.
	KEKInstance = "instance"

	wrapVersion = "hds:1:safe"
)

// scanSafes calls visit on every safe store lists that decodes, until visit
// returns false, and returns the keys of the records that do not decode.
//
// Such a record is corrupted, since a version refuses a database written by a
// later one: it may be any safe, of any owner and kind. What that means is
// the caller's decision, to be made explicitly: it depends on what the caller
// guarantees.
func scanSafes(store SafeStorage, visit func(*happydns.Safe) bool) (damaged []string, err error) {
	iter, err := store.ListAllSafes()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.NextWithError() {
		if iter.Err() != nil {
			damaged = append(damaged, iter.Key())
			continue
		}
		if !visit(iter.Item()) {
			break
		}
	}
	return damaged, iter.Err()
}

// wrapAssociatedData binds a wrapped keyset to its safe and owner, so that it
// cannot be moved to another safe.
//
// This is a persisted format: changing it makes every safe unreadable.
func wrapAssociatedData(safe *happydns.Safe) []byte {
	return lengthPrefixed([]byte(wrapVersion), safe.Id, safe.Owner)
}

// Wrap encrypts dek, the key of safe, under the instance keyset.
func (k *InstanceKey) Wrap(safe *happydns.Safe, dek *keyset.Handle) (happydns.WrappedKeyset, error) {
	var buf bytes.Buffer
	if err := dek.WriteWithAssociatedData(keyset.NewBinaryWriter(&buf), k.aead, wrapAssociatedData(safe)); err != nil {
		return happydns.WrappedKeyset{}, fmt.Errorf("unable to wrap the key of safe %s: %w", safe.Id.String(), err)
	}

	return happydns.WrappedKeyset{KEK: KEKInstance, Blob: buf.Bytes()}, nil
}

// instanceEntry returns the key of safe wrapped by the instance keyset.
func instanceEntry(safe *happydns.Safe) (happydns.WrappedKeyset, bool) {
	for _, w := range safe.Keyring {
		if w.KEK == KEKInstance {
			return w, true
		}
	}
	return happydns.WrappedKeyset{}, false
}

// Unwrap decrypts the key of safe from its instance keyring entry.
func (k *InstanceKey) Unwrap(safe *happydns.Safe) (*keyset.Handle, error) {
	w, ok := instanceEntry(safe)
	if !ok {
		return nil, fmt.Errorf("safe %s has no key wrapped by the instance keyset", safe.Id.String())
	}

	h, err := keyset.ReadWithAssociatedData(keyset.NewBinaryReader(bytes.NewReader(w.Blob)), k.aead, wrapAssociatedData(safe))
	if err != nil {
		return nil, fmt.Errorf("unable to unwrap the key of safe %s: %w", safe.Id.String(), err)
	}
	return h, nil
}

// safeRegistry finds, creates and opens the safes.
type safeRegistry struct {
	store SafeStorage
	key   *InstanceKey

	// owners, when set, is checked before a safe is created.
	owners OwnerStorage

	// locks holds one mutex per owner, so that two requests of the same
	// user do not both create their instance safe.
	locks sync.Map

	// primitives holds the primitive of each safe met, by safe identifier,
	// so that its key is unwrapped once and not on every seal or open.
	primitives sync.Map
}

// cachedPrimitive is the primitive of a safe, valid as long as the safe
// carries the same owner and wrapped key.
type cachedPrimitive struct {
	owner   happydns.Identifier
	wrapped []byte
	aead    tink.AEAD
}

func newSafeRegistry(store SafeStorage, key *InstanceKey) *safeRegistry {
	return &safeRegistry{store: store, key: key}
}

// instanceSafe returns the instance safe of owner, creating it on first use.
func (r *safeRegistry) instanceSafe(owner happydns.Identifier) (*happydns.Safe, error) {
	if r.key == nil {
		return nil, errors.New("no instance keyset configured")
	}

	mu, _ := r.locks.LoadOrStore(owner.String(), &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	safe, err := r.store.GetSafeByOwner(owner, KindInstance)
	if err == nil {
		return safe, nil
	}
	if !errors.Is(err, happydns.ErrSafeNotFound) {
		return nil, err
	}

	if err := r.checkOwner(owner); err != nil {
		return nil, err
	}

	id, err := happydns.NewRandomIdentifier()
	if err != nil {
		return nil, err
	}
	safe = &happydns.Safe{
		Id:        id,
		Owner:     owner,
		Kind:      KindInstance,
		CreatedAt: time.Now().UTC(),
	}

	dek, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, err
	}
	w, err := r.key.Wrap(safe, dek)
	if err != nil {
		return nil, err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}

	if err := r.store.CreateSafe(safe); errors.Is(err, happydns.ErrAlreadyExists) {
		// Another process created it since the lookup: use theirs.
		return r.store.GetSafeByOwner(owner, KindInstance)
	} else if err != nil {
		return nil, fmt.Errorf("unable to create the safe of %s: %w", owner.String(), err)
	}
	return safe, nil
}

// checkOwner refuses owner unless it is an existing user.
//
// It narrows, without closing, the window in which a user deleted while one
// of their objects is being sealed gets a safe: such a safe is left to
// TidySafes.
func (r *safeRegistry) checkOwner(owner happydns.Identifier) error {
	if r.owners == nil {
		return nil
	}
	_, err := r.owners.GetUser(owner)
	if errors.Is(err, happydns.ErrUserNotFound) {
		return fmt.Errorf("%w: %s", ErrUnknownOwner, owner.String())
	}
	if err != nil {
		return fmt.Errorf("unable to check the owner %s: %w", owner.String(), err)
	}
	return nil
}

// aead returns the primitive sealing and opening the values of safe.
//
// safe must come from the storage: the cache is only checked against what
// it carries, so a deleted safe is refused by the lookup, not here.
func (r *safeRegistry) aead(safe *happydns.Safe) (tink.AEAD, error) {
	if safe.Kind != KindInstance {
		return nil, fmt.Errorf("safe %s is of unsupported kind %q", safe.Id.String(), safe.Kind)
	}
	if r.key == nil {
		return nil, fmt.Errorf("safe %s needs the instance keyset, which is not configured", safe.Id.String())
	}

	w, _ := instanceEntry(safe)
	if c, ok := r.primitives.Load(safe.Id.String()); ok {
		c := c.(*cachedPrimitive)
		// The wrapped key is bound to its owner: a changed owner has to go
		// through Unwrap, and fail there.
		if c.owner.Equals(safe.Owner) && bytes.Equal(c.wrapped, w.Blob) {
			return c.aead, nil
		}
	}

	dek, err := r.key.Unwrap(safe)
	if err != nil {
		return nil, err
	}
	p, err := aead.New(dek)
	if err != nil {
		return nil, err
	}

	r.primitives.Store(safe.Id.String(), &cachedPrimitive{owner: safe.Owner, wrapped: w.Blob, aead: p})
	return p, nil
}

// forget drops what is cached about the safe id, once deleted.
func (r *safeRegistry) forget(id happydns.Identifier) {
	r.primitives.Delete(id.String())
}
