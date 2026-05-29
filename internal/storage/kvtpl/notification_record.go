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
	"sort"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

// Same primary+per-user-index layout as notification_channel.go: the index
// value is empty and the record is resolved from the primary key.

const (
	notificationRecordPrimaryPrefix   = "notifrec|"
	notificationRecordUserIndexPrefix = "notifrec-user|"
)

func notifrecPrimaryKey(id happydns.Identifier) string {
	return notificationRecordPrimaryPrefix + id.String()
}

func notifrecUserKey(userId, recId happydns.Identifier) string {
	return fmt.Sprintf("%s%s|%s", notificationRecordUserIndexPrefix, userId.String(), recId.String())
}

func recIdFromUserIndexKey(key string) (string, bool) {
	rest, ok := strings.CutPrefix(key, notificationRecordUserIndexPrefix)
	if !ok {
		return "", false
	}
	_, recId, ok := strings.Cut(rest, "|")
	if !ok || recId == "" {
		return "", false
	}
	return recId, true
}

func (s *KVStorage) getRecord(recId happydns.Identifier) (*happydns.NotificationRecord, error) {
	rec := &happydns.NotificationRecord{}
	if err := s.db.Get(notifrecPrimaryKey(recId), rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *KVStorage) CreateRecord(rec *happydns.NotificationRecord) error {
	key, id, err := s.db.FindIdentifierKey(notificationRecordPrimaryPrefix)
	if err != nil {
		return err
	}
	rec.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, rec); err != nil {
		return err
	}
	if err := batch.Put(notifrecUserKey(rec.UserId, rec.Id), ""); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) ListRecordsByUser(userId happydns.Identifier, limit int) ([]*happydns.NotificationRecord, error) {
	prefix := notificationRecordUserIndexPrefix + userId.String() + "|"
	iter := s.db.Search(prefix)
	defer iter.Release()

	var records []*happydns.NotificationRecord
	for iter.Next() {
		idStr, ok := recIdFromUserIndexKey(iter.Key())
		if !ok {
			continue
		}
		id, err := happydns.NewIdentifierFromString(idStr)
		if err != nil {
			log.Printf("storage: malformed notification record index key %q: %v", iter.Key(), err)
			continue
		}
		rec, err := s.getRecord(id)
		if err != nil {
			// Index drift: skip rather than fail the whole listing.
			log.Printf("storage: notification record index points to missing record %q: %v", idStr, err)
			continue
		}
		records = append(records, rec)
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
	iter := s.db.Search(notificationRecordPrimaryPrefix)
	defer iter.Release()

	var errs []error
	for iter.Next() {
		var rec happydns.NotificationRecord
		if err := s.db.DecodeData(iter.Value(), &rec); err != nil {
			log.Printf("storage: malformed notification record at %q: %v", iter.Key(), err)
			continue
		}
		if rec.SentAt.Before(before) {
			batch := s.db.NewBatch()
			batch.Delete(iter.Key())
			batch.Delete(notifrecUserKey(rec.UserId, rec.Id))
			if err := batch.Commit(); err != nil {
				errs = append(errs, fmt.Errorf("delete record %s: %w", iter.Key(), err))
			}
		}
	}
	return errors.Join(errs...)
}
