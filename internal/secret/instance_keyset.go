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
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
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

	// On disk before taking the place of the previous keyset.
	err = insecurecleartextkeyset.Write(h, keyset.NewJSONWriter(tmp))
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
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
