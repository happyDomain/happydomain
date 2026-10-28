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
	"maps"
	"slices"

	"git.happydns.org/happyDomain/model"
)

// DropSafes deletes every instance safe, then the check record of the
// instance keyset, once back to the plaintext policy: the keyset is then no
// longer needed at startup. Safes of other kinds are left as they are. It
// returns how many safes it deleted.
//
// Whatever is still sealed in them no longer opens: the caller makes sure
// nothing is, see SecretsUsecase.DropSafes. A safe record that does not
// decode is left as it is, and so is the check record: the keyset stays
// required until it is dealt with. So is it while a safe is left whose store
// seals its entries under the instance keyset, see KeyBorrower.
func (m *Manager) DropSafes(ctx context.Context, checks CheckStorage) (int, error) {
	if m == nil || m.safes == nil {
		return 0, errNoManager
	}
	if m.policy != PolicyPlaintext {
		return 0, fmt.Errorf("safes are only dropped under the %q policy: under %q, the next secret saved needs one", PolicyPlaintext, m.policy)
	}

	var ids []happydns.Identifier
	damaged := 0
	borrowers := map[string]int{}
	err := eachSafe(m.safes, func(safe *happydns.Safe) {
		// Only the instance keyset is let go of: a safe of another kind
		// holds secrets its owner chose to keep there.
		if safe.Kind == KindInstance {
			ids = append(ids, safe.Id)
		} else if b, ok := m.stores[safe.Kind].(KeyBorrower); ok && b.Lender() == KindInstance {
			borrowers[safe.Kind]++
		}
	}, func(key string, err error) {
		log.Printf("secret: safe record %q does not decode, left as it is: %s", key, err)
		damaged++
	})
	if err != nil {
		return 0, err
	}

	dropped := 0
	for _, id := range ids {
		if err := m.safes.DeleteSafe(id); err != nil && !errors.Is(err, happydns.ErrSafeNotFound) {
			return dropped, fmt.Errorf("unable to delete safe %s: %w", id.String(), err)
		}
		dropped++
	}

	if damaged > 0 {
		return dropped, fmt.Errorf("%d safe records do not decode and were left as they are, with the keyset check record: the keyset stays required at startup", damaged)
	}
	if len(borrowers) > 0 {
		var errs []error
		for _, kind := range slices.Sorted(maps.Keys(borrowers)) {
			errs = append(errs, fmt.Errorf("%d safes of kind %q seal their keys under the instance keyset and were left as they are", borrowers[kind], kind))
		}
		errs = append(errs, errors.New("the keyset check record is kept: the keyset stays required at startup"))
		return dropped, errors.Join(errs...)
	}

	if err := checks.DeleteSecretCheck(); err != nil && !errors.Is(err, happydns.ErrNotFound) {
		return dropped, fmt.Errorf("unable to delete the keyset check record: %w", err)
	}
	return dropped, nil
}
