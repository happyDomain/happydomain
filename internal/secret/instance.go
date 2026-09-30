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
	"log"
	"os"
	"path/filepath"
	"syscall"

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

// RotateInstanceKeyset returns h with a new AES256-GCM key as primary. The
// previous keys are kept: what they sealed still opens.
func RotateInstanceKeyset(h *keyset.Handle) (*keyset.Handle, error) {
	m := keyset.NewManagerFromHandle(h)

	id, err := m.Add(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, err
	}
	if err := m.SetPrimary(id); err != nil {
		return nil, err
	}

	return m.Handle()
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
	if err := writeKeysetFile(f, h); err != nil {
		os.Remove(path)
		return err
	}

	return syncDir(filepath.Dir(path))
}

// writeKeysetFile writes h to f as a cleartext JSON keyset, flushes it to
// disk and closes f, whatever the outcome.
func writeKeysetFile(f *os.File, h *keyset.Handle) error {
	err := insecurecleartextkeyset.Write(h, keyset.NewJSONWriter(f))
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// syncDir flushes the directory dir to disk, so that a file just created or
// renamed in it survives a power loss.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	return d.Sync()
}

// ReplaceKeysetFile replaces the existing keyset file at path with h, through
// a rename so that an interruption never leaves a truncated keyset.
//
// The new file gets the owner, group and mode of the one it replaces: the
// service may read its keyset through its group, and a rotation run through
// sudo must not leave it unreadable. When path is a symbolic link, the file it
// points at is replaced and the link is kept.
func ReplaceKeysetFile(path string, h *keyset.Handle) error {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	st, err := os.Stat(target)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".keyset-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	// Owner first: changing it may clear mode bits.
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		if err := tmp.Chown(int(sys.Uid), int(sys.Gid)); err != nil {
			tmp.Close()
			return fmt.Errorf("unable to give the new keyset the owner of %s: %w", target, err)
		}
	}
	if err := tmp.Chmod(st.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if err := writeKeysetFile(tmp, h); err != nil {
		return err
	}

	if err := os.Rename(tmp.Name(), target); err != nil {
		return err
	}

	return syncDir(filepath.Dir(target))
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
	Id      uint32 `json:"id"`
	Primary bool   `json:"primary"`
	Status  string `json:"status"`
	Type    string `json:"type,omitempty"`
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

// PrimaryKeyId is the identifier of the key new wrappings are made with.
func (k *InstanceKey) PrimaryKeyId() uint32 {
	return k.handle.KeysetInfo().GetPrimaryKeyId()
}

// CheckStorage keeps the check record of the instance keyset.
type CheckStorage interface {
	// GetSecretCheck returns the check record, or happydns.ErrNotFound.
	GetSecretCheck() ([]byte, error)

	// PutSecretCheck stores the check record.
	PutSecretCheck(record []byte) error

	// DeleteSecretCheck removes the check record, if any: the next keyset
	// configured creates a new one.
	DeleteSecretCheck() error
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

	pt, err := k.aead.Decrypt(record, []byte(checkAssociatedData))
	if err != nil || !bytes.Equal(pt, []byte(checkPlaintext)) {
		return ErrWrongInstanceKey
	}

	// Sealed under an older key, the record would keep that key needed
	// after every safe has been rewrapped, and removing it, as the key
	// report allows, would refuse to start: move it under the primary.
	if id, err := ciphertextKeyId(record); err != nil || id != k.PrimaryKeyId() {
		return k.putCheck(store)
	}
	return nil
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
// from its Tink output prefix: one version byte, then the key identifier.
func ciphertextKeyId(ct []byte) (uint32, error) {
	if len(ct) < 5 || ct[0] != 0x01 {
		return 0, errors.New("ciphertext without a Tink key prefix")
	}
	return binary.BigEndian.Uint32(ct[1:5]), nil
}

// Storage is what secret management keeps in the database.
type Storage interface {
	CheckStorage
	SafeStorage
}

// StartupCheck refuses a configuration that would leave secrets unreadable:
// the instance policy without a keyset, instance safes without a keyset to
// open them, or a keyset that is not the one the database was used with.
//
// The check record only stands for the instance safes: with none left, it is
// stale and dropped, and none is created unless the instance policy is about
// to seal. Otherwise, starting once with a keyset configured would refuse
// every other keyset later, though nothing is sealed under it.
//
// A safe record that does not decode may be an instance safe. While one is
// there, the check record stays, and the keyset it names stays required,
// until the record is repaired, or removed by hand if known to be lost: `tidy`
// never deletes a safe record that does not decode. Without a check record,
// nothing then vouches for a keyset: it is accepted, but not recorded.
func StartupCheck(policy Policy, key *InstanceKey, store Storage) error {
	if policy == PolicyInstance && key == nil {
		return errors.New("the instance secret policy requires an instance keyset: see `happydomain secret-keyset generate`")
	}

	safe, damaged, err := firstSafeOfKind(store, KindInstance)
	if err != nil {
		return fmt.Errorf("unable to list safes: %w", err)
	}
	for _, k := range damaged {
		log.Printf("secret: safe record %q does not decode, it may be an instance safe", k)
	}

	if safe == nil && len(damaged) == 0 {
		if err := store.DeleteSecretCheck(); err != nil {
			return fmt.Errorf("unable to delete the stale keyset check record: %w", err)
		}
		if key == nil || policy != PolicyInstance {
			return nil
		}
		return key.VerifyCheck(store)
	}

	_, err = store.GetSecretCheck()
	recorded := err == nil
	if err != nil && !errors.Is(err, happydns.ErrNotFound) {
		return fmt.Errorf("unable to read the keyset check record: %w", err)
	}

	if key == nil {
		if safe != nil {
			return errors.New("secrets are sealed under an instance keyset, but none is configured: set -secret-keyset-file")
		}
		if recorded {
			return fmt.Errorf("%d safe record(s) do not decode and may be instance safes, and the keyset check record is there: set -secret-keyset-file until they are repaired, or removed by hand if known to be lost", len(damaged))
		}
		return nil
	}

	if !recorded {
		// VerifyCheck would record whatever keyset it is given. A safe
		// already stored tells whether it is the right one; without one,
		// nothing does.
		if safe == nil {
			log.Printf("secret: no keyset check record, and no instance safe that decodes to check the keyset on: it is not recorded")
			return nil
		}
		if _, err := key.Unwrap(safe); err != nil {
			return fmt.Errorf("%w (no check record yet, so safe %s was tried instead)", ErrWrongInstanceKey, safe.Id.String())
		}
	}

	return key.VerifyCheck(store)
}

// firstSafeOfKind returns a safe of kind, or nil when there is none, along
// with the keys of the records met that do not decode. Those are only all of
// them when no safe of kind is found.
func firstSafeOfKind(store SafeStorage, kind string) (safe *happydns.Safe, damaged []string, err error) {
	damaged, err = scanSafes(store, func(s *happydns.Safe) bool {
		if s.Kind == kind {
			safe = s
			return false
		}
		return true
	})
	return safe, damaged, err
}

// SafeStorage keeps the safes.
type SafeStorage interface {
	// ListAllSafes lists every safe.
	ListAllSafes() (happydns.Iterator[happydns.Safe], error)

	// GetSafe returns the safe with the given identifier, or
	// happydns.ErrSafeNotFound.
	GetSafe(id happydns.Identifier) (*happydns.Safe, error)

	// GetSafeByOwner returns the safe of the given kind owned by owner, or
	// happydns.ErrSafeNotFound.
	GetSafeByOwner(owner happydns.Identifier, kind string) (*happydns.Safe, error)

	// CreateSafe stores a new safe under the identifier it carries, which the
	// caller generates with happydns.NewRandomIdentifier. Fails with
	// happydns.ErrInvalidIdentifier, or happydns.ErrAlreadyExists when the
	// identifier is taken or its owner already has a safe of its kind.
	CreateSafe(safe *happydns.Safe) error

	// UpdateSafe replaces a stored safe.
	UpdateSafe(safe *happydns.Safe) error

	// ReplaceSafe rewrites the safe id with what update returns from the
	// stored safe. update may be called with a copy it can change. It
	// writes nothing when update returns nil, and fails, writing nothing,
	// with happydns.ErrSafeNotFound or happydns.ErrChangedMeanwhile when the
	// safe was deleted or changed in between. The owner and kind of a safe
	// cannot change.
	ReplaceSafe(id happydns.Identifier, update func(*happydns.Safe) (*happydns.Safe, error)) error

	// DeleteSafe removes a safe: every value sealed in it becomes
	// unreadable.
	DeleteSafe(id happydns.Identifier) error
}

// DamagedSafeStorage deals with the safe records that do not decode.
type DamagedSafeStorage interface {
	// ListDamagedSafes lists the safe records that do not decode.
	ListDamagedSafes() ([]*happydns.DamagedSafe, error)

	// DeleteDamagedSafe deletes the safe record id, and the owner index
	// pointing to it, as long as it does not decode: it fails with
	// happydns.ErrSafeNotDamaged when it does, and happydns.ErrSafeNotFound
	// when there is none.
	DeleteDamagedSafe(id happydns.Identifier) error

	// RepairSafe puts safe in place of its record, as long as that record
	// does not decode: it fails with happydns.ErrSafeNotDamaged when it
	// does, and happydns.ErrSafeNotFound when there is none. It points the
	// owner index to safe unless it points to another safe.
	RepairSafe(safe *happydns.Safe) error
}

// SafeRestorer puts back the safes of a backup.
type SafeRestorer interface {
	// RestoreSafe stores a safe coming from elsewhere, such as a backup,
	// under the identifier it carries. Unlike CreateSafe, it does not fail
	// when the owner already has a safe of that kind: both are kept, and new
	// secrets keep going to the one in place. Fails with
	// happydns.ErrInvalidIdentifier, or happydns.ErrAlreadyExists when the
	// identifier is taken.
	RestoreSafe(safe *happydns.Safe) error
}
