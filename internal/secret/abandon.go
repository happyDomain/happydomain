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
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

// ErrKeysetNotLost refuses to abandon a keyset that still opens the check
// record: what it protects can still be read.
var ErrKeysetNotLost = errors.New("the configured instance keyset opens the check record: it is not lost, nothing needs to be abandoned")

// CountInstanceSafes returns how many safes need the instance keyset. A safe
// record that does not decode is not counted.
func CountInstanceSafes(store SafeStorage) (int, error) {
	n := 0
	err := eachSafe(store, func(safe *happydns.Safe) {
		if safe.Kind == KindInstance {
			n++
		}
	}, nil)
	return n, err
}

// AbandonInstanceKeyset deletes every instance safe, then the check record,
// once the instance keyset is lost: happyDomain then starts again, without a
// keyset or with a new one. It returns how many safes it deleted.
//
// The secrets sealed in those safes stay where they are, unreadable for good,
// until their owners enter them again. key is the keyset configured, if any:
// when it still opens the check record, nothing is deleted and
// ErrKeysetNotLost is returned.
//
// A safe record that does not decode is left as it is: it does not prevent
// happyDomain from starting.
func AbandonInstanceKeyset(store Storage, key *InstanceKey) (int, error) {
	if key != nil {
		opens, err := key.OpensCheckRecord(store)
		if err != nil {
			return 0, err
		}
		if opens {
			return 0, ErrKeysetNotLost
		}
	}

	var ids []happydns.Identifier
	err := eachSafe(store, func(safe *happydns.Safe) {
		if safe.Kind == KindInstance {
			ids = append(ids, safe.Id)
		}
	}, func(key string, err error) {
		log.Printf("secret: safe record %q does not decode, left as it is: %s", key, err)
	})
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, id := range ids {
		if err := store.DeleteSafe(id); err != nil && !errors.Is(err, happydns.ErrSafeNotFound) {
			return deleted, fmt.Errorf("unable to delete safe %s: %w", id.String(), err)
		}
		deleted++
	}

	if err := store.DeleteSecretCheck(); err != nil && !errors.Is(err, happydns.ErrNotFound) {
		return deleted, fmt.Errorf("unable to delete the keyset check record: %w", err)
	}
	return deleted, nil
}
