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

	"github.com/tink-crypto/tink-go/v2/keyset"

	"git.happydns.org/happyDomain/model"
)

// A safe wrapped by a key the keyset lacks, removed or disabled too early, is
// not lost: putting the key back opens it again. Dropping the instance safes
// keeps it, reported like a damaged one, along with the check record, and
// deletes the others.
func TestDropInstanceSafesKeepsSafesWrappedByAnUnusableKey(t *testing.T) {
	ctx := context.Background()

	for name, retire := range map[string]func(t *testing.T, h *keyset.Handle) *keyset.Handle{
		"removed":  onlyPrimary,
		"disabled": disableNonPrimary,
	} {
		t.Run(name, func(t *testing.T) {
			store := newMemStartupStorage()

			h, err := GenerateInstanceKeyset()
			if err != nil {
				t.Fatal(err)
			}
			old, err := NewInstanceKey(h)
			if err != nil {
				t.Fatal(err)
			}
			if err := old.VerifyCheck(store); err != nil {
				t.Fatal(err)
			}
			oldInstance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: old, Safes: store})
			if err != nil {
				t.Fatal(err)
			}
			sealOne(t, oldInstance, objectContext(), "v")

			rotated, err := RotateInstanceKeyset(h)
			if err != nil {
				t.Fatal(err)
			}
			current, err := NewInstanceKey(retire(t, rotated))
			if err != nil {
				t.Fatal(err)
			}

			// Another user's safe, wrapped by the current key.
			instance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: current, Safes: store})
			if err != nil {
				t.Fatal(err)
			}
			other := objectContext()
			other.Owner = happydns.Identifier{0x02}
			sealOne(t, instance, other, "w")

			plaintext, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: current, Safes: store})
			if err != nil {
				t.Fatal(err)
			}
			report, err := plaintext.DropInstanceSafes(ctx, store)
			if !errors.Is(err, ErrSafesLeft) || report.Changed != 1 || report.Failed != 1 {
				t.Fatalf("DropInstanceSafes = %+v, %v; want 1 dropped, 1 kept, ErrSafesLeft", report, err)
			}
			if _, err := store.GetSafeByOwner(objectContext().Owner, KindInstance); err != nil {
				t.Errorf("the safe wrapped by the %s key was deleted: %v", name, err)
			}
			if _, err := store.GetSafeByOwner(other.Owner, KindInstance); err == nil {
				t.Error("the safe wrapped by the current key is still there")
			}
			if store.record == nil {
				t.Error("the check record was deleted while a safe is kept")
			}
		})
	}
}

// disableNonPrimary returns h with its non-primary keys disabled, as after an
// operator ran tinkey disable-key.
func disableNonPrimary(t *testing.T, h *keyset.Handle) *keyset.Handle {
	t.Helper()
	km := keyset.NewManagerFromHandle(h)
	primary := h.KeysetInfo().GetPrimaryKeyId()
	for _, k := range h.KeysetInfo().GetKeyInfo() {
		if k.GetKeyId() != primary {
			if err := km.Disable(k.GetKeyId()); err != nil {
				t.Fatal(err)
			}
		}
	}
	out, err := km.Handle()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// vanishingSafes has every safe deleted by someone else right before it is
// deleted, like the safes of a user deleted meanwhile.
type vanishingSafes struct {
	*memStartupStorage
}

func (s vanishingSafes) DeleteSafe(id happydns.Identifier) error {
	if err := s.memStartupStorage.DeleteSafe(id); err != nil {
		return err
	}
	return happydns.ErrSafeNotFound
}

// A safe deleted since it was listed is what dropping it was for: it is
// neither a failure nor a reason to keep the check record.
func TestDropInstanceSafesGoesPastASafeDeletedMeanwhile(t *testing.T) {
	ctx := context.Background()
	key := testInstanceKey(t)
	store := newMemStartupStorage()
	if err := key.VerifyCheck(store); err != nil {
		t.Fatal(err)
	}

	instance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	sealOne(t, instance, objectContext(), "v")

	plaintext, err := NewManager(Config{Policy: PolicyPlaintext, InstanceKey: key, Safes: vanishingSafes{store}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := plaintext.DropInstanceSafes(ctx, store)
	if err != nil || report.Failed != 0 || report.Skipped != 0 {
		t.Fatalf("DropInstanceSafes = %+v, %v; want no failure", report, err)
	}
	if store.record != nil {
		t.Error("the check record is still there")
	}
}
