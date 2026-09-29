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

// undecodableSafes lists, after the safes it holds, a record that does not
// decode, as a corrupted one.
type undecodableSafes struct {
	*memStartupStorage
}

func newUndecodableSafes() undecodableSafes {
	return undecodableSafes{newMemStartupStorage()}
}

func (u undecodableSafes) ListAllSafes() (happydns.Iterator[happydns.Safe], error) {
	inner, err := u.Safes.ListAllSafes()
	if err != nil {
		return nil, err
	}
	var entries []iterEntry[happydns.Safe]
	for inner.Next() {
		entries = append(entries, iterEntry[happydns.Safe]{key: "safe-" + inner.Item().Id.String(), item: inner.Item()})
	}
	entries = append(entries, iterEntry[happydns.Safe]{key: "safe-corrupt", err: errors.New("invalid character")})
	return newEntriesIterator(entries...), nil
}

// One damaged safe must not keep happyDomain from starting.
func TestStartupCheckGoesPastAnUndecodableSafe(t *testing.T) {
	store := newUndecodableSafes()
	if err := StartupCheck(PolicyPlaintext, nil, store); err != nil {
		t.Errorf("StartupCheck without keyset = %v", err)
	}

	key := testInstanceKey(t)
	if err := StartupCheck(PolicyInstance, key, store); err != nil {
		t.Errorf("StartupCheck with a keyset = %v", err)
	}

	// Readable instance safes still require their keyset.
	if _, err := newSafeRegistry(store, key).instanceSafe(happydns.Identifier{0x01}); err != nil {
		t.Fatal(err)
	}
	if err := StartupCheck(PolicyPlaintext, nil, store); err == nil {
		t.Error("StartupCheck accepted instance safes without their keyset")
	}
}
