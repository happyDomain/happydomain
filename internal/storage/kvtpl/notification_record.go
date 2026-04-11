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
	"fmt"
	"sort"
	"time"

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) CreateRecord(rec *happydns.NotificationRecord) error {
	key, id, err := s.db.FindIdentifierKey("notifrec|")
	if err != nil {
		return err
	}
	rec.Id = id

	if err := s.db.Put(key, rec); err != nil {
		return err
	}

	indexKey := fmt.Sprintf("notifrec-user|%s|%s", rec.UserId.String(), rec.Id.String())
	return s.db.Put(indexKey, rec)
}

func (s *KVStorage) ListRecordsByUser(userId happydns.Identifier, limit int) ([]*happydns.NotificationRecord, error) {
	prefix := fmt.Sprintf("notifrec-user|%s|", userId.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	var records []*happydns.NotificationRecord
	for iter.Next() {
		var rec happydns.NotificationRecord
		if err := s.db.DecodeData(iter.Value(), &rec); err != nil {
			continue
		}
		records = append(records, &rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].SentAt.After(records[j].SentAt)
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

func (s *KVStorage) DeleteRecordsOlderThan(before time.Time) error {
	iter := s.db.Search("notifrec|")
	defer iter.Release()

	for iter.Next() {
		var rec happydns.NotificationRecord
		if err := s.db.DecodeData(iter.Value(), &rec); err != nil {
			continue
		}
		if rec.SentAt.Before(before) {
			_ = s.db.Delete(iter.Key())
			userIndexKey := fmt.Sprintf("notifrec-user|%s|%s", rec.UserId.String(), rec.Id.String())
			_ = s.db.Delete(userIndexKey)
		}
	}
	return nil
}
