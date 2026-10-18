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

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

// ErrWrongInstanceKey is returned when the configured instance keyset does
// not open the check record: it is not the keyset the stored secrets were
// sealed with.
var ErrWrongInstanceKey = errors.New("the instance keyset does not open the check record: it is not the keyset this database was used with")

// The check record has its own associated data, so it can never be passed
// off as a wrapped safe key, nor the reverse.
const (
	checkPlaintext      = "happyDomain instance keyset check"
	checkAssociatedData = "hds:1:check"
)

// InstanceKey is the keyset held by the instance, which wraps the keys of
// the safes.
type InstanceKey struct {
	handle *keyset.Handle
	aead   tink.AEAD
}

// StartupCheck refuses a configuration that would leave secrets unreadable:
// the instance policy without a keyset, instance safes without a keyset to
// open them, or a keyset that is not the one the database was used with.
func StartupCheck(policy Policy, key *InstanceKey, store Storage) error {
	if policy == PolicyInstance && key == nil {
		return errors.New("the instance secret policy requires an instance keyset: see `happydomain secret-keyset generate`")
	}

	if key == nil {
		has, err := hasSafeOfKind(store, KindInstance)
		if err != nil {
			return fmt.Errorf("unable to list safes: %w", err)
		}
		if has {
			return errors.New("secrets are sealed under an instance keyset, but none is configured: set -secret-keyset-file")
		}
		return nil
	}

	return key.VerifyCheck(store)
}

// hasSafeOfKind tells whether a safe of kind is stored. A record that does not
// decode tells no kind: it does not keep happyDomain from starting.
func hasSafeOfKind(store SafeStorage, kind string) (bool, error) {
	has := false
	err := eachSafe(store, func(safe *happydns.Safe) {
		has = has || safe.Kind == kind
	}, nil)
	return has, err
}

// NewInstanceKey returns the InstanceKey of h.
func NewInstanceKey(h *keyset.Handle) (*InstanceKey, error) {
	a, err := aead.New(h)
	if err != nil {
		return nil, fmt.Errorf("unusable instance keyset: %w", err)
	}
	return &InstanceKey{handle: h, aead: a}, nil
}

// LoadInstanceKey reads the instance keyset from a cleartext JSON file.
func LoadInstanceKey(path string) (*InstanceKey, error) {
	h, err := ReadKeysetFile(path)
	if err != nil {
		return nil, err
	}
	return NewInstanceKey(h)
}

// PrimaryKeyId is the identifier of the key new wrappings are made with.
func (k *InstanceKey) PrimaryKeyId() uint32 {
	return k.handle.KeysetInfo().GetPrimaryKeyId()
}

// VerifyCheck opens the check record stored in store, or creates it under the
// primary key when there is none. A record the keyset does not open is
// ErrWrongInstanceKey: starting would make every secret sealed with the
// right keyset unreadable, one provider at a time.
func (k *InstanceKey) VerifyCheck(store CheckStorage) error {
	record, err := store.GetSecretCheck()
	if errors.Is(err, happydns.ErrNotFound) {
		return k.putCheck(store)
	}
	if err != nil {
		return fmt.Errorf("unable to read the instance keyset check record: %w", err)
	}

	if !k.opensCheck(record) {
		return ErrWrongInstanceKey
	}

	// Sealed under an older key, the record would keep that key needed
	// after every safe has been rewrapped, and removing it, as the key
	// report allows, would refuse to start: move it under the primary. A
	// RAW or LEGACY key does not tell its id: resealed, harmlessly, at
	// every start.
	if id, err := ciphertextKeyId(record); err != nil || id != k.PrimaryKeyId() {
		return k.putCheck(store)
	}
	return nil
}

// OpensCheckRecord tells whether this keyset opens the check record stored in
// store, without changing it. Without a check record, it is false.
func (k *InstanceKey) OpensCheckRecord(store CheckStorage) (bool, error) {
	record, err := store.GetSecretCheck()
	if errors.Is(err, happydns.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("unable to read the instance keyset check record: %w", err)
	}
	return k.opensCheck(record), nil
}

// opensCheck tells whether record is the check record sealed by this keyset.
func (k *InstanceKey) opensCheck(record []byte) bool {
	pt, err := k.aead.Decrypt(record, []byte(checkAssociatedData))
	return err == nil && bytes.Equal(pt, []byte(checkPlaintext))
}

// putCheck seals the check record under the primary key and stores it.
func (k *InstanceKey) putCheck(store CheckStorage) error {
	ct, err := k.aead.Encrypt([]byte(checkPlaintext), []byte(checkAssociatedData))
	if err != nil {
		return err
	}
	return store.PutSecretCheck(ct)
}

// ciphertextKeyId returns the identifier of the key that produced ct, read
// from its TINK output prefix: 0x01, then the big-endian key id. The LEGACY
// and CRUNCHY prefixes are not recognized: the keysets happyDomain generates
// and rotates never use them.
func ciphertextKeyId(ct []byte) (uint32, error) {
	if len(ct) < 5 || ct[0] != 0x01 {
		return 0, errors.New("ciphertext without a Tink key prefix")
	}
	return binary.BigEndian.Uint32(ct[1:5]), nil
}
