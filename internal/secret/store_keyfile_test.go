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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/mac"

	"git.happydns.org/happyDomain/model"
)

// keyFileFixture is a keyfile store over a directory of its own, with the
// directory of owner.
type keyFileFixture struct {
	dir   string
	owner happydns.Identifier
	store *keyFileStore
}

func newKeyFileFixture(t *testing.T) keyFileFixture {
	t.Helper()
	f := keyFileFixture{dir: t.TempDir(), owner: happydns.Identifier{0x01, 0x02}}
	if err := os.Mkdir(f.userDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := newKeyFileStoreIn(f.dir)
	if err != nil {
		t.Fatal(err)
	}
	f.store = store.(*keyFileStore)
	return f
}

func (f keyFileFixture) userDir() string {
	return filepath.Join(f.dir, f.owner.String())
}

// keyset writes a new keyset as name in the directory of the owner.
func (f keyFileFixture) keyset(t *testing.T, name string) {
	t.Helper()
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteNewKeysetFile(filepath.Join(f.userDir(), name), h); err != nil {
		t.Fatal(err)
	}
}

func keyFileParams(name string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{keyFileParam: name})
	return b
}

func (f keyFileFixture) newSafe(t *testing.T, name string) *happydns.Safe {
	t.Helper()
	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindKeyFile}
	if err := f.store.Create(context.Background(), safe, keyFileParams(name)); err != nil {
		t.Fatalf("Create(%q): %v", name, err)
	}
	return safe
}

func TestKeyFileStoreNeedsItsDirectory(t *testing.T) {
	if _, err := newKeyFileStoreIn(""); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("newKeyFileStore(no directory) = %v, want ErrStoreUnavailable", err)
	}

	store, err := newKeyFileStoreIn(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.(StartupChecker).CheckStartup(context.Background()); err == nil {
		t.Error("CheckStartup accepted a missing directory")
	}
}

func TestKeyFileStoreOpensUnattended(t *testing.T) {
	f := newKeyFileFixture(t)
	if !opensUnattended(f.store) {
		t.Error("keyfile does not open unattended")
	}
}

func TestKeyFileStoreSealsAndOpens(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	ctx := context.Background()

	safe := f.newSafe(t, "k.json")
	u, err := f.store.Unlock(ctx, safe)
	if err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	sc := SecretContext{}
	sealed, err := u.Seal(ctx, sc, []byte("hunter2"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("hunter2")) {
		t.Error("the sealed value holds the secret in clear")
	}

	// Another Unlock, as after a restart, opens what the first sealed.
	u2, err := f.store.Unlock(ctx, safe)
	if err != nil {
		t.Fatal(err)
	}
	got, err := u2.Open(ctx, sc, sealed)
	if err != nil || string(got) != "hunter2" {
		t.Errorf("Open = %q, %v, want hunter2", got, err)
	}
}

func TestKeyFileStoreStaysLockedWithoutItsFile(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	ctx := context.Background()
	safe := f.newSafe(t, "k.json")

	if err := os.Rename(filepath.Join(f.userDir(), "k.json"), filepath.Join(f.userDir(), "away.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Unlock(ctx, safe); err == nil {
		t.Fatal("Unlock opened a keystore whose file is missing")
	} else if strings.Contains(err.Error(), f.dir) {
		t.Errorf("the error repeats the path: %v", err)
	}

	// Unlocked as soon as the file is back.
	if err := os.Rename(filepath.Join(f.userDir(), "away.json"), filepath.Join(f.userDir(), "k.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Unlock(ctx, safe); err != nil {
		t.Errorf("Unlock with the file back: %v", err)
	}
}

func TestKeyFileStoreRefusesAnotherKeyset(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	safe := f.newSafe(t, "k.json")

	other := filepath.Join(f.userDir(), "k.json")
	if err := os.Remove(other); err != nil {
		t.Fatal(err)
	}
	f.keyset(t, "k.json")

	if _, err := f.store.Unlock(context.Background(), safe); err == nil {
		t.Error("Unlock opened a keystore with a keyset that did not wrap its key")
	}
}

func TestKeyFileStoreCreateChecksTheFile(t *testing.T) {
	f := newKeyFileFixture(t)
	if err := os.WriteFile(filepath.Join(f.userDir(), "junk.json"), []byte("not a keyset"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"", "missing.json", "junk.json"} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindKeyFile}
		if err := f.store.Create(context.Background(), safe, keyFileParams(name)); err == nil {
			t.Errorf("Create(%q) accepted it", name)
		}
		if len(safe.Keyring) != 0 {
			t.Errorf("Create(%q) failed but filled the keyring", name)
		}
	}
}

func TestKeyFileStoreStaysInItsDirectory(t *testing.T) {
	f := newKeyFileFixture(t)

	// A keyset next to the directory of the owner, such as the one of the
	// instance, and a link to it from inside.
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(f.dir, "ks.json")
	if err := WriteNewKeysetFile(outside, h); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(f.userDir(), "link.json")); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"../ks.json", outside, "link.json", "a/../../ks.json"} {
		safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindKeyFile}
		if err := f.store.Create(context.Background(), safe, keyFileParams(name)); err == nil {
			t.Errorf("Create(%q) reached outside the directory of the owner", name)
		}
	}
}

