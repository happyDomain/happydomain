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
	"encoding/json"
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindKeyFile is a personal keystore whose key is wrapped by a Tink
	// keyset kept in a file, in a directory of its owner on the server. It
	// opens by itself as long as the file exists.
	KindKeyFile = "keyfile"

	// keyFileParam names, in the parameters creating a keystore of
	// KindKeyFile, its keyset file, relative to the directory of its owner.
	keyFileParam = "file"
)

var errInvalidKeysetName = errors.New("the keyset file is named by its path relative to your directory")

var errInvalidKeysetFile = errors.New("this file is not a Tink keyset in JSON")

// keyFileStore wraps the key of each safe under the keyset of a file of its
// owner, as the instance store does with the keyset of the instance. The files
// are found as by the file store, which it relies on for the directory.
type keyFileStore struct {
	files *fileStore
}

func newKeyFileStore(Config) (Store, error) { return newKeyFileStoreIn(fileDir) }

// newKeyFileStoreIn returns the keyfile store over dir.
func newKeyFileStoreIn(dir string) (Store, error) {
	s, err := newFileStoreIn(dir)
	if err != nil {
		return nil, err
	}
	return &keyFileStore{s.(*fileStore)}, nil
}

// OpensUnattended: the keyset file is read whenever the keystore is unlocked.
func (s *keyFileStore) OpensUnattended() bool { return true }

// CheckStartup refuses a -secret-file-dir that is missing or not a directory.
func (s *keyFileStore) CheckStartup(ctx context.Context) error { return s.files.CheckStartup(ctx) }

func (s *keyFileStore) CreateForm() []happydns.Field {
	return []happydns.Field{{
		Id:          keyFileParam,
		Type:        "string",
		Label:       "Keyset file",
		Description: "The name of a Tink keyset file (JSON) in your directory on the server, which the administrator creates. Without this file, the keystore stays locked; lost, what it holds is lost.",
		Required:    true,
	}}
}

// loadKey reads the keyset in the file name of the directory of owner. Its
// errors reach the user: they never carry the underlying error, which would
// reveal paths of the server.
func (s *keyFileStore) loadKey(owner happydns.Identifier, name string) (*InstanceKey, error) {
	if !validFileReference(name) {
		return nil, errInvalidKeysetName
	}
	root, err := s.files.openRoot(owner, "")
	if err != nil {
		return nil, errors.New("your directory on the server cannot be opened")
	}
	defer root.Close()

	content, err := readSmallFile(root, name)
	if errors.Is(err, errNoSecretFile) {
		return nil, errors.New("no such keyset file in your directory")
	}
	if err != nil {
		return nil, errors.New("this keyset file cannot be read")
	}
	h, err := ReadKeyset(bytes.NewReader(content))
	if err != nil {
		return nil, errInvalidKeysetFile
	}
	key, err := NewInstanceKey(h)
	if err != nil {
		return nil, errInvalidKeysetFile
	}
	return key, nil
}

// Create wraps a new key of safe under the keyset of the file params names.
func (s *keyFileStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	p, err := readFields(params)
	if err != nil {
		return err
	}
	key, err := s.loadKey(safe.Owner, p[keyFileParam])
	if err != nil {
		return err
	}
	dek, err := GenerateInstanceKeyset()
	if err != nil {
		return err
	}
	w, err := key.WrapAs(KindKeyFile, safe, dek)
	if err != nil {
		return err
	}
	if w.Params, err = json.Marshal(map[string]string{keyFileParam: p[keyFileParam]}); err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{w}
	return nil
}

// Unlock reads the keyset file again, so that a file put back or replaced is
// taken into account, and unwraps the key of safe with it.
func (s *keyFileStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != KindKeyFile {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindKeyFile)
	}
	for _, w := range safe.Keyring {
		if w.KEK != KindKeyFile {
			continue
		}
		p, err := readFields(w.Params)
		if err != nil {
			return nil, fmt.Errorf("safe %s has unreadable parameters: %w", safe.Id.String(), err)
		}
		key, err := s.loadKey(safe.Owner, p[keyFileParam])
		if err != nil {
			return nil, fmt.Errorf("keystore %s is locked: %w", safe.Id.String(), err)
		}
		return key.open(safe, w)
	}
	return nil, fmt.Errorf("safe %s has no key wrapped by a keyset file", safe.Id.String())
}
