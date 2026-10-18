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
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/mac"

	"git.happydns.org/happyDomain/model"
)

// memCheckStorage keeps the check record in memory.
type memCheckStorage struct {
	record []byte
	puts   int
}

func (s *memCheckStorage) GetSecretCheck() ([]byte, error) {
	if s.record == nil {
		return nil, happydns.ErrNotFound
	}
	return s.record, nil
}

func (s *memCheckStorage) PutSecretCheck(b []byte) error {
	s.record = append([]byte(nil), b...)
	s.puts++
	return nil
}

func TestGenerateInstanceKeyset(t *testing.T) {
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}

	info := DescribeKeyset(h)
	if len(info) != 1 {
		t.Fatalf("%d keys, want 1", len(info))
	}
	if !info[0].Primary || info[0].Status != "ENABLED" {
		t.Errorf("key = %+v, want the enabled primary", info[0])
	}
	if !strings.Contains(info[0].Type, "AesGcmKey") {
		t.Errorf("key type = %q, want AES-GCM", info[0].Type)
	}
}

func TestWriteNewKeysetFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")

	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}

	if err := WriteNewKeysetFile(path, h); err != nil {
		t.Fatalf("WriteNewKeysetFile: %v", err)
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", st.Mode().Perm())
	}

	back, err := ReadKeysetFile(path)
	if err != nil {
		t.Fatalf("ReadKeysetFile: %v", err)
	}
	if back.KeysetInfo().PrimaryKeyId != h.KeysetInfo().PrimaryKeyId {
		t.Error("the keyset read back differs")
	}

	// Never overwrite: losing a keyset loses every secret it protects.
	other, _ := GenerateInstanceKeyset()
	if err := WriteNewKeysetFile(path, other); err == nil {
		t.Error("WriteNewKeysetFile overwrote an existing file")
	}
	again, err := ReadKeysetFile(path)
	if err != nil || again.KeysetInfo().PrimaryKeyId != h.KeysetInfo().PrimaryKeyId {
		t.Error("the existing keyset was changed")
	}
}

func TestDescribeKeysetShowsNoKeyMaterial(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	h, _ = RotateInstanceKeyset(h)

	var buf bytes.Buffer
	for _, k := range DescribeKeyset(h) {
		buf.WriteString(k.String())
		buf.WriteByte('\n')
	}

	for _, key := range insecurecleartextkeyset.KeysetMaterial(h).Key {
		value := key.GetKeyData().GetValue()
		for _, enc := range []string{string(value), base64.StdEncoding.EncodeToString(value), base64.RawStdEncoding.EncodeToString(value)} {
			if strings.Contains(buf.String(), enc) {
				t.Fatalf("keyset description shows key material:\n%s", buf.String())
			}
		}
	}
}

func TestKeysetFileTooOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	h, _ := GenerateInstanceKeyset()
	if err := WriteNewKeysetFile(path, h); err != nil {
		t.Fatal(err)
	}

	if open, err := KeysetFileTooOpen(path); err != nil || open {
		t.Errorf("KeysetFileTooOpen(0600) = %v, %v; want false", open, err)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if open, err := KeysetFileTooOpen(path); err != nil || !open {
		t.Errorf("KeysetFileTooOpen(0644) = %v, %v; want true", open, err)
	}
}

func TestVerifyCheckCreatesThenVerifies(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, err := NewInstanceKey(h)
	if err != nil {
		t.Fatal(err)
	}

	store := &memCheckStorage{}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatalf("VerifyCheck on an empty store: %v", err)
	}
	if store.puts != 1 || len(store.record) == 0 {
		t.Fatal("VerifyCheck did not create the check record")
	}

	if err := key.VerifyCheck(store); err != nil {
		t.Errorf("VerifyCheck with its own record: %v", err)
	}
	if store.puts != 1 {
		t.Error("VerifyCheck rewrote a valid record")
	}

	// After a rotation, the record sealed by the old primary still opens.
	rotated, _ := RotateInstanceKeyset(h)
	rkey, _ := NewInstanceKey(rotated)
	if err := rkey.VerifyCheck(store); err != nil {
		t.Errorf("VerifyCheck after rotation: %v", err)
	}
}

