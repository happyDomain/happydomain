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

// Same primary+per-user-index layout as notification_channel.go.

const (
	notifprefPrimaryPrefix = "notifpref|"
	notifprefUserPrefix    = "notifpref-user|"
)

func notifprefPrimaryKey(id happydns.Identifier) string {
	return notifprefPrimaryPrefix + id.String()
}

func notifprefUserKey(userId, prefId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", notifprefUserPrefix, userId.String(), prefId.String())
}

func prefIdFromUserIndexKey(key string) (string, bool) {
	rest, ok := strings.CutPrefix(key, notifprefUserPrefix)
	if !ok {
		return "", false
	}
	_, prefId, ok := strings.Cut(rest, "|")
	if !ok || prefId == "" {
		return "", false
	}
	return prefId, true
}

func (s *KVStorage) ListPreferencesByUser(userId happydns.Identifier) ([]*happydns.NotificationPreference, error) {
	prefix := notifprefUserPrefix + userId.String() + "|"
	iter := s.db.Search(prefix)
	defer iter.Release()

	var prefs []*happydns.NotificationPreference
	for iter.Next() {
		idStr, ok := prefIdFromUserIndexKey(iter.Key())
		if !ok {
			continue
		}
		id, err := happydns.NewIdentifierFromString(idStr)
		if err != nil {
			log.Printf("storage: malformed preference index key %q: %v", iter.Key(), err)
			continue
		}
		pref, err := s.GetPreference(id)
		if err != nil {
			log.Printf("storage: preference index points to missing preference %q: %v", idStr, err)
			continue
		}
		prefs = append(prefs, pref)
	}
	return prefs, nil
}

func (s *KVStorage) GetPreference(prefId happydns.Identifier) (*happydns.NotificationPreference, error) {
	pref := &happydns.NotificationPreference{}
	err := s.db.Get(notifprefPrimaryKey(prefId), pref)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotificationPreferenceNotFound
	}
	return pref, err
}

func (s *KVStorage) CreatePreference(pref *happydns.NotificationPreference) error {
	key, id, err := s.db.FindIdentifierKey(notifprefPrimaryPrefix)
	if err != nil {
		return err
	}
	pref.Id = id

	if err := s.db.Put(key, pref); err != nil {
		return err
	}
	if err := s.db.Put(notifprefUserKey(pref.UserId, pref.Id), ""); err != nil {
		if delErr := s.db.Delete(key); delErr != nil {
			log.Printf("storage: orphan preference %q after index write failed (rollback also failed: %v)", pref.Id.String(), delErr)
		}
		return err
	}
	return nil
}

func (s *KVStorage) UpdatePreference(pref *happydns.NotificationPreference) error {
	return s.db.Put(notifprefPrimaryKey(pref.Id), pref)
}

func (s *KVStorage) DeletePreference(prefId happydns.Identifier) error {
	pref, err := s.GetPreference(prefId)
	if err != nil {
		return err
	}

	if err := s.db.Delete(notifprefUserKey(pref.UserId, prefId)); err != nil {
		return err
	}
	if err := s.db.Delete(notifprefPrimaryKey(prefId)); err != nil {
		log.Printf("storage: preference %q index removed but primary delete failed: %v", prefId.String(), err)
		return err
	}
	return nil
}
