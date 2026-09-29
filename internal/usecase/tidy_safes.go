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

package usecase

import (
	"errors"
	"fmt"
	"log"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/model"
)

// TidySafes deletes the safes whose owner no longer exists. The safe of the
// instance itself has no user owner, and is kept. A safe record
// that does not decode is kept whatever dropInvalid says: deleting it would
// make the secrets it holds unreadable for good.
func (tu *tidyUpUsecase) TidySafes(_ bool) error {
	iter, err := tu.store.ListAllSafes()
	if err != nil {
		return err
	}

	var orphans []happydns.Identifier
	err = iterateTidy(iter, false, func(safe *happydns.Safe) error {
		if secret.IsInstanceOwner(safe.Owner) {
			return nil
		}
		_, err := tu.store.GetUser(safe.Owner)
		if errors.Is(err, happydns.ErrUserNotFound) {
			orphans = append(orphans, safe.Id)
			return nil
		}
		return err
	})
	iter.Close()
	if err != nil {
		return err
	}

	for _, id := range orphans {
		log.Printf("Deleting orphan safe %s (owner not found)", id.String())
		// Deleted meanwhile, with its owner say: that is what was asked.
		if err := tu.store.DeleteSafe(id); err != nil && !errors.Is(err, happydns.ErrSafeNotFound) {
			return fmt.Errorf("unable to delete orphan safe %s: %w", id.String(), err)
		}
	}

	return nil
}