func TestVerifyCheckRefusesAnotherKey(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)
	store := &memCheckStorage{}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	other, _ := GenerateInstanceKeyset()
	okey, _ := NewInstanceKey(other)
	err := okey.VerifyCheck(store)
	if !errors.Is(err, ErrWrongInstanceKey) {
		t.Errorf("VerifyCheck with another keyset = %v, want ErrWrongInstanceKey", err)
	}
	if store.puts != 1 {
		t.Error("VerifyCheck replaced the record of another keyset")
	}
}

// A committed test keyset opens a committed check record: a change to the
// check format, or to how keysets are read, fails here.
// None of these may replace the stored record: it is how a wrong keyset is
// told apart.
func TestVerifyCheckRefusesTamperedRecord(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)

	good := &memCheckStorage{}
	if err := key.VerifyCheck(good); err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte(nil), good.record...)
	tampered[len(tampered)-1] ^= 0x01

	otherAD, err := key.aead.Encrypt([]byte(checkPlaintext), []byte("hds:1:safe"))
	if err != nil {
		t.Fatal(err)
	}
	otherPlaintext, err := key.aead.Encrypt([]byte("something else"), []byte(checkAssociatedData))
	if err != nil {
		t.Fatal(err)
	}

	for name, record := range map[string][]byte{
		"tampered":        tampered,
		"truncated":       good.record[:len(good.record)-1],
		"empty":           {},
		"other AD":        otherAD,
		"other plaintext": otherPlaintext,
	} {
		store := &memCheckStorage{record: record}
		if err := key.VerifyCheck(store); !errors.Is(err, ErrWrongInstanceKey) {
			t.Errorf("%s: VerifyCheck = %v, want ErrWrongInstanceKey", name, err)
		}
		if store.puts != 0 {
			t.Errorf("%s: VerifyCheck replaced a record it could not open", name)
		}
	}
}

type failingCheckStorage struct{ memCheckStorage }

func (*failingCheckStorage) GetSecretCheck() ([]byte, error) {
	return nil, errors.New("storage unavailable")
}

// A read error is not a missing record: writing a fresh one would accept any
// keyset.
func TestVerifyCheckStorageError(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)

	store := &failingCheckStorage{}
	err := key.VerifyCheck(store)
	if err == nil || errors.Is(err, ErrWrongInstanceKey) {
		t.Errorf("VerifyCheck on a failing store = %v, want the storage error", err)
	}
	if store.puts != 0 {
		t.Error("VerifyCheck created a check record over a storage error")
	}
}

func TestLoadInstanceKeyRefusesBadFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	macPath := filepath.Join(dir, "mac.json")
	mh, err := keyset.NewHandle(mac.HMACSHA256Tag256KeyTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteNewKeysetFile(macPath, mh); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"missing":     filepath.Join(dir, "missing.json"),
		"empty":       write("empty.json", ""),
		"not json":    write("garbage.json", "not a keyset"),
		"no key":      write("nokey.json", `{"primaryKeyId":1,"key":[]}`),
		"not an AEAD": macPath,
	} {
		if key, err := LoadInstanceKey(path); err == nil || key != nil {
			t.Errorf("%s: LoadInstanceKey = %v, %v; want an error", name, key, err)
		}
	}
}

func TestCheckRecordGolden(t *testing.T) {
	key, err := LoadInstanceKey("testdata/instance-keyset.json")
	if err != nil {
		t.Fatalf("LoadInstanceKey: %v", err)
	}

	record, err := os.ReadFile("testdata/check-record.b64")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(record)))
	if err != nil {
		t.Fatal(err)
	}

	store := &memCheckStorage{record: raw}
	if err := key.VerifyCheck(store); err != nil {
		t.Errorf("the committed keyset does not open the committed check record: %v", err)
	}
}

// startOn starts a Manager under policy, with key when not nil, on store.
func startOn(policy Policy, key *InstanceKey, store *memStartupStorage) error {
	_, err := Start(context.Background(), Config{Policy: policy, InstanceKey: key, Safes: store, Checks: store})
	return err
}

// What to do is told, not only what is wrong: these are the errors an
// administrator meets first.
func TestStartExplainsTheMissingKeyset(t *testing.T) {
	err := startOn(PolicyInstance, nil, &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()})
	if err == nil || !strings.Contains(err.Error(), "secret-keyset generate") {
		t.Errorf("instance policy without a keyset = %v, want it to tell how to generate one", err)
	}

	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}
	if err := store.CreateSafe(newTestSafe(t, happydns.Identifier{0x01})); err != nil {
		t.Fatal(err)
	}
	err = startOn(PolicyPlaintext, nil, store)
	if err == nil || !strings.Contains(err.Error(), "-secret-keyset-file") {
		t.Errorf("instance safes without a keyset = %v, want it to tell how to configure one", err)
	}
}

