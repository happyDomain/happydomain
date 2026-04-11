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
	"log"
	"strings"

	"git.happydns.org/happyDomain/model"
)

// Layout: notifch|<channelId> -> full record; notifch-user|<userId>|<channelId> -> "" (index only, no double-write).

const (
	notifchPrimaryPrefix = "notifch|"
	notifchUserPrefix    = "notifch-user|"
)

func notifchPrimaryKey(id happydns.Identifier) string {
	return notifchPrimaryPrefix + id.String()
}

func notifchUserKey(userId, channelId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", notifchUserPrefix, userId.String(), channelId.String())
}

func channelIdFromUserIndexKey(key string) (string, bool) {
	rest, ok := strings.CutPrefix(key, notifchUserPrefix)
	if !ok {
		return "", false
	}
	_, channelId, ok := strings.Cut(rest, "|")
	if !ok || channelId == "" {
		return "", false
	}
	return channelId, true
}

func (s *KVStorage) ListChannelsByUser(userId happydns.Identifier) ([]*happydns.NotificationChannel, error) {
	prefix := notifchUserPrefix + userId.String() + "|"
	iter := s.db.Search(prefix)
	defer iter.Release()

	var channels []*happydns.NotificationChannel
	for iter.Next() {
		idStr, ok := channelIdFromUserIndexKey(iter.Key())
		if !ok {
			continue
		}
		id, err := happydns.NewIdentifierFromString(idStr)
		if err != nil {
			log.Printf("storage: malformed channel index key %q: %v", iter.Key(), err)
			continue
		}
		ch, err := s.GetChannel(id)
		if err != nil {
			// Index drift: skip rather than fail the whole list.
			log.Printf("storage: channel index points to missing channel %q: %v", idStr, err)
			continue
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func (s *KVStorage) GetChannel(channelId happydns.Identifier) (*happydns.NotificationChannel, error) {
	ch := &happydns.NotificationChannel{}
	err := s.db.Get(notifchPrimaryKey(channelId), ch)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotificationChannelNotFound
	}
	return ch, err
}

func (s *KVStorage) CreateChannel(ch *happydns.NotificationChannel) error {
	key, id, err := s.db.FindIdentifierKey(notifchPrimaryPrefix)
	if err != nil {
		return err
	}
	ch.Id = id

	if err := s.db.Put(key, ch); err != nil {
		return err
	}
	if err := s.db.Put(notifchUserKey(ch.UserId, ch.Id), ""); err != nil {
		// Roll back primary so a failed index write doesn't orphan it.
		if delErr := s.db.Delete(key); delErr != nil {
			log.Printf("storage: orphan channel %q after index write failed (rollback also failed: %v)", ch.Id.String(), delErr)
		}
		return err
	}
	return nil
}

func (s *KVStorage) UpdateChannel(ch *happydns.NotificationChannel) error {
	// Index has no payload, so only the primary needs writing.
	return s.db.Put(notifchPrimaryKey(ch.Id), ch)
}

func (s *KVStorage) DeleteChannel(channelId happydns.Identifier) error {
	ch, err := s.GetChannel(channelId)
	if err != nil {
		return err
	}

	// Delete index first so partial failure hides the channel rather than leaving it visible-but-broken.
	if err := s.db.Delete(notifchUserKey(ch.UserId, channelId)); err != nil {
		return err
	}
	if err := s.db.Delete(notifchPrimaryKey(channelId)); err != nil {
		log.Printf("storage: channel %q index removed but primary delete failed: %v", channelId.String(), err)
		return err
	}
	return nil
}
