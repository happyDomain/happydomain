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
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"git.happydns.org/happyDomain/model"
)

const (
	// KindFile is a personal keystore whose references name files on the
	// server, see fileStore.
	KindFile = "file"

	// fileDirParam names, in the parameters creating a keystore of
	// KindFile, its directory, relative to the one of its owner.
	fileDirParam = "directory"

	// maxSecretFileSize is the size of the largest file read as a secret.
	maxSecretFileSize = 64 * 1024
)

var (
	errNoSecretFile         = errors.New("no such file in the directory of the keystore")
	errUnreadableSecret     = errors.New("this file of the keystore directory cannot be read as a secret")
	errInvalidFileReference = errors.New("a reference names a file, by its path relative to the directory of the keystore")
)

// fileDir is the directory of the file store, from -secret-file-dir.
var fileDir string

func init() {
	flag.StringVar(&fileDir, "secret-file-dir", "", "Directory holding, for the \"file\" secret store, a directory per user named after their identifier, where they keep one file per credential")
}

// fileStore resolves references to files, each holding one secret, as the
// secrets of a container or the credentials of a systemd unit do. The files
// of an owner are under their directory, named after their identifier in the
// directory of the store: nothing outside it is read, through a symbolic
// link neither.
type fileStore struct {
	dir string
}

func newFileStore(Config) (Store, error) { return newFileStoreIn(fileDir) }

// newFileStoreIn is newFileStore over dir rather than -secret-file-dir, so
// that tests need not set the flag.
func newFileStoreIn(dir string) (Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: no directory for the files of the keystores, set -secret-file-dir", ErrStoreUnavailable)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &fileStore{dir: dir}, nil
}

// CheckStartup refuses a directory that is not one.
func (s *fileStore) CheckStartup(context.Context) error {
	fi, err := os.Stat(s.dir)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", s.dir)
	}
	return nil
}

// OpensUnattended: the files are read whenever a secret is used.
func (s *fileStore) OpensUnattended() bool { return true }

func (s *fileStore) ReferenceHint() string {
	return "the name of the file holding the secret, relative to the directory of the keystore"
}

func (s *fileStore) Reference(payload []byte) (string, error) {
	ref := string(payload)
	if !validFileReference(ref) {
		return "", errInvalidFileReference
	}
	return ref, nil
}

// validFileReference tells whether ref names a file below a directory.
func validFileReference(ref string) bool {
	return filepath.IsLocal(ref) && !strings.ContainsRune(ref, 0)
}

// ownerDir is the directory holding the files of owner.
func (s *fileStore) ownerDir(owner happydns.Identifier) string {
	return filepath.Join(s.dir, owner.String())
}

func (s *fileStore) CreateForm() []happydns.Field {
	return []happydns.Field{{
		Id:          fileDirParam,
		Type:        "string",
		Label:       "Directory",
		Description: "The directory holding one file per secret, relative to your own directory on the server, which the administrator creates. Leave empty for your own directory.",
	}}
}

// Create points safe to the directory params names, once sure it exists.
func (s *fileStore) Create(_ context.Context, safe *happydns.Safe, params json.RawMessage) error {
	var p map[string]string
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	dir := p[fileDirParam]
	if dir != "" && !validFileReference(dir) {
		return fmt.Errorf("the directory %q is not relative to yours", dir)
	}

	// Errors name the identifier, never s.dir nor the cause: where and how
	// the server keeps the files is not the user's business.
	root, err := s.openRoot(safe.Owner, dir)
	if errors.Is(err, fs.ErrNotExist) && dir == "" {
		return fmt.Errorf("your directory does not exist: ask the administrator to create it, named after your identifier %s", safe.Owner.String())
	}
	if err != nil {
		return fmt.Errorf("the directory %q cannot be opened in yours", dir)
	}
	root.Close()

	raw, err := json.Marshal(map[string]string{fileDirParam: dir})
	if err != nil {
		return err
	}
	safe.Keyring = []happydns.WrappedKeyset{{KEK: KindFile, Params: raw}}
	return nil
}

// openRoot opens dir, in the directory of owner, so that nothing outside it
// is reached.
func (s *fileStore) openRoot(owner happydns.Identifier, dir string) (*os.Root, error) {
	root, err := os.OpenRoot(s.ownerDir(owner))
	if err != nil || dir == "" {
		return root, err
	}
	defer root.Close()
	return root.OpenRoot(dir)
}

func (s *fileStore) Unlock(_ context.Context, safe *happydns.Safe) (UnlockedSafe, error) {
	if safe.Kind != KindFile {
		return nil, fmt.Errorf("safe %s is of kind %q, not %q", safe.Id.String(), safe.Kind, KindFile)
	}
	for _, e := range safe.Keyring {
		if e.KEK != KindFile {
			continue
		}
		var p map[string]string
		if err := json.Unmarshal(e.Params, &p); err != nil {
			return nil, fmt.Errorf("safe %s has unreadable parameters: %w", safe.Id.String(), err)
		}
		if dir := p[fileDirParam]; dir != "" && !validFileReference(dir) {
			return nil, fmt.Errorf("safe %s names a directory outside the one of its owner", safe.Id.String())
		}
		return &fileUnlocked{store: s, owner: safe.Owner, dir: p[fileDirParam]}, nil
	}
	return nil, fmt.Errorf("safe %s has no directory", safe.Id.String())
}

// fileUnlocked reads the files of the directory of one keystore.
type fileUnlocked struct {
	store *fileStore
	owner happydns.Identifier
	dir   string
}

// Seal keeps ref, once sure it names a file holding a secret.
func (u *fileUnlocked) Seal(_ context.Context, _ SecretContext, ref []byte) ([]byte, error) {
	if _, err := u.read(string(ref)); err != nil {
		return nil, err
	}
	return bytes.Clone(ref), nil
}

// Open reads the file payload names.
func (u *fileUnlocked) Open(_ context.Context, _ SecretContext, payload []byte) ([]byte, error) {
	return u.read(string(payload))
}

// read returns the secret the file ref names holds, without its final
// newline. What ref holds may be a secret entered by mistake: no error
// repeats it, nor the path made of it.
func (u *fileUnlocked) read(ref string) ([]byte, error) {
	if !validFileReference(ref) {
		return nil, errInvalidFileReference
	}
	root, err := u.store.openRoot(u.owner, u.dir)
	if err != nil {
		return nil, errors.New("the directory of the keystore cannot be opened")
	}
	defer root.Close()

	// Without waiting: a named pipe would block until a writer comes. It
	// is refused below, as anything but a regular file.
	f, err := root.OpenFile(ref, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errNoSecretFile
	}
	if err != nil {
		return nil, errUnreadableSecret
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxSecretFileSize {
		return nil, errUnreadableSecret
	}
	content, err := io.ReadAll(io.LimitReader(f, maxSecretFileSize+1))
	if err != nil || len(content) > maxSecretFileSize {
		return nil, errUnreadableSecret
	}

	content = bytes.TrimSuffix(content, []byte("\n"))
	content = bytes.TrimSuffix(content, []byte("\r"))
	if len(content) == 0 {
		return nil, errors.New("this file of the keystore directory is empty")
	}
	return content, nil
}
