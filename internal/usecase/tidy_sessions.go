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

func (tu *tidyUpUsecase) TidySessions(dropInvalid bool) error {
	iter, err := tu.store.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	return iterateTidy(iter, dropInvalid, func(session *happydns.Session) error {
		_, err := tu.store.GetUser(session.IdUser)
		if errors.Is(err, happydns.ErrUserNotFound) {
			// Drop session from unexistant users
			log.Printf("Deleting orphan session (user %s not found): %v\n", session.IdUser.String(), session)
			if err = iter.DropItem(); err != nil {
				return err
			}
		}
		return nil
	})
}
