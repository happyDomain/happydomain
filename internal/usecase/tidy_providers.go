// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"log"

	"git.happydns.org/happyDomain/model"
)

func (tu *tidyUpUsecase) TidyProviders(dropInvalid bool) error {
	iter, err := tu.store.ListAllProviders()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(prvd *happydns.ProviderMessage) error {
		_, err := tu.store.GetUser(prvd.Owner)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop providers of unexistant users
			log.Printf("Deleting orphan provider (user %s not found): %v\n", prvd.Owner.String(), prvd)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