// Without its check storage, the instance store cannot tell whether its
// keyset is the right one: starting is refused.
func TestStartRequiresTheCheckStorage(t *testing.T) {
	_, err := Start(context.Background(), Config{Policy: PolicyInstance, InstanceKey: testInstanceKey(t), Safes: newMemSafeStorage()})
	if err == nil {
		t.Error("Start of the instance store without a check storage succeeded")
	}
}

func TestStart(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)

	if err := startOn(PolicyInstance, nil, &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}); err == nil {
		t.Error("the instance policy without a keyset must refuse to start")
	}
	if err := startOn(PolicyPlaintext, nil, &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}); err != nil {
		t.Errorf("plaintext without a keyset: %v", err)
	}
	if err := startOn(PolicyInstance, key, &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}); err != nil {
		t.Errorf("instance with a keyset: %v", err)
	}

	other, _ := GenerateInstanceKeyset()
	okey, _ := NewInstanceKey(other)
	store := &memCheckStorage{}
	_ = key.VerifyCheck(store)
	if err := startOn(PolicyPlaintext, okey, &memStartupStorage{memCheckStorage: store, memSafeStorage: newMemSafeStorage()}); err == nil {
		t.Error("a keyset that does not open the check record must refuse to start")
	}
}

func TestRotateKeepsOldKeysDecrypting(t *testing.T) {
	h, err := GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	oldPrimary := h.KeysetInfo().PrimaryKeyId

	a, _ := aead.New(h)
	ct, err := a.Encrypt([]byte("sealed before rotation"), []byte("ad"))
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := RotateInstanceKeyset(h)
	if err != nil {
		t.Fatalf("RotateInstanceKeyset: %v", err)
	}

	info := DescribeKeyset(rotated)
	if len(info) != 2 {
		t.Fatalf("%d keys after rotation, want 2", len(info))
	}
	if rotated.KeysetInfo().PrimaryKeyId == oldPrimary {
		t.Error("the primary key did not change")
	}

	ra, _ := aead.New(rotated)
	pt, err := ra.Decrypt(ct, []byte("ad"))
	if err != nil || string(pt) != "sealed before rotation" {
		t.Errorf("rotated keyset cannot open what the old key sealed: %v", err)
	}
}

func TestReplaceKeysetFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")

	h, _ := GenerateInstanceKeyset()
	if err := WriteNewKeysetFile(path, h); err != nil {
		t.Fatal(err)
	}

	rotated, _ := RotateInstanceKeyset(h)
	if err := ReplaceKeysetFile(path, rotated); err != nil {
		t.Fatalf("ReplaceKeysetFile: %v", err)
	}

	back, err := ReadKeysetFile(path)
	if err != nil || len(DescribeKeyset(back)) != 2 {
		t.Errorf("ReadKeysetFile after replace = %v", err)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", st.Mode().Perm())
	}

	if err := ReplaceKeysetFile(filepath.Join(t.TempDir(), "absent.json"), rotated); err == nil {
		t.Error("ReplaceKeysetFile created a file that did not exist")
	}
}

// A keyset kept in a secrets directory and linked from the configuration:
// rotating updates the keyset, and the link keeps pointing at it.
func TestReplaceKeysetFileThroughSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "secrets", "keyset.json")
	if err := os.Mkdir(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	h, _ := GenerateInstanceKeyset()
	if err := WriteNewKeysetFile(target, h); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "keyset.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	rotated, _ := RotateInstanceKeyset(h)
	if err := ReplaceKeysetFile(link, rotated); err != nil {
		t.Fatalf("ReplaceKeysetFile: %v", err)
	}

	if st, err := os.Lstat(link); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symbolic link was replaced by a file (%v)", err)
	}
	back, err := ReadKeysetFile(target)
	if err != nil || len(DescribeKeyset(back)) != 2 {
		t.Errorf("the link target was not updated: %v", err)
	}
}

