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
	"encoding/binary"
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

// wrapAssociatedData binds a wrapped keyset to its safe and owner, so that it
// cannot be moved to another safe. Same length-prefixed layout as
// AssociatedData.
//
// This is a persisted format: changing it makes every safe unreadable.
func wrapAssociatedData(safe *happydns.Safe) []byte {
	var ad []byte
	for _, p := range [][]byte{[]byte(wrapVersion), safe.Id, safe.Owner} {
		ad = binary.BigEndian.AppendUint32(ad, uint32(len(p)))
		ad = append(ad, p...)
	}
	return ad
}

// Wrap encrypts dek, the key of safe, under the instance keyset.
func (k *InstanceKey) Wrap(safe *happydns.Safe, dek *keyset.Handle) (happydns.WrappedKeyset, error) {
	var buf bytes.Buffer
	if err := dek.WriteWithAssociatedData(keyset.NewBinaryWriter(&buf), k.aead, wrapAssociatedData(safe)); err != nil {
		return happydns.WrappedKeyset{}, fmt.Errorf("unable to wrap the key of safe %s: %w", safe.Id.String(), err)
	}

	return happydns.WrappedKeyset{KEK: KEKInstance, Blob: buf.Bytes()}, nil
}

// Unwrap decrypts the key of safe from its instance keyring entry.
func (k *InstanceKey) Unwrap(safe *happydns.Safe) (*keyset.Handle, error) {
	for _, w := range safe.Keyring {
		if w.KEK != KEKInstance {
			continue
		}

		h, err := keyset.ReadWithAssociatedData(keyset.NewBinaryReader(bytes.NewReader(w.Blob)), k.aead, wrapAssociatedData(safe))
		if err != nil {
			return nil, fmt.Errorf("unable to unwrap the key of safe %s: %w", safe.Id.String(), err)
		}
		return h, nil
	}

	return nil, fmt.Errorf("safe %s has no key wrapped by the instance keyset", safe.Id.String())
}

// safeRegistry finds, creates and opens the safes.
type safeRegistry struct {
	store SafeStorage
	key   *InstanceKey

	// locks holds one mutex per owner, so that two requests of the same
	// user do not both create their instance safe.
	locks sync.Map
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

	if err := r.store.CreateSafe(safe); err != nil {
		return nil, fmt.Errorf("unable to create the safe of %s: %w", owner.String(), err)
	}
	return safe, nil
}

// aead returns the primitive sealing and opening the values of safe.
func (r *safeRegistry) aead(safe *happydns.Safe) (tink.AEAD, error) {
	if safe.Kind != KindInstance {
		return nil, fmt.Errorf("safe %s is of unsupported kind %q", safe.Id.String(), safe.Kind)
	}
	if r.key == nil {
		return nil, fmt.Errorf("safe %s needs the instance keyset, which is not configured", safe.Id.String())
	}

	dek, err := r.key.Unwrap(safe)
	if err != nil {
		return nil, err
	}
	return aead.New(dek)
}
