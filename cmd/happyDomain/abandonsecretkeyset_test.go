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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

type abandonStore struct {
	*secrettest.Safes
	*memCheck
}

// sealedStore returns a database used with the keyset at path, holding a
// secret sealed for each of two owners.
func sealedStore(t *testing.T, path string) *abandonStore {
	t.Helper()
	if code, _, stderr := runKeyset(t, "generate", path); code != 0 {
		t.Fatal(stderr)
	}
	key, err := secret.LoadInstanceKey(path)
	if err != nil {
		t.Fatal(err)
	}
	store := &abandonStore{Safes: secrettest.NewSafes(), memCheck: &memCheck{}}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		s := happydns.NewSecret("v")
		if err := m.SealSecret(context.Background(), secret.SecretContext{Owner: owner, ObjectType: "provider", ObjectId: "p", Field: "ApiKey"}, &s); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func runAbandon(t *testing.T, store secret.Storage, keyPath, stdin string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := abandonSecretKeyset(store, keyPath, strings.NewReader(stdin), &out)
	return code, out.String()
}

func safesLeft(t *testing.T, store *abandonStore) int {
	t.Helper()
	n, err := secret.CountInstanceSafes(store)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAbandonSecretKeysetConfirmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	store := sealedStore(t, path)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	code, out := runAbandon(t, store, path, "abandon\n")
	if code != 0 {
		t.Fatalf("abandon exited %d: %s", code, out)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("the prompt does not say how many users' keys go: %s", out)
	}
	if n := safesLeft(t, store); n != 0 {
		t.Errorf("%d safes left", n)
	}
	if store.record != nil {
		t.Error("the check record was kept")
	}
}

func TestAbandonSecretKeysetNotConfirmed(t *testing.T) {
	for _, answer := range []string{"", "\n", "yes\n", "ABANDON\n", "abandonn\n"} {
		path := filepath.Join(t.TempDir(), "keyset.json")
		store := sealedStore(t, path)
		_ = os.Remove(path)

		if code, out := runAbandon(t, store, path, answer); code == 0 {
			t.Errorf("answer %q: abandon succeeded: %s", answer, out)
		}
		if n := safesLeft(t, store); n != 2 || store.record == nil {
			t.Errorf("answer %q: deleted without confirmation: %d safes left, record %v", answer, n, store.record)
		}
	}
}

// The keyset configured still opens the check record: it is not lost, and
// nothing is asked nor deleted.
func TestAbandonSecretKeysetRefusesKeysetNotLost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyset.json")
	store := sealedStore(t, path)

	code, out := runAbandon(t, store, path, "abandon\n")
	if code == 0 {
		t.Fatalf("abandon with a working keyset succeeded: %s", out)
	}
	if strings.Contains(out, "Type abandon") {
		t.Errorf("asked for a confirmation before refusing: %s", out)
	}
	if n := safesLeft(t, store); n != 2 || store.record == nil {
		t.Errorf("deleted with a working keyset: %d safes left", n)
	}
}

// A keyset file that is no longer a keyset, or another keyset, is as good
// as lost.
func TestAbandonSecretKeysetWithUnusableKeyset(t *testing.T) {
	for name, replace := range map[string]func(path string){
		"corrupted": func(path string) { _ = os.WriteFile(path, []byte("not a keyset"), 0o600) },
		"another": func(path string) {
			_ = os.Remove(path)
			if code, _, stderr := runKeyset(t, "generate", path); code != 0 {
				t.Fatal(stderr)
			}
		},
	} {
		path := filepath.Join(t.TempDir(), "keyset.json")
		store := sealedStore(t, path)
		replace(path)

		if code, out := runAbandon(t, store, path, "abandon\n"); code != 0 {
			t.Errorf("%s keyset: abandon exited %d: %s", name, code, out)
		}
		if n := safesLeft(t, store); n != 0 {
			t.Errorf("%s keyset: %d safes left", name, n)
		}
	}
}

func TestAbandonSecretKeysetWithoutKeysetConfigured(t *testing.T) {
	store := sealedStore(t, filepath.Join(t.TempDir(), "keyset.json"))
	if code, out := runAbandon(t, store, "", "abandon\n"); code != 0 {
		t.Fatalf("abandon exited %d: %s", code, out)
	}
	if n := safesLeft(t, store); n != 0 {
		t.Errorf("%d safes left", n)
	}
}

func TestAbandonSecretKeysetNothingToDo(t *testing.T) {
	store := &abandonStore{Safes: secrettest.NewSafes(), memCheck: &memCheck{}}
	if code, out := runAbandon(t, store, "", ""); code != 0 {
		t.Errorf("abandon on a database without safes exited %d: %s", code, out)
	}
}
