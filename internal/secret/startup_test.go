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
	"errors"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// Starting under the plaintext policy with a keyset configured, nothing being
// sealed, must not record that keyset: it would refuse every other one later.
func TestStartupCheckRecordsNoKeysetWithNothingSealed(t *testing.T) {
	store := newMemStartupStorage()

	if err := StartupCheck(PolicyPlaintext, testInstanceKey(t), store); err != nil {
		t.Fatalf("StartupCheck: %v", err)
	}
	if store.record != nil || store.puts != 0 {
		t.Error("a check record was written while nothing is sealed")
	}

	if err := StartupCheck(PolicyInstance, testInstanceKey(t), store); err != nil {
		t.Errorf("StartupCheck with another keyset: %v", err)
	}
}

// With no instance safe left, a check record is stale: whatever keyset it
// names opens nothing. It is dropped, so that another keyset can come.
func TestStartupCheckDropsAStaleRecord(t *testing.T) {
	old := testInstanceKey(t)

	for name, key := range map[string]*InstanceKey{
		"without keyset":      nil,
		"with the old keyset": old,
		"with another keyset": testInstanceKey(t),
	} {
		t.Run(name, func(t *testing.T) {
			store := newMemStartupStorage()
			if err := old.VerifyCheck(store); err != nil {
				t.Fatal(err)
			}

			if err := StartupCheck(PolicyPlaintext, key, store); err != nil {
				t.Fatalf("StartupCheck: %v", err)
			}
			if store.record != nil {
				t.Error("the stale check record is still there")
			}

			next := testInstanceKey(t)
			if err := StartupCheck(PolicyInstance, next, store); err != nil {
				t.Fatalf("StartupCheck with a new keyset: %v", err)
			}
			if err := next.VerifyCheck(store); err != nil {
				t.Errorf("the new keyset does not open the check record: %v", err)
			}
		})
	}
}

// Under the instance policy, a stale record gives way to one for the keyset
// configured.
func TestStartupCheckReplacesAStaleRecordUnderInstance(t *testing.T) {
	store := newMemStartupStorage()
	if err := testInstanceKey(t).VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	key := testInstanceKey(t)
	if err := StartupCheck(PolicyInstance, key, store); err != nil {
		t.Fatalf("StartupCheck: %v", err)
	}
	if err := key.VerifyCheck(store); err != nil {
		t.Errorf("the check record is not the configured keyset's: %v", err)
	}
}

// While an instance safe is stored, the record stays, and keeps refusing
// another keyset, whatever the policy.
func TestStartupCheckKeepsTheRecordOfInstanceSafes(t *testing.T) {
	key := testInstanceKey(t)
	store := newMemStartupStorage()
	m, _ := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	sealFor(t, m, happydns.Identifier{0x01})

	if err := StartupCheck(PolicyPlaintext, key, store); err != nil {
		t.Fatalf("StartupCheck: %v", err)
	}
	if store.record == nil {
		t.Error("no check record while an instance safe is stored")
	}
	if err := StartupCheck(PolicyPlaintext, testInstanceKey(t), store); !errors.Is(err, ErrWrongInstanceKey) {
		t.Errorf("StartupCheck with another keyset = %v, want ErrWrongInstanceKey", err)
	}
}

// A safe record that does not decode may be an instance safe: while it is
// there, the record stays, as DropInstanceSafes keeps it, and the keyset it
// names stays required. Starting without it would let the operator lose the
// keyset while something may still need it.
func TestStartupCheckRequiresTheKeysetWhileASafeIsDamaged(t *testing.T) {
	old := testInstanceKey(t)
	store := newUndecodableSafes()
	if err := old.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	if err := StartupCheck(PolicyPlaintext, nil, store); err == nil {
		t.Error("StartupCheck accepted no keyset while a safe record does not decode and a check record is there")
	}
	if store.record == nil {
		t.Fatal("the check record was dropped while a safe could not be read")
	}
	if err := StartupCheck(PolicyPlaintext, testInstanceKey(t), store); !errors.Is(err, ErrWrongInstanceKey) {
		t.Errorf("StartupCheck with another keyset = %v, want ErrWrongInstanceKey", err)
	}
	if err := StartupCheck(PolicyPlaintext, old, store); err != nil {
		t.Errorf("StartupCheck with the keyset of the check record = %v", err)
	}
}

// With no check record, and no instance safe that decodes to try it on,
// nothing tells whether a keyset is the right one: none is recorded, or the
// right one would be refused once the damaged records decode again.
func TestStartupCheckRecordsNoKeysetOverDamagedSafesOnly(t *testing.T) {
	store := newUndecodableSafes()

	for _, policy := range []Policy{PolicyInstance, PolicyPlaintext} {
		if err := StartupCheck(policy, testInstanceKey(t), store); err != nil {
			t.Fatalf("StartupCheck(%s) = %v", policy, err)
		}
		if store.record != nil || store.puts != 0 {
			t.Fatalf("StartupCheck(%s) recorded a keyset it could not verify", policy)
		}
	}
}
