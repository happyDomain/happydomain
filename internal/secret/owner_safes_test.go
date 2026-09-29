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
	"testing"

	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/model"
)

// deletedMeanwhile has every safe deleted by someone else right before it
// is deleted, like a tidy run at the same time.
type deletedMeanwhile struct {
	*secrettest.Safes
}

func (s deletedMeanwhile) DeleteSafe(id happydns.Identifier) error {
	if err := s.Safes.DeleteSafe(id); err != nil {
		return err
	}
	return happydns.ErrSafeNotFound
}

// A safe deleted since it was listed is what deleting it was for: the account
// of its owner can still be deleted.
func TestDeleteOwnerSafesGoesPastASafeDeletedMeanwhile(t *testing.T) {
	key := testInstanceKey(t)
	store := secrettest.NewSafes()

	instance, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: store})
	if err != nil {
		t.Fatal(err)
	}
	sealOne(t, instance, objectContext(), "v")

	m, err := NewManager(Config{Policy: PolicyInstance, InstanceKey: key, Safes: deletedMeanwhile{store}})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteOwnerSafes(objectContext().Owner); err != nil {
		t.Errorf("DeleteOwnerSafes = %v, want no error", err)
	}
}
