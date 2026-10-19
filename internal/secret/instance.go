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

	return happydns.WrappedKeyset{KEK: KindInstance, Blob: buf.Bytes()}, nil
}

// Unwrap decrypts the key of safe from its instance keyring entry.
func (k *InstanceKey) Unwrap(safe *happydns.Safe) (*keyset.Handle, error) {
	for _, w := range safe.Keyring {
		if w.KEK != KindInstance {
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