func TestKeyFileStoreBindsTheKeyToItsSafe(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	safe := f.newSafe(t, "k.json")

	moved := *safe
	moved.Id = happydns.Identifier{0x06}
	if _, err := f.store.Unlock(context.Background(), &moved); err == nil {
		t.Error("Unlock opened the key of a safe copied under another identifier")
	}
}

func TestKeyFileStoreRefusesASafeOfAnotherKind(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	safe := f.newSafe(t, "k.json")

	safe.Kind = KindFile
	if _, err := f.store.Unlock(context.Background(), safe); err == nil {
		t.Error("Unlock opened a safe of another kind")
	}
}

func TestKeyFileKindIsBuiltin(t *testing.T) {
	if _, ok := builtinStores[KindKeyFile]; !ok {
		t.Fatal("keyfile is not a builtin store")
	}
	for _, kind := range DefaultStores() {
		if kind == KindKeyFile {
			t.Error("keyfile is enabled by default: it reads files of the server")
		}
	}

	store, err := newKeyFileStoreIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(Creator); !ok {
		t.Error("keyfile is not a Creator")
	}
	if _, ok := store.(Referencing); ok {
		t.Error("keyfile is Referencing: it encrypts, it keeps no references")
	}
	if _, ok := store.(StartupChecker); !ok {
		t.Error("keyfile does not check its directory at startup")
	}
	if _, ok := store.(Interactive); ok {
		t.Error("keyfile is Interactive")
	}
}

// The key is bound to the owner too: the same keyset file in the directory
// of another owner does not open a safe moved to them.
func TestKeyFileStoreBindsTheKeyToItsOwner(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	safe := f.newSafe(t, "k.json")

	other := happydns.Identifier{0x03, 0x04}
	if err := os.Mkdir(filepath.Join(f.dir, other.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(f.userDir(), "k.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, other.String(), "k.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	moved := *safe
	moved.Owner = other
	if _, err := f.store.Unlock(context.Background(), &moved); err == nil {
		t.Error("Unlock opened the key of a safe moved to another owner")
	}
}

func TestKeyFileStoreRefusesAKeysetNotForEncryption(t *testing.T) {
	f := newKeyFileFixture(t)
	h, err := keyset.NewHandle(mac.HMACSHA256Tag256KeyTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteNewKeysetFile(filepath.Join(f.userDir(), "mac.json"), h); err != nil {
		t.Fatal(err)
	}

	safe := &happydns.Safe{Id: happydns.Identifier{0x05}, Owner: f.owner, Kind: KindKeyFile}
	if err := f.store.Create(context.Background(), safe, keyFileParams("mac.json")); !errors.Is(err, errInvalidKeysetFile) {
		t.Errorf("Create(a MAC keyset) = %v, want errInvalidKeysetFile", err)
	}
}

// Parameters come from the database: Unlock checks them again.
func TestKeyFileStoreUnlockChecksItsParameters(t *testing.T) {
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	ctx := context.Background()

	for name, params := range map[string]json.RawMessage{
		"outside":    keyFileParams("../k.json"),
		"unreadable": json.RawMessage(`not json`),
	} {
		safe := f.newSafe(t, "k.json")
		safe.Keyring[0].Params = params
		if _, err := f.store.Unlock(ctx, safe); err == nil {
			t.Errorf("Unlock opened a safe with %s parameters", name)
		}
	}

	safe := f.newSafe(t, "k.json")
	safe.Keyring[0].KEK = KindInstance
	if _, err := f.store.Unlock(ctx, safe); err == nil {
		t.Error("Unlock opened a safe without a keyfile entry")
	}
}

func TestKeyFileKeystoreThroughManager(t *testing.T) {
	ctx := context.Background()
	f := newKeyFileFixture(t)
	f.keyset(t, "k.json")
	sc := objectContext()
	sc.Owner = f.owner
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage(), Defaults: newMemDefaults()}.WithStores(map[string]StoreFactory{
		KindKeyFile: func(Config) (Store, error) { return newKeyFileStoreIn(f.dir) },
	}))
	if err != nil {
		t.Fatal(err)
	}

	ks, err := m.CreateKeystore(ctx, sc.Owner, KindKeyFile, "My keyset", keyFileParams("k.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ks.References {
		t.Errorf("keystore = %+v, want it encrypting", ks)
	}

	obj := &managedObject{ApiKey: happydns.NewSecret("s3cr3t")}
	if err := m.SealObjectInto(ctx, sc, obj, Destination{Keystore: ks.Ref}); err != nil {
		t.Fatalf("SealObjectInto: %v", err)
	}
	if strings.Contains(obj.ApiKey.Token(), "s3cr3t") {
		t.Error("the stored value holds the secret")
	}

	back := &managedObject{ApiKey: sealedFromStorage(t, obj.ApiKey.Token())}
	if err := m.OpenObject(ctx, sc, back); err != nil || back.ApiKey.Reveal() != "s3cr3t" {
		t.Errorf("OpenObject = %q, %v", back.ApiKey.Reveal(), err)
	}
}
