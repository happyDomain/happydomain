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
	"context"
	"errors"
	"fmt"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindInstance is a safe whose key is wrapped by the instance keyset,
	// and names the entries of that keyset in a keyring.
	KindInstance = "instance"

	wrapVersion = "hds:1:safe"
)

// instanceStore keeps the key of each safe wrapped by the instance keyset,
// held by the server outside the database.
type instanceStore struct {
	key *InstanceKey

	// checks keeps the check record, verified at startup.
	checks CheckStorage
}

func newInstanceStore(cfg Config) (Store, error) {
	if cfg.InstanceKey == nil {
		return nil, fmt.Errorf("%w: no instance keyset, set -secret-keyset-file (see `happydomain secret-keyset generate`)", ErrStoreUnavailable)
	}
	if cfg.Safes == nil {
		return nil, errors.New("an instance keyset requires a safe storage")
	}
	return &instanceStore{key: cfg.InstanceKey, checks: cfg.Checks}, nil
}

// CheckStartup refuses a keyset that is not the one the database was used
// with, see VerifyCheck.
func (s *instanceStore) CheckStartup(context.Context) error {
	if s.checks == nil {
		return errors.New("no storage for the instance keyset check record")
	}
	return s.key.VerifyCheck(s.checks)
}

// Provision gives safe a new key, wrapped by the instance keyset.
func (s *instanceStore) Provision(_ context.Context, safe *happydns.Safe) error {
	dek, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return err
	}
	w, err := s.key.Wrap(safe, dek)
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}
	return nil
}

// Unlock unwraps the key of safe with the instance keyset.
func (s *instanceStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != KindInstance {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindInstance)
	}

	dek, err := s.key.Unwrap(safe)
	if err != nil {
		return nil, err
	}
	primitive, err := aead.New(dek)
	if err != nil {
		return nil, err
	}
	return &aeadSafe{id: safe.Id, primitive: primitive}, nil
}

// Rewrap wraps w again under the primary instance key, when another one made
// it.
func (s *instanceStore) Rewrap(_ context.Context, safe *happydns.Safe, w *happydns.WrappedKeyset) (bool, error) {
	id, err := wrappingKeyId(*w)
	if err != nil {
		return false, err
	}
	if id == s.key.PrimaryKeyId() {
		return false, nil
	}

	dek, err := s.key.unwrapEntry(safe, *w)
	if err != nil {
		return false, err
	}
	nw, err := s.key.Wrap(safe, dek)
	if err != nil {
		return false, err
	}
	*w = nw
	return true, nil
}

// Keys lists the keys of the instance keyset.
func (s *instanceStore) Keys() []KeyInfo {
	return DescribeKeyset(s.key.handle)
}

// WrappingKey returns the identifier of the instance key that made w.
func (s *instanceStore) WrappingKey(w happydns.WrappedKeyset) (uint32, error) {
	return wrappingKeyId(w)
}

// wrappingKeyId returns the identifier of the instance key that wrapped w,
// read from the Tink output prefix of the ciphertext.
func wrappingKeyId(w happydns.WrappedKeyset) (uint32, error) {
	enc, err := keyset.NewBinaryReader(bytes.NewReader(w.Blob)).ReadEncrypted()
	if err != nil {
		return 0, err
	}

	return ciphertextKeyId(enc.GetEncryptedKeyset())
}
