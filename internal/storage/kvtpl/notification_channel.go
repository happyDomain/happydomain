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

package database

import (
	"errors"
	"fmt"

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListChannelsByUser(userId happydns.Identifier) ([]*happydns.NotificationChannel, error) {
	prefix := fmt.Sprintf("notifch-user|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var channels []*happydns.NotificationChannel
	for iter.Next() {
		var ch happydns.NotificationChannel
		if err := s.db.DecodeData(iter.Value(), &ch); err != nil {
			continue
		}
		channels = append(channels, &ch)
	}
	return channels, nil
}

func (s *KVStorage) GetChannel(channelId happydns.Identifier) (*happydns.NotificationChannel, error) {
	ch := &happydns.NotificationChannel{}
	err := s.db.Get(fmt.Sprintf("notifch|%s", channelId.String()), ch)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotificationChannelNotFound
	}
	return ch, err
}

func (s *KVStorage) CreateChannel(ch *happydns.NotificationChannel) error {
	key, id, err := s.db.FindIdentifierKey("notifch|")
	if err != nil {
		return err
	}
	ch.Id = id

	if err := s.db.Put(key, ch); err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifch-user|%s|%s", ch.UserId.String(), ch.Id.String())
	return s.db.Put(indexKey, ch)
}

func (s *KVStorage) UpdateChannel(ch *happydns.NotificationChannel) error {
	if err := s.db.Put(fmt.Sprintf("notifch|%s", ch.Id.String()), ch); err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifch-user|%s|%s", ch.UserId.String(), ch.Id.String())
	return s.db.Put(indexKey, ch)
}

func (s *KVStorage) DeleteChannel(channelId happydns.Identifier) error {
	ch, err := s.GetChannel(channelId)
	if err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifch-user|%s|%s", ch.UserId.String(), channelId.String())
	if err := s.db.Delete(indexKey); err != nil {
		return err
	}

	return s.db.Delete(fmt.Sprintf("notifch|%s", channelId.String()))
}
