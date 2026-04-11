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

func (s *KVStorage) ListPreferencesByUser(userId happydns.Identifier) ([]*happydns.NotificationPreference, error) {
	prefix := fmt.Sprintf("notifpref-user|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var prefs []*happydns.NotificationPreference
	for iter.Next() {
		var pref happydns.NotificationPreference
		if err := s.db.DecodeData(iter.Value(), &pref); err != nil {
			continue
		}
		prefs = append(prefs, &pref)
	}
	return prefs, nil
}

func (s *KVStorage) GetPreference(prefId happydns.Identifier) (*happydns.NotificationPreference, error) {
	pref := &happydns.NotificationPreference{}
	err := s.db.Get(fmt.Sprintf("notifpref|%s", prefId.String()), pref)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrNotificationPreferenceNotFound
	}
	return pref, err
}

func (s *KVStorage) CreatePreference(pref *happydns.NotificationPreference) error {
	key, id, err := s.db.FindIdentifierKey("notifpref|")
	if err != nil {
		return err
	}
	pref.Id = id

	if err := s.db.Put(key, pref); err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifpref-user|%s|%s", pref.UserId.String(), pref.Id.String())
	return s.db.Put(indexKey, pref)
}

func (s *KVStorage) UpdatePreference(pref *happydns.NotificationPreference) error {
	if err := s.db.Put(fmt.Sprintf("notifpref|%s", pref.Id.String()), pref); err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifpref-user|%s|%s", pref.UserId.String(), pref.Id.String())
	return s.db.Put(indexKey, pref)
}

func (s *KVStorage) DeletePreference(prefId happydns.Identifier) error {
	pref, err := s.GetPreference(prefId)
	if err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifpref-user|%s|%s", pref.UserId.String(), prefId.String())
	if err := s.db.Delete(indexKey); err != nil {
		return err
	}

	return s.db.Delete(fmt.Sprintf("notifpref|%s", prefId.String()))
}
