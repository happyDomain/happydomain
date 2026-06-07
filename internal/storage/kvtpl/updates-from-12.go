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
	"log"
	"strings"

	happydns "git.happydns.org/happyDomain/model"
)

// migrateFrom12 rewrites all secondary index keys and the notification-state
// primary keys that exceeded the 64-char backend limit:
//
//   - Session user index: old "user.session-user|{uid}|{hash43}" (84 chars)
//     → new "su|{uid}|{hash32}" (58 chars), full hash stored as value.
//
//   - Evaluation plan index: old "chckeval-plan|{planId}|{chrono20}|{evalId}" (80 chars)
//     → new "evp|{planId}|{chrono11}|{evalId}" (61 chars) via compact chrono.
//
//   - Execution user index: old "chckexec-user|{uid}|{chrono20}|{execId}" (80 chars)
//     → new "exu|{uid}|{chrono11}|{execId}" (61 chars) via compact chrono.
//
//   - Execution domain index: old "chckexec-domain|{did}|{chrono20}|{execId}" (82 chars)
//     → new "exd|{did}|{chrono11}|{execId}" (61 chars) via compact chrono.
//
//   - Evaluation checker index and execution checker index: rebuilt with compact
//     chrono (still unbounded due to variable checkerID/target, but consistent).
//
//   - Notification state: old "notifstate|{uid}|{checkerID}|{target}" (unbounded)
//     → new "notifstate|{uid}|{hash28}" (62 chars).
//
//   - Checker options: old "chckrcfg|{name}|{uid}|{did}|{sid}" (unbounded)
//     → new "cfg|{hash28}" (32 chars) with secondary index "cfg-c|..." (63 chars).
func migrateFrom12(s *KVStorage) error {
	if err := migrateFrom12_sessions(s); err != nil {
		return err
	}
	if err := rebuildExecutionTimeIndexes(s); err != nil {
		return err
	}
	if err := rebuildEvaluationTimeIndexes(s); err != nil {
		return err
	}
	if err := migrateFrom12_notifState(s); err != nil {
		return err
	}
	return migrateFrom12_checkerOptions(s)
}

// migrateFrom12_sessions clears the old "user.session-user|" index and
// rebuilds it in the new "su|" format with the full hash stored as value.
func migrateFrom12_sessions(s *KVStorage) error {
	const oldSessionUserPrefix = "user.session-user|"
	if err := s.clearByPrefix(oldSessionUserPrefix); err != nil {
		return fmt.Errorf("migrateFrom12: clear old session user index: %w", err)
	}

	iter, err := s.ListAllSessions()
	if err != nil {
		return err
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		session := iter.Item()
		if session == nil || session.IdUser.IsEmpty() {
			continue
		}

		hash := strings.TrimPrefix(iter.Key(), sessionPrimaryPrefix)
		if hash == "" || hash == iter.Key() {
			log.Printf("migrateFrom12: skipping session with unexpected key %q", iter.Key())
			continue
		}

		if err := s.db.Put(sessionUserIndexKey(session.IdUser, sessionShortHash(hash)), hash); err != nil {
			return err
		}
		n++
	}

	if err := iter.Err(); err != nil {
		return err
	}

	log.Printf("migrateFrom12: rebuilt session user index for %d sessions", n)
	return nil
}

// migrateFrom12_notifState rewrites notification state keys from the old
// variable-length format to the new bounded hash-based format.
func migrateFrom12_notifState(s *KVStorage) error {
	const prefix = notificationStatePrimaryPrefix

	iter := s.db.Search(prefix)
	defer iter.Release()

	type oldEntry struct {
		key    string
		newKey string
		state  happydns.NotificationState
	}
	var toMigrate []oldEntry

	for iter.Next() {
		var state happydns.NotificationState
		if err := s.db.DecodeData(iter.Value(), &state); err != nil {
			log.Printf("migrateFrom12: skipping undecodable notif state at %q: %v", iter.Key(), err)
			continue
		}
		newKey := notifStateKey(state.CheckerID, state.Target, state.UserId)
		if iter.Key() != newKey {
			toMigrate = append(toMigrate, oldEntry{key: iter.Key(), newKey: newKey, state: state})
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}

	for _, e := range toMigrate {
		if err := s.db.Put(e.newKey, &e.state); err != nil {
			return fmt.Errorf("migrateFrom12: write notif state %q: %w", e.newKey, err)
		}
		if err := s.db.Delete(e.key); err != nil {
			return fmt.Errorf("migrateFrom12: delete old notif state %q: %w", e.key, err)
		}
	}

	log.Printf("migrateFrom12: migrated %d notification state keys", len(toMigrate))
	return nil
}

// migrateFrom12_checkerOptions rewrites checker option entries from the old
// "chckrcfg|{name}|..." primary keys into the new "cfg|{hash}" format and
// backfills the "cfg-c|" secondary index.
func migrateFrom12_checkerOptions(s *KVStorage) error {
	const oldPrefix = "chckrcfg|"

	iter := s.db.Search(oldPrefix)
	defer iter.Release()

	type oldEntry struct {
		key  string
		data happydns.CheckerOptionsPositional
	}
	var toMigrate []oldEntry

	for iter.Next() {
		// Parse the positional components from the old key format:
		// "chckrcfg|{checkerName}|{userId}|{domainId}|{serviceId}"
		trimmed := strings.TrimPrefix(iter.Key(), oldPrefix)
		parts := strings.SplitN(trimmed, "|", 4)
		if len(parts) < 4 {
			log.Printf("migrateFrom12: skipping malformed checker options key %q", iter.Key())
			continue
		}

		var stored happydns.CheckerOptionsPositional
		if err := s.db.DecodeData(iter.Value(), &stored.Options); err != nil {
			log.Printf("migrateFrom12: skipping undecodable checker options at %q: %v", iter.Key(), err)
			continue
		}

		stored.CheckName = parts[0]
		if parts[1] != "" {
			if id, err := happydns.NewIdentifierFromString(parts[1]); err == nil {
				stored.UserId = &id
			}
		}
		if parts[2] != "" {
			if id, err := happydns.NewIdentifierFromString(parts[2]); err == nil {
				stored.DomainId = &id
			}
		}
		if parts[3] != "" {
			if id, err := happydns.NewIdentifierFromString(parts[3]); err == nil {
				stored.ServiceId = &id
			}
		}

		toMigrate = append(toMigrate, oldEntry{key: iter.Key(), data: stored})
	}
	if err := iter.Err(); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	for _, e := range toMigrate {
		compoundHash := hash28(checkerOptionsCompound(e.data.CheckName, e.data.UserId, e.data.DomainId, e.data.ServiceId))
		primaryKey := checkerOptionPrimaryPrefix + compoundHash
		indexKey := checkerOptionNameIndexKey(e.data.CheckName, compoundHash)

		if err := batch.Put(primaryKey, e.data); err != nil {
			return fmt.Errorf("migrateFrom12: write checker options primary %q: %w", primaryKey, err)
		}
		if err := batch.Put(indexKey, ""); err != nil {
			return fmt.Errorf("migrateFrom12: write checker options index %q: %w", indexKey, err)
		}
		batch.Delete(e.key)
	}
	if len(toMigrate) > 0 {
		if err := batch.Commit(); err != nil {
			return fmt.Errorf("migrateFrom12: commit checker options migration: %w", err)
		}
	}

	log.Printf("migrateFrom12: migrated %d checker option entries", len(toMigrate))
	return nil
}
