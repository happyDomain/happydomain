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

package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

func runKeyset(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runSecretKeyset(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestSecretKeysetGenerate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")

	if code, _, stderr := runKeyset(t, "generate", path); code != 0 {
		t.Fatalf("generate exited %d: %s", code, stderr)
	}

	h, err := secret.ReadKeysetFile(path)
	if err != nil {
		t.Fatalf("the generated file is not a keyset: %v", err)
	}
	if n := len(secret.DescribeKeyset(h)); n != 1 {
		t.Errorf("%d keys, want 1", n)
	}

	before, _ := os.ReadFile(path)
	if code, _, _ := runKeyset(t, "generate", path); code == 0 {
		t.Error("generate overwrote an existing keyset")
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Error("the existing keyset was changed")
	}
}

func TestSecretKeysetRotate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	if code, _, stderr := runKeyset(t, "generate", path); code != 0 {
		t.Fatal(stderr)
	}

	key, _ := secret.LoadInstanceKey(path)
	oldPrimary := key.PrimaryKeyId()
	store := &memCheck{}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	if code, _, stderr := runKeyset(t, "rotate", path); code != 0 {
		t.Fatalf("rotate exited %d: %s", code, stderr)
	}

	rotated, err := secret.LoadInstanceKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.PrimaryKeyId() == oldPrimary {
		t.Error("rotate did not change the primary key")
	}
	if err := rotated.VerifyCheck(store); err != nil {
		t.Errorf("the rotated keyset does not open what the old key sealed: %v", err)
	}
}

func TestSecretKeysetInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	if code, _, stderr := runKeyset(t, "generate", path); code != 0 {
		t.Fatal(stderr)
	}
	if code, _, stderr := runKeyset(t, "rotate", path); code != 0 {
		t.Fatal(stderr)
	}

	code, stdout, stderr := runKeyset(t, "info", path)
	if code != 0 {
		t.Fatalf("info exited %d: %s", code, stderr)
	}

	h, _ := secret.ReadKeysetFile(path)
	if !strings.Contains(stdout, "primary") || strings.Count(stdout, "ENABLED") != 2 {
		t.Errorf("info = %q, want both keys and the primary", stdout)
	}
	for _, key := range insecurecleartextkeyset.KeysetMaterial(h).Key {
		value := key.GetKeyData().GetValue()
		if strings.Contains(stdout+stderr, base64.StdEncoding.EncodeToString(value)) || strings.Contains(stdout+stderr, string(value)) {
			t.Fatalf("info shows key material: %s", stdout)
		}
	}
}

func TestSecretKeysetUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"generate"}, {"bogus", "file"}, {"info", "a", "b"}} {
		if code, _, _ := runKeyset(t, args...); code == 0 {
			t.Errorf("secret-keyset %v succeeded, want a usage error", args)
		}
	}
}

type memCheck struct{ record []byte }

func (m *memCheck) GetSecretCheck() ([]byte, error) {
	if m.record == nil {
		return nil, happydns.ErrNotFound
	}
	return m.record, nil
}

func (m *memCheck) PutSecretCheck(b []byte) error {
	m.record = b
	return nil
}
