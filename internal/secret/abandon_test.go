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
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// lostKeysetStore returns a database used with a keyset now lost: two
// owners each holding a secret sealed in their safe, and the check record.
func lostKeysetStore(t *testing.T) (*memStartupStorage, map[string]string) {
	t.Helper()
	key := testInstanceKey(t)
	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}

	tokens := map[string]string{}
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		sc := objectContext()
		sc.Owner = owner
		tokens[owner.String()] = sealOne(t, m, sc, "lost")
	}
	return store, tokens
}

func TestCountInstanceSafes(t *testing.T) {
	store, _ := lostKeysetStore(t)
	if n, err := CountInstanceSafes(store); err != nil || n != 2 {
		t.Errorf("CountInstanceSafes = %d, %v; want 2", n, err)
	}
	if n, err := CountInstanceSafes(newMemSafeStorage()); err != nil || n != 0 {
		t.Errorf("CountInstanceSafes on an empty store = %d, %v; want 0", n, err)
	}
}

// A safe record that does not decode is not counted, nor does it fail the
// count: abandoning leaves it as it is.
func TestCountInstanceSafesSkipsDamagedRecords(t *testing.T) {
	store, _ := lostKeysetStore(t)
	store.damaged["safe-broken"] = errors.New("does not decode")

	if n, err := CountInstanceSafes(store); err != nil || n != 2 {
		t.Errorf("CountInstanceSafes = %d, %v; want 2", n, err)
	}
}

// A damaged safe record is left by abandoning: it must not keep happyDomain
// from starting again.
func TestAbandonLetsStartAgainDespiteDamagedRecords(t *testing.T) {
	store, _ := lostKeysetStore(t)
	store.damaged["safe-broken"] = errors.New("does not decode")

	if _, err := AbandonInstanceKeyset(store, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.damaged) != 1 {
		t.Errorf("%d damaged records left, want the one left as it is", len(store.damaged))
	}
	if err := startOn(PolicyPlaintext, nil, store); err != nil {
		t.Errorf("plaintext without a keyset after abandoning: %v", err)
	}
}

func TestAbandonDeletesSafesAndCheckRecord(t *testing.T) {
	store, _ := lostKeysetStore(t)

	n, err := AbandonInstanceKeyset(store, nil)
	if err != nil || n != 2 {
		t.Fatalf("AbandonInstanceKeyset = %d, %v; want 2 deleted", n, err)
	}
	if len(store.safes) != 0 {
		t.Errorf("%d safes left", len(store.safes))
	}
	if store.record != nil {
		t.Error("the check record was kept: a new keyset would be refused at startup")
	}
}

// Abandoning is what lets happyDomain start again, without a keyset or with
// a new one.
func TestAbandonLetsStartAgain(t *testing.T) {
	store, _ := lostKeysetStore(t)
	if err := startOn(PolicyPlaintext, nil, store); err == nil {
		t.Fatal("the lost keyset store started without a keyset: the test proves nothing")
	}

	if _, err := AbandonInstanceKeyset(store, nil); err != nil {
		t.Fatal(err)
	}

	if err := startOn(PolicyPlaintext, nil, store); err != nil {
		t.Errorf("plaintext without a keyset after abandoning: %v", err)
	}
	if err := startOn(PolicyInstance, testInstanceKey(t), store); err != nil {
		t.Errorf("instance with a new keyset after abandoning: %v", err)
	}
}

// The secrets sealed under the lost keyset are reported unreadable, and their
// owners can enter them again, sealed in a new safe.
func TestAbandonedSecretsCanBeEnteredAgain(t *testing.T) {
	store, tokens := lostKeysetStore(t)
	if _, err := AbandonInstanceKeyset(store, nil); err != nil {
		t.Fatal(err)
	}

	newKey := testInstanceKey(t)
	if err := startOn(PolicyInstance, newKey, store); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: newKey, Safes: store})
	if err != nil {
		t.Fatal(err)
	}

	sc := objectContext()
	sc.Owner = happydns.Identifier{0x01}

	old := &managedObject{ApiKey: sealedFromStorage(t, tokens[sc.Owner.String()])}
	if err := m.OpenObject(context.Background(), sc, old); !errors.Is(err, ErrUnknownSafe) {
		t.Errorf("opening an abandoned secret = %v, want ErrUnknownSafe", err)
	}

	token := sealOne(t, m, sc, "entered again")
	again := &managedObject{ApiKey: sealedFromStorage(t, token)}
	if err := m.OpenObject(context.Background(), sc, again); err != nil || again.ApiKey.Reveal() != "entered again" {
		t.Errorf("a secret entered again does not open: %v", err)
	}
}

// A keyset that still opens the check record is not lost: abandoning would
// throw away credentials that can still be read.
func TestAbandonRefusesKeysetNotLost(t *testing.T) {
	key := testInstanceKey(t)
	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	sealOne(t, m, objectContext(), "still readable")
	record := store.record

	if _, err := AbandonInstanceKeyset(store, key); !errors.Is(err, ErrKeysetNotLost) {
		t.Errorf("AbandonInstanceKeyset with the right keyset = %v, want ErrKeysetNotLost", err)
	}
	if len(store.safes) != 1 || string(store.record) != string(record) {
		t.Error("AbandonInstanceKeyset refused but deleted something")
	}
}

// Another keyset configured, such as a new one generated already, does not
// stand in the way.
func TestAbandonWithAnotherKeyset(t *testing.T) {
	store, _ := lostKeysetStore(t)
	if n, err := AbandonInstanceKeyset(store, testInstanceKey(t)); err != nil || n != 2 {
		t.Errorf("AbandonInstanceKeyset with another keyset = %d, %v; want 2 deleted", n, err)
	}
}

func TestOpensCheckRecord(t *testing.T) {
	key := testInstanceKey(t)
	store := &memCheckStorage{}

	if opens, err := key.OpensCheckRecord(store); err != nil || opens {
		t.Errorf("without a check record = %v, %v; want false", opens, err)
	}
	if store.record != nil {
		t.Error("OpensCheckRecord created the check record")
	}

	_ = key.VerifyCheck(store)
	puts := store.puts
	if opens, err := key.OpensCheckRecord(store); err != nil || !opens {
		t.Errorf("with its own check record = %v, %v; want true", opens, err)
	}
	if opens, err := testInstanceKey(t).OpensCheckRecord(store); err != nil || opens {
		t.Errorf("with the check record of another keyset = %v, %v; want false", opens, err)
	}
	if store.puts != puts {
		t.Error("OpensCheckRecord rewrote the check record")
	}
}

func TestAbandonNothingToDo(t *testing.T) {
	store := &memStartupStorage{memCheckStorage: &memCheckStorage{}, memSafeStorage: newMemSafeStorage()}
	if n, err := AbandonInstanceKeyset(store, nil); err != nil || n != 0 {
		t.Errorf("AbandonInstanceKeyset on an empty store = %d, %v; want 0", n, err)
	}
}
