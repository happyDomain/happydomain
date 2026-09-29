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
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"

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

// rotateInstanceKeyset returns h with a new AES256-GCM key as primary, the
// previous keys kept. happyDomain does not rotate yet, but an administrator can
// with tinkey: what is built on the keyset has to cope with several keys.
func rotateInstanceKeyset(h *keyset.Handle) (*keyset.Handle, error) {
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
	h, _ = rotateInstanceKeyset(h)

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
	rotated, _ := rotateInstanceKeyset(h)
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

func TestStartupCheck(t *testing.T) {
	h, _ := GenerateInstanceKeyset()
	key, _ := NewInstanceKey(h)

	if err := StartupCheck(PolicyInstance, nil, &memCheckStorage{}); err == nil {
		t.Error("the instance policy without a keyset must refuse to start")
	}
	if err := StartupCheck(PolicyPlaintext, nil, &memCheckStorage{}); err != nil {
		t.Errorf("plaintext without a keyset: %v", err)
	}
	if err := StartupCheck(PolicyInstance, key, &memCheckStorage{}); err != nil {
		t.Errorf("instance with a keyset: %v", err)
	}

	other, _ := GenerateInstanceKeyset()
	okey, _ := NewInstanceKey(other)
	store := &memCheckStorage{}
	_ = key.VerifyCheck(store)
	if err := StartupCheck(PolicyPlaintext, okey, store); err == nil {
		t.Error("a keyset that does not open the check record must refuse to start")
	}
}
