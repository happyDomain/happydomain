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
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// fileFixture is a file store over a directory of its own, with the
// directory of owner, ready to hold their files.
type fileFixture struct {
	dir   string
	owner happydns.Identifier
	store *fileStore
}

func newFileFixture(t *testing.T) fileFixture {
	t.Helper()
	f := fileFixture{dir: t.TempDir(), owner: happydns.Identifier{0x01, 0x02}}
	if err := os.Mkdir(f.userDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := newFileStoreIn(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	f.store = store.(*fileStore)
	return f
}

func (f fileFixture) userDir() string {
	return filepath.Join(f.dir, f.owner.String())
}

func (f fileFixture) write(t *testing.T, name, content string) {
	t.Helper()
	path := filepath.Join(f.userDir(), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func fileParams(dir string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{fileDirParam: dir})
	return b
}

// safe returns a new file safe of the owner of f, over dir.
func (f fileFixture) safe(t *testing.T, dir string) *happydns.Safe {
	t.Helper()
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindFile}
	if err := f.store.Create(context.Background(), safe, fileParams(dir)); err != nil {
		t.Fatalf("Create(%q): %v", dir, err)
	}
	return safe
}

func (f fileFixture) unlocked(t *testing.T, dir string) UnlockedSafe {
	t.Helper()
	u, err := f.store.Unlock(context.Background(), f.safe(t, dir))
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	return u
}

func TestFileStoreNeedsItsDirectory(t *testing.T) {
	if _, err := newFileStoreIn(""); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newFileStore(no directory) = %v, want ErrStoreUnavailable", err)
	}

	store, err := newFileStoreIn(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted a missing directory")
	}

	file := filepath.Join(t.TempDir(), "keyset.json")
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err = newFileStoreIn(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted a file")
	}

	f := newFileFixture(t)
	if err := f.store.CheckStartup(context.Background()); err != nil {
		t.Errorf("CheckStartup = %v", err)
	}
}

func TestFileStoreSealsReferencesAndOpensFiles(t *testing.T) {
	ctx := context.Background()
	f := newFileFixture(t)
	f.write(t, "ovh_secret", "s3cr3t\n")
	f.write(t, "ovh/key", "with\r\n")
	u := f.unlocked(t, "")
	sc := objectContext()

	payload, err := u.Seal(ctx, sc, []byte("ovh_secret"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if strings.Contains(string(payload), "s3cr3t") {
		t.Error("the payload holds the secret")
	}
	if ref, err := f.store.Reference(payload); err != nil || ref != "ovh_secret" {
		t.Errorf("Reference = %q, %v", ref, err)
	}
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "s3cr3t" {
		t.Errorf("Open = %q, %v; want the file without its final newline", got, err)
	}

	sub, err := u.Seal(ctx, sc, []byte("ovh/key"))
	if err != nil {
		t.Fatalf("Seal(in a subdirectory): %v", err)
	}
	if got, err := u.Open(ctx, sc, sub); err != nil || string(got) != "with" {
		t.Errorf("Open(in a subdirectory) = %q, %v", got, err)
	}

	// Only the final newline is dropped, and the largest file is read whole.
	f.write(t, "two_lines", "a\n\n")
	f.write(t, "exact", strings.Repeat("x", maxSecretFileSize))
	for ref, want := range map[string]string{"two_lines": "a\n", "exact": strings.Repeat("x", maxSecretFileSize)} {
		p, err := u.Seal(ctx, sc, []byte(ref))
		if err != nil {
			t.Fatalf("Seal(%s): %v", ref, err)
		}
		if got, err := u.Open(ctx, sc, p); err != nil || string(got) != want {
			t.Errorf("Open(%s) = %d bytes, %v; want %d bytes", ref, len(got), err, len(want))
		}
	}

	// The file is read each time: its owner changes the secret there.
	f.write(t, "ovh_secret", "rotated")
	if got, err := u.Open(ctx, sc, payload); err != nil || string(got) != "rotated" {
		t.Errorf("Open after a change = %q, %v", got, err)
	}
}

func TestFileStoreKeystoreDirectory(t *testing.T) {
	ctx := context.Background()
	f := newFileFixture(t)
	f.write(t, "work/token", "w")
	f.write(t, "token", "home")

	u := f.unlocked(t, "work")
	payload, err := u.Seal(ctx, objectContext(), []byte("token"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := u.Open(ctx, objectContext(), payload); err != nil || string(got) != "w" {
		t.Errorf("Open = %q, %v; want the file of the keystore directory", got, err)
	}

	for name, dir := range map[string]string{
		"missing":   "nope",
		"parent":    "../other",
		"absolute":  "/etc",
		"a file":    "token",
		"traversal": "work/../../other",
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindFile}
		if err := f.store.Create(ctx, safe, fileParams(dir)); err == nil {
			t.Errorf("Create(%s) succeeded", name)
		}
	}

	stranger := &happydns.Safe{Id: happydns.Identifier{0x06}, Owner: happydns.Identifier{0x09}, Kind: KindFile}
	err = f.store.Create(ctx, stranger, fileParams(""))
	if err == nil {
		t.Fatal("Create succeeded for an owner without a directory")
	}
	if !strings.Contains(err.Error(), stranger.Owner.String()) || strings.Contains(err.Error(), f.dir) {
		t.Errorf("Create(no directory) = %v, want it naming the directory to create by the identifier of its owner, not where the server keeps it", err)
	}
}

// A reference names a file of the keystore directory, never one elsewhere:
// not of another user, nor the instance keyset.
func TestFileStoreStaysInItsDirectory(t *testing.T) {
	ctx := context.Background()
	f := newFileFixture(t)
	other := filepath.Join(f.dir, "other-user")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "secret"), []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, "keyset.json"), []byte("instance"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(other, "secret"), filepath.Join(f.userDir(), "link")); err != nil {
		t.Fatal(err)
	}
	f.write(t, "dir/file", "x")
	f.write(t, "big", strings.Repeat("x", maxSecretFileSize+1))
	f.write(t, "empty", "\n")
	u := f.unlocked(t, "")

	for name, ref := range map[string]string{
		"empty":            "",
		"parent":           "../other-user/secret",
		"instance keyset":  "../keyset.json",
		"absolute":         filepath.Join(other, "secret"),
		"escaping symlink": "link",
		"directory":        "dir",
		"missing":          "missing",
		"too big":          "big",
		"empty file":       "empty",
		"nul":              "dir/\x00file",
	} {
		if _, err := u.Seal(ctx, objectContext(), []byte(ref)); err == nil {
			t.Errorf("Seal(%s) succeeded", name)
		}
		if _, err := u.Open(ctx, objectContext(), []byte(ref)); err == nil {
			t.Errorf("Open(%s) succeeded", name)
		}
	}
}

// A file its owner made a named pipe is refused at once: opening it would
// wait for a writer, holding the request, or the background task, for ever.
func TestFileStoreDoesNotWaitOnANamedPipe(t *testing.T) {
	f := newFileFixture(t)
	if err := syscall.Mkfifo(filepath.Join(f.userDir(), "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	u := f.unlocked(t, "")

	done := make(chan error, 2)
	go func() {
		_, err := u.Seal(context.Background(), objectContext(), []byte("fifo"))
		done <- err
		_, err = u.Open(context.Background(), objectContext(), []byte("fifo"))
		done <- err
	}()
	for _, op := range []string{"Seal", "Open"} {
		select {
		case err := <-done:
			if err == nil {
				t.Errorf("%s(named pipe) succeeded", op)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s(named pipe) is still waiting", op)
		}
	}
}

// What is entered as a reference may be a secret, typed in by mistake: an
// error never repeats it.
func TestFileStoreErrorsDoNotRepeatTheReference(t *testing.T) {
	f := newFileFixture(t)
	f.write(t, "pa55w0rd-dir/x", "x")
	f.write(t, "pa55w0rd-big", strings.Repeat("x", maxSecretFileSize+1))
	u := f.unlocked(t, "")
	for _, ref := range []string{"pa55w0rd-typed-in", "../pa55w0rd", "/pa55w0rd", "pa55w0rd-dir", "pa55w0rd-big"} {
		_, err := u.Seal(context.Background(), objectContext(), []byte(ref))
		if err == nil {
			t.Fatalf("Seal(%q) succeeded", ref)
		}
		if strings.Contains(err.Error(), "pa55w0rd") {
			t.Errorf("Seal(%q): the error repeats the reference: %v", ref, err)
		}
	}
	if _, err := f.store.Reference([]byte("../pa55w0rd")); err == nil || strings.Contains(err.Error(), "pa55w0rd") {
		t.Errorf("Reference(invalid) = %v, want an error not repeating it", err)
	}
}

// A safe whose stored directory leaves the one of its owner, as a tampered
// database would hold, does not open: Create is not there to check it.
func TestFileStoreUnlockChecksTheStoredDirectory(t *testing.T) {
	ctx := context.Background()
	f := newFileFixture(t)
	for name, keyring := range map[string][]happydns.WrappedKeyset{
		"parent":     {{KEK: KindFile, Params: fileParams("../other-user")}},
		"absolute":   {{KEK: KindFile, Params: fileParams("/etc")}},
		"unreadable": {{KEK: KindFile, Params: json.RawMessage("{")}},
		"no entry":   {{KEK: KindInstance, Params: fileParams("")}},
		"empty":      nil,
	} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindFile, Keyring: keyring}
		if _, err := f.store.Unlock(ctx, safe); err == nil {
			t.Errorf("Unlock(%s) succeeded", name)
		}
	}
}

// A keystore directory that is a symbolic link out of the directory of its
// owner is refused, as a file is.
func TestFileStoreKeystoreDirectoryStaysInItsDirectory(t *testing.T) {
	f := newFileFixture(t)
	other := filepath.Join(f.dir, "other-user")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(f.userDir(), "theirs")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../"+filepath.Base(other), filepath.Join(f.userDir(), "theirs-relative")); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"theirs", "theirs-relative"} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindFile}
		if err := f.store.Create(context.Background(), safe, fileParams(dir)); err == nil {
			t.Errorf("Create(%s) succeeded through a symbolic link out of the directory of its owner", dir)
		}
	}
}

func TestFileStoreRefusesASafeOfAnotherKind(t *testing.T) {
	f := newFileFixture(t)
	safe := f.safe(t, "")
	safe.Kind = KindInstance
	if _, err := f.store.Unlock(context.Background(), safe); err == nil {
		t.Error("a safe of another kind opened")
	}
}

func TestFileKindIsBuiltin(t *testing.T) {
	if _, ok := builtinStores[KindFile]; !ok {
		t.Fatalf("no builtin store %q", KindFile)
	}
	store, err := newFileStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(Creator); !ok {
		t.Error("file is not a Creator")
	}
	if r, ok := store.(Referencing); !ok || r.ReferenceHint() == "" {
		t.Error("file is not Referencing, with a hint")
	}
	if _, ok := store.(StartupChecker); !ok {
		t.Error("file does not check its directory at startup")
	}
	if _, ok := store.(Interactive); ok {
		t.Error("file is Interactive")
	}
	if !opensUnattended(store) {
		t.Error("file does not open unattended")
	}
}

func TestFileKeystoreThroughManager(t *testing.T) {
	ctx := context.Background()
	f := newFileFixture(t)
	f.write(t, "ovh/key", "s3cr3t\n")
	sc := objectContext()
	sc.Owner = f.owner
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage(), Defaults: newMemDefaults()}.WithStores(map[string]StoreFactory{
		KindFile: func(Config) (Store, error) { return newFileStoreIn(f.dir) },
	}))
	if err != nil {
		t.Fatal(err)
	}

	ks, err := m.CreateKeystore(ctx, sc.Owner, KindFile, "My files", fileParams(""))
	if err != nil {
		t.Fatal(err)
	}
	if !ks.References {
		t.Errorf("keystore = %+v, want it keeping references", ks)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("ovh/key")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	if strings.Contains(obj.ApiKey.Token(), "s3cr3t") {
		t.Error("the stored value holds the secret")
	}

	shown := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.ShowReferences(ctx, sc, shown); err != nil || !shown.ApiKey.IsReference() {
		t.Fatalf("ShowReferences = %v, reference shown: %v", err, shown.ApiKey.IsReference())
	}

	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
}
