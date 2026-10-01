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
	ListAllChannels() (happydns.Iterator[happydns.NotificationChannel], error)
	ListChannelsByUser(userId happydns.Identifier) ([]*happydns.NotificationChannel, error)
	GetChannel(channelId happydns.Identifier) (*happydns.NotificationChannel, error)
	// CreateChannel stores a new channel under the identifier it carries.
	// ChannelService.CreateChannel, which generates that identifier, is its
	// only caller outside of tests. Fails with happydns.ErrInvalidIdentifier
	// or happydns.ErrAlreadyExists, or with whatever error the underlying
	// database returns.
	CreateChannel(ch *happydns.NotificationChannel) error
	// ReplaceChannel rewrites the channel id with what update returns from
	// the stored one. It writes nothing when update returns nil, and fails,
	// writing nothing, with happydns.ErrNotificationChannelNotFound or
	// happydns.ErrChangedMeanwhile when the channel was deleted or changed
	// in between. Its identifier and user cannot change.
	//
	// It is the only way to rewrite a channel: an unconditional write could
	// put back, over a reseal, a token of a safe that is dropped next.
	ReplaceChannel(id happydns.Identifier, update func(*happydns.NotificationChannel) (*happydns.NotificationChannel, error)) error
	DeleteChannel(channelId happydns.Identifier) error
}

type NotificationPreferenceStorage interface {
	ListPreferencesByUser(userId happydns.Identifier) ([]*happydns.NotificationPreference, error)
	GetPreference(prefId happydns.Identifier) (*happydns.NotificationPreference, error)
	CreatePreference(pref *happydns.NotificationPreference) error
	UpdatePreference(pref *happydns.NotificationPreference) error
	DeletePreference(prefId happydns.Identifier) error
}

type NotificationStateStorage interface {
	GetState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) (*happydns.NotificationState, error)
	PutState(state *happydns.NotificationState) error
	DeleteState(checkerID string, target happydns.CheckTarget, userId happydns.Identifier) error
	ListStatesByUser(userId happydns.Identifier) ([]*happydns.NotificationState, error)
}

type NotificationRecordStorage interface {
	CreateRecord(rec *happydns.NotificationRecord) error
	ListRecordsByUser(userId happydns.Identifier, limit int) ([]*happydns.NotificationRecord, error)
	DeleteRecordsOlderThan(before time.Time) error
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