// The service may read its keyset through its group (root:happydomain 0640):
// rotating, for instance through sudo, must not change who can read it.
func TestReplaceKeysetFileKeepsAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	h, _ := GenerateInstanceKeyset()
	if err := WriteNewKeysetFile(path, h); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}

	// A group the test user belongs to, other than its primary one, shows
	// the group is carried over rather than taken from the process.
	wantGid := os.Getgid()
	if groups, err := os.Getgroups(); err == nil {
		for _, g := range groups {
			if g != wantGid {
				if err := os.Chown(path, -1, g); err == nil {
					wantGid = g
				}
				break
			}
		}
	}

	rotated, _ := RotateInstanceKeyset(h)
	if err := ReplaceKeysetFile(path, rotated); err != nil {
		t.Fatalf("ReplaceKeysetFile: %v", err)
	}

	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o640 {
		t.Errorf("mode = %o, want the original 640", st.Mode().Perm())
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		if int(sys.Uid) != os.Getuid() || int(sys.Gid) != wantGid {
			t.Errorf("owner = %d:%d, want %d:%d", sys.Uid, sys.Gid, os.Getuid(), wantGid)
		}
	}
}

// Once the operator has rewrapped every safe, the report says the old key
// wraps nothing and can be removed. The check record must not keep needing
// it: VerifyCheck moves it under the primary key.
func TestCiphertextKeyId(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	a, _ := aead.New(h)
	ct, err := a.Encrypt([]byte("x"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := ciphertextKeyId(ct); err != nil || id != h.KeysetInfo().PrimaryKeyId {
		t.Errorf("ciphertextKeyId = %d, %v; want %d", id, err, h.KeysetInfo().PrimaryKeyId)
	}

	for _, bad := range [][]byte{nil, {0x01, 0, 0, 0}, {0x00, 0, 0, 0, 1}} {
		if _, err := ciphertextKeyId(bad); err == nil {
			t.Errorf("ciphertextKeyId(%x) accepted a ciphertext without a TINK prefix", bad)
		}
	}
}

func TestVerifyCheckResealsUnderPrimary(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldPrimary := h.KeysetInfo().PrimaryKeyId
	key, _ := NewInstanceKey(h)
	store := &memCheckStorage{}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	rotated, _ := RotateInstanceKeyset(h)
	rkey, _ := NewInstanceKey(rotated)
	if err := rkey.VerifyCheck(store); err != nil {
		t.Fatalf("VerifyCheck after rotation: %v", err)
	}
	if store.puts != 2 {
		t.Fatalf("%d writes, want the record resealed once under the new primary", store.puts)
	}
	if id, err := ciphertextKeyId(store.record); err != nil || id != rkey.PrimaryKeyId() {
		t.Errorf("record sealed under key %d (%v), want the primary %d", id, err, rkey.PrimaryKeyId())
	}

	// Already under the primary: nothing to rewrite.
	if err := rkey.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}
	if store.puts != 2 {
		t.Error("VerifyCheck rewrote a record already under the primary key")
	}

	// The old key removed, the keyset still starts.
	m := keyset.NewManagerFromHandle(rotated)
	if err := m.Disable(oldPrimary); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(oldPrimary); err != nil {
		t.Fatal(err)
	}
	pruned, err := m.Handle()
	if err != nil {
		t.Fatal(err)
	}
	pkey, _ := NewInstanceKey(pruned)
	if err := pkey.VerifyCheck(store); err != nil {
		t.Errorf("VerifyCheck once the old key is removed: %v", err)
	}
}

func TestDeleteInstanceKey(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	oldPrimary := h.KeysetInfo().PrimaryKeyId
	rotated, err := RotateInstanceKeyset(h)
	if err != nil {
		t.Fatal(err)
	}

	got, err := DeleteInstanceKey(rotated, oldPrimary)
	if err != nil {
		t.Fatalf("DeleteInstanceKey: %v", err)
	}
	info := DescribeKeyset(got)
	if len(info) != 1 || info[0].Id == oldPrimary || !info[0].Primary {
		t.Errorf("unexpected keys after deletion: %v", info)
	}
	if len(DescribeKeyset(rotated)) != 2 {
		t.Error("the original handle was modified")
	}
}

func TestDeleteInstanceKeyRefusals(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	rotated, _ := RotateInstanceKeyset(h)

	if _, err := DeleteInstanceKey(rotated, rotated.KeysetInfo().PrimaryKeyId); err == nil {
		t.Error("the primary key was deleted")
	}
	if _, err := DeleteInstanceKey(rotated, 12345); err == nil {
		t.Error("an unknown key was deleted")
	}
}
