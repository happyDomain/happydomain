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
	"fmt"
	"log"

	"git.happydns.org/happyDomain/model"
)

// DropSafes deletes every safe, then the check record of the instance keyset,
// once back to the plaintext policy: the keyset is then no longer needed at
// startup. It returns how many safes it deleted.
//
// Whatever is still sealed in them no longer opens: the caller makes sure
// nothing is, see SecretsUsecase.DropSafes. A safe record that does not
// decode is left as it is, and so is the check record: the keyset stays
// required until it is dealt with.
func (m *Manager) DropSafes(ctx context.Context, checks CheckStorage) (int, error) {
	if m == nil || m.safes == nil {
		return 0, errNoManager
	}
	if m.policy != PolicyPlaintext {
		return 0, fmt.Errorf("safes are only dropped under the %q policy: under %q, the next secret saved needs one", PolicyPlaintext, m.policy)
	}

	iter, err := m.safes.store.ListAllSafes()
	if err != nil {
		return 0, err
	}

	// Collected first: deleting while iterating is not safe on every
	// storage.
	var ids []happydns.Identifier
	damaged := 0
	for iter.NextWithError() {
		if err := iter.Err(); err != nil {
			log.Printf("secret: safe record %q does not decode, left as it is: %s", iter.Key(), err)
			damaged++
			continue
		}
		ids = append(ids, iter.Item().Id)
	}
	err = iter.Err()
	iter.Close()
	if err != nil {
		return 0, err
	}

	dropped := 0
	for _, id := range ids {
		if err := m.safes.store.DeleteSafe(id); err != nil && !errors.Is(err, happydns.ErrSafeNotFound) {
			return dropped, fmt.Errorf("unable to delete safe %s: %w", id.String(), err)
		}
		dropped++
	}

	if damaged > 0 {
		return dropped, fmt.Errorf("%d safe records do not decode and were left as they are, with the keyset check record: the keyset stays required at startup", damaged)
	}

	if err := checks.DeleteSecretCheck(); err != nil && !errors.Is(err, happydns.ErrNotFound) {
		return dropped, fmt.Errorf("unable to delete the keyset check record: %w", err)
	}
	return dropped, nil
}
