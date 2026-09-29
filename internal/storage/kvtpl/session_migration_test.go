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
	"encoding/json"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/model"
)

// legacyToken is shaped like a token from NewSessionID: 103 base32 chars.
var legacyToken = strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", 4)[:103]

// putLegacySession writes a session as it was stored before migrateFrom14:
// hashed key and index, but the token itself in Id.
func putLegacySession(t *testing.T, s *KVStorage, token string, owner happydns.Identifier) {
	t.Helper()

	hash := sessionHash(token)
	if err := s.db.Put(sessionKey(token), &happydns.Session{Id: token, IdUser: owner, Description: "legacy"}); err != nil {
		t.Fatalf("put legacy session: %v", err)
	}
	if err := s.db.Put(sessionUserIndexKey(owner, sessionShortHash(hash)), hash); err != nil {
		t.Fatalf("put legacy session index: %v", err)
	}
}

func assertNoTokenStored(t *testing.T, s *KVStorage, token string) {
	t.Helper()

	iter := s.db.Search("")
	defer iter.Release()
	for iter.Next() {
		raw, err := json.Marshal(iter.Value())
		if err != nil {
			t.Fatalf("encode %q: %v", iter.Key(), err)
		}
		if strings.Contains(iter.Key(), token) || strings.Contains(string(raw), token) {
			t.Errorf("record %q still carries the token", iter.Key())
		}
	}
}

func TestMigrateFrom14RemovesTokens(t *testing.T) {
	s := &KVStorage{db: newFakeKV()}
	owner := happydns.Identifier([]byte("owner"))
	putLegacySession(t, s, legacyToken, owner)

	if err := migrateFrom14(s); err != nil {
		t.Fatalf("migrateFrom14: %v", err)
	}

	assertNoTokenStored(t, s, legacyToken)

	// The client still holds the token: it must still resolve to its session.
	id := happydns.SessionIDFromToken(legacyToken)
	got, err := s.GetSession(id)
	if err != nil {
		t.Fatalf("GetSession after migration: %v", err)
	}
	if got.Id != id || got.Description != "legacy" {
		t.Errorf("migrated session = %+v, want Id %q and description kept", got, id)
	}

	listed, err := s.ListUserSessions(owner)
	if err != nil {
		t.Fatalf("ListUserSessions: %v", err)
	}
	if len(listed) != 1 || listed[0].Id != id {
		t.Errorf("listed sessions = %+v, want the migrated session", listed)
	}

	// Running it again leaves things as they are.
	if err := migrateFrom14(s); err != nil {
		t.Fatalf("second migrateFrom14: %v", err)
	}
	if got, err := s.GetSession(id); err != nil || got.Id != id {
		t.Errorf("session after second run = %+v, %v", got, err)
	}
}

func TestMigrateFrom14OverwritesMismatchedId(t *testing.T) {
	s := &KVStorage{db: newFakeKV()}
	id := happydns.SessionIDFromToken(legacyToken)
	if err := s.db.Put(sessionPrimaryKeyFromHash(id), &happydns.Session{Id: "unrelated"}); err != nil {
		t.Fatalf("put session: %v", err)
	}

	if err := migrateFrom14(s); err != nil {
		t.Fatalf("migrateFrom14: %v", err)
	}

	got, err := s.GetSession(id)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.Id != id {
		t.Errorf("session Id = %q, want the key's %q", got.Id, id)
	}
}

// TestMigrateFrom9StoresPublicIdentifier covers the older migration, which
// moved sessions from a key holding the token to a hashed key: it must now
// write them in the current shape, without the token.
func TestMigrateFrom9StoresPublicIdentifier(t *testing.T) {
	s := &KVStorage{db: newFakeKV()}
	owner := happydns.Identifier([]byte("owner"))
	if err := s.db.Put(sessionPrimaryPrefix+legacyToken, &happydns.Session{Id: legacyToken, IdUser: owner}); err != nil {
		t.Fatalf("put pre-9 session: %v", err)
	}

	if err := migrateFrom9(s); err != nil {
		t.Fatalf("migrateFrom9: %v", err)
	}

	assertNoTokenStored(t, s, legacyToken)

	id := happydns.SessionIDFromToken(legacyToken)
	if got, err := s.GetSession(id); err != nil || got.Id != id {
		t.Errorf("session after migrateFrom9 = %+v, %v", got, err)
	}
	if listed, err := s.ListUserSessions(owner); err != nil || len(listed) != 1 {
		t.Errorf("listed sessions after migrateFrom9 = %+v, %v", listed, err)
	}
}
