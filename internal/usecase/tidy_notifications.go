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
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// TidyNotificationChannels deletes the notification channels whose user no
// longer exists, with their user index entry.
func (tu *tidyUpUsecase) TidyNotificationChannels(dropInvalid bool) error {
	iter, err := tu.store.ListAllChannels()
	if err != nil {
		return err
	}

	return tidyUserOrphans(tu, iter, dropInvalid,
		func(ch *happydns.NotificationChannel) happydns.Identifier { return ch.UserId },
		func(ch *happydns.NotificationChannel) string { return "notification channel " + ch.Id.String() },
		func(ch *happydns.NotificationChannel) error { return tu.store.DeleteChannel(ch.Id) },
		happydns.ErrNotificationChannelNotFound,
	)
}

// TidyNotificationPreferences deletes the notification preferences whose user
// no longer exists, with their user index entry.
func (tu *tidyUpUsecase) TidyNotificationPreferences(dropInvalid bool) error {
	iter, err := tu.store.ListAllPreferences()
	if err != nil {
		return err
	}

	return tidyUserOrphans(tu, iter, dropInvalid,
		func(pref *happydns.NotificationPreference) happydns.Identifier { return pref.UserId },
		func(pref *happydns.NotificationPreference) string {
			return "notification preference " + pref.Id.String()
		},
		func(pref *happydns.NotificationPreference) error { return tu.store.DeletePreference(pref.Id) },
		happydns.ErrNotificationPreferenceNotFound,
	)
}

// TidyNotificationStates deletes the notification states whose user no longer
// exists.
func (tu *tidyUpUsecase) TidyNotificationStates(dropInvalid bool) error {
	iter, err := tu.store.ListAllStates()
	if err != nil {
		return err
	}

	return tidyUserOrphans(tu, iter, dropInvalid,
		func(state *happydns.NotificationState) happydns.Identifier { return state.UserId },
		func(state *happydns.NotificationState) string {
			return fmt.Sprintf("notification state of %s for %s", state.CheckerID, state.UserId.String())
		},
		func(state *happydns.NotificationState) error {
			return tu.store.DeleteState(state.CheckerID, state.Target, state.UserId)
		},
		happydns.ErrNotificationStateNotFound,
	)
}

// TidyNotificationRecords deletes the notification records whose user no
// longer exists, with their user index entry.
func (tu *tidyUpUsecase) TidyNotificationRecords(dropInvalid bool) error {
	iter, err := tu.store.ListAllRecords()
	if err != nil {
		return err
	}

	return tidyUserOrphans(tu, iter, dropInvalid,
		func(rec *happydns.NotificationRecord) happydns.Identifier { return rec.UserId },
		func(rec *happydns.NotificationRecord) string { return "notification record " + rec.Id.String() },
		func(rec *happydns.NotificationRecord) error { return tu.store.DeleteRecord(rec.Id) },
		nil,
	)
}
