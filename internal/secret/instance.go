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
	"os"
	"path/filepath"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"

	"git.happydns.org/happyDomain/model"
)

// ErrWrongInstanceKey is returned when the configured instance keyset does
// not open the check record: it is not the keyset the stored secrets were
// sealed with.
var ErrWrongInstanceKey = errors.New("the instance keyset does not open the check record: it is not the keyset this database was used with")

const (
	checkPlaintext      = "happyDomain instance keyset check"
	checkAssociatedData = "hds:1:check"
)

// GenerateInstanceKeyset returns a new keyset holding one AES256-GCM key.
func GenerateInstanceKeyset() (*keyset.Handle, error) {
	return keyset.NewHandle(aead.AES256GCMKeyTemplate())
}

// ReadKeysetFile reads a cleartext JSON keyset, as written by tinkey.
func ReadKeysetFile(path string) (*keyset.Handle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(f))
	if err != nil {
		return nil, fmt.Errorf("unable to read keyset %s: %w", path, err)
	}
	return h, nil
}

// WriteNewKeysetFile writes h as a cleartext JSON keyset readable by its
// owner only. It refuses to overwrite a file: losing a keyset loses every
// secret it protects.
func WriteNewKeysetFile(path string, h *keyset.Handle) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}

	// On disk before saying it is written: a power loss must not leave an
	// empty file in place of the only copy of the key.
	err = insecurecleartextkeyset.Write(h, keyset.NewJSONWriter(f))
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(path)
		return err
	}

	return syncDir(filepath.Dir(path))
}

// syncDir flushes the directory dir to disk, so that a file just created in
// it survives a power loss.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Sync()
}

// KeysetFileTooOpen reports whether the keyset file can be read by others
// than its owner.
func KeysetFileTooOpen(path string) (bool, error) {
	st, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return st.Mode().Perm()&0o077 != 0, nil
}

// KeyInfo describes a key of a keyset, without its material.
type KeyInfo struct {
	Id      uint32
	Primary bool
	Status  string
	Type    string
}

func (k KeyInfo) String() string {
	primary := ""
	if k.Primary {
		primary = " (primary)"
	}
	return fmt.Sprintf("%d%s %s %s", k.Id, primary, k.Status, k.Type)
}

// DescribeKeyset lists the keys of h, without their material.
func DescribeKeyset(h *keyset.Handle) []KeyInfo {
	info := h.KeysetInfo()

	keys := make([]KeyInfo, 0, len(info.GetKeyInfo()))
	for _, k := range info.GetKeyInfo() {
		keys = append(keys, KeyInfo{
			Id:      k.GetKeyId(),
			Primary: k.GetKeyId() == info.GetPrimaryKeyId(),
			Status:  k.GetStatus().String(),
			Type:    k.GetTypeUrl(),
		})
	}
	return keys
}

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

// CheckStorage keeps the check record of the instance keyset.
type CheckStorage interface {
	// GetSecretCheck returns the check record, or happydns.ErrNotFound.
	GetSecretCheck() ([]byte, error)

	// PutSecretCheck stores the check record.
	PutSecretCheck(record []byte) error
}

// VerifyCheck opens the check record stored in store, or creates it under the
// primary key when there is none. A record the keyset does not open is
// ErrWrongInstanceKey: starting would make every secret sealed with the
// right keyset unreadable, one provider at a time.
func (k *InstanceKey) VerifyCheck(store CheckStorage) error {
	record, err := store.GetSecretCheck()
	if errors.Is(err, happydns.ErrNotFound) {
		ct, err := k.aead.Encrypt([]byte(checkPlaintext), []byte(checkAssociatedData))
		if err != nil {
			return err
		}
		return store.PutSecretCheck(ct)
	}
	if err != nil {
		return fmt.Errorf("unable to read the instance keyset check record: %w", err)
	}

	pt, err := k.aead.Decrypt(record, []byte(checkAssociatedData))
	if err != nil || !bytes.Equal(pt, []byte(checkPlaintext)) {
		return ErrWrongInstanceKey
	}
	return nil
}

// StartupCheck refuses a configuration that would leave secrets unreadable:
// the instance policy without a keyset, or a keyset that is not the one the
// database was used with.
func StartupCheck(policy Policy, key *InstanceKey, store CheckStorage) error {
	if policy == PolicyInstance && key == nil {
		return errors.New("the instance secret policy requires an instance keyset: see `happydomain secret-keyset generate`")
	}

	if key != nil {
		if err := key.VerifyCheck(store); err != nil {
			return err
		}
	}

	return nil
}
