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
	"testing"

	"git.happydns.org/happyDomain/model"
)

func (s *memCheckStorage) DeleteSecretCheck() error {
	s.record = nil
	return nil
}

// Safes are only dropped once back to clear: under the instance policy, the
// next secret saved would need one again.
func TestDropSafesRefusedUnderInstancePolicy(t *testing.T) {
	instance, _, store := instanceManagers(t)
	sealOne(t, instance, objectContext(), "v")
	checks := &memCheckStorage{record: []byte("record")}

	if _, err := instance.DropSafes(context.Background(), checks); err == nil {
		t.Error("DropSafes under the instance policy succeeded, want an error")
	}
	if len(store.safes) != 1 || checks.record == nil {
		t.Errorf("DropSafes refused but deleted: %d safes, record %v", len(store.safes), checks.record)
	}
}

func TestDropSafesDeletesSafesAndCheckRecord(t *testing.T) {
	instance, plaintext, store := instanceManagers(t)
	for _, owner := range []happydns.Identifier{{0x01}, {0x02}} {
		sc := objectContext()
		sc.Owner = owner
		sealOne(t, instance, sc, "v")
	}
	checks := &memCheckStorage{record: []byte("record")}

	n, err := plaintext.DropSafes(context.Background(), checks)
	if err != nil || n != 2 {
		t.Fatalf("DropSafes = %d, %v; want 2 dropped", n, err)
	}
	if len(store.safes) != 0 {
		t.Errorf("%d safes left", len(store.safes))
	}
	if checks.record != nil {
		t.Error("the check record was kept: a new keyset would be refused at startup")
	}

	// Nothing left to do.
	if n, err := plaintext.DropSafes(context.Background(), checks); err != nil || n != 0 {
		t.Errorf("second DropSafes = %d, %v; want nothing to do", n, err)
	}
}
