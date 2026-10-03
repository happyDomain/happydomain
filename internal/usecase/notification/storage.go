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

package notification

import (
	"time"

	"git.happydns.org/happyDomain/model"
)

type NotificationChannelStorage interface {
	ListChannelsByUser(userId happydns.Identifier) ([]*happydns.NotificationChannel, error)
	GetChannel(channelId happydns.Identifier) (*happydns.NotificationChannel, error)
	CreateChannel(ch *happydns.NotificationChannel) error
	UpdateChannel(ch *happydns.NotificationChannel) error
	DeleteChannel(channelId happydns.Identifier) error
	// ListAllChannels lists the channels of every user, for administrative
	// passes such as the backup and tidy.
	ListAllChannels() (happydns.Iterator[happydns.NotificationChannel], error)
	// RestoreChannel stores ch under its identifier, replacing any channel
	// stored under it, as a backup restore does.
	RestoreChannel(ch *happydns.NotificationChannel) error
}

type NotificationPreferenceStorage interface {
	ListPreferencesByUser(userId happydns.Identifier) ([]*happydns.NotificationPreference, error)
	GetPreference(prefId happydns.Identifier) (*happydns.NotificationPreference, error)
	CreatePreference(pref *happydns.NotificationPreference) error
	UpdatePreference(pref *happydns.NotificationPreference) error
	DeletePreference(prefId happydns.Identifier) error
	// ListAllPreferences lists the preferences of every user, for
	// administrative passes such as the backup and tidy.
	ListAllPreferences() (happydns.Iterator[happydns.NotificationPreference], error)
	// RestorePreference stores pref under its identifier, replacing any
	// preference stored under it, as a backup restore does.
	RestorePreference(pref *happydns.NotificationPreference) error
}

type NotificationStateStorage interface {
	GetState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) (*happydns.NotificationState, error)
	PutState(state *happydns.NotificationState) error
	DeleteState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) error
	ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error)
	// ListAllStates lists the states of every user, for administrative
	// passes such as the backup.
	ListAllStates() (happydns.Iterator[happydns.NotificationState], error)
}

type NotificationRecordStorage interface {
	CreateRecord(rec *happydns.NotificationRecord) error
	ListRecordsByUser(userId happydns.Identifier, limit int) ([]*happydns.NotificationRecord, error)
	// ListAllRecords lists the records of every user, for administrative
	// passes such as the backup.
	ListAllRecords() (happydns.Iterator[happydns.NotificationRecord], error)
	// RestoreRecord stores rec under its identifier, replacing any record
	// stored under it, as a backup restore does.
	RestoreRecord(rec *happydns.NotificationRecord) error
	DeleteRecordsOlderThan(before time.Time) error
	// DeleteRecord removes one record with its user index entry, a record
	// already gone being no error.
	DeleteRecord(recId happydns.Identifier) error
}

type UserGetter interface {
	GetUser(id happydns.Identifier) (*happydns.User, error)
}

type DomainGetter interface {
	GetDomain(id happydns.Identifier) (*happydns.Domain, error)
}

type ZoneGetter interface {
	GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error)
}
