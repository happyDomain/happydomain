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

	"git.happydns.org/happyDomain/model"
)

// TidyNotificationChannels deletes the notification channels whose user no
// longer exists, with their user index entry.
func (tu *tidyUpUsecase) TidyNotificationChannels(dropInvalid bool) error {
	iter, err := tu.store.ListAllChannels()
	if err != nil {
		return err
	}

	var orphans []happydns.Identifier
	err = iterateTidy(iter, dropInvalid, func(ch *happydns.NotificationChannel) error {
		_, err := tu.store.GetUser(ch.UserId)
		if errors.Is(err, happydns.ErrUserNotFound) {
			orphans = append(orphans, ch.Id)
			return nil
		}
		return err
	})
	iter.Close()
	if err != nil {
		return err
	}

	for _, id := range orphans {
		log.Printf("Deleting orphan notification channel %s (user not found)", id.String())
		if err := tu.store.DeleteChannel(id); err != nil && !errors.Is(err, happydns.ErrNotificationChannelNotFound) {
			return fmt.Errorf("unable to delete orphan notification channel %s: %w", id.String(), err)
		}
	}

	return nil
}
