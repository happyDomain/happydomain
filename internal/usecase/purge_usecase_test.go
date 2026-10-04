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

package usecase_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase"
	"git.happydns.org/happyDomain/internal/usecase/session"
	"git.happydns.org/happyDomain/model"
)

func TestPurge(t *testing.T) {
	db, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}

	owner := newTestId(t)
	if err := db.CreateOrUpdateUser(&happydns.User{Id: owner, Email: "owner@example.com"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	newSession := func(expiresOn time.Time) *happydns.Session {
		s := &happydns.Session{Id: happydns.SessionIDFromToken(session.NewSessionID()), IdUser: owner, IssuedAt: now, ExpiresOn: expiresOn}
		if err := db.UpdateSession(s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	expired := newSession(now.Add(-time.Hour))
	valid := newSession(now.Add(time.Hour))

	if err := db.CreateSnapshot(&happydns.ObservationSnapshot{Target: happydns.CheckTarget{UserId: owner.String()}, CollectedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateRecord(&happydns.NotificationRecord{UserId: owner, SentAt: now}); err != nil {
		t.Fatal(err)
	}

	pu := usecase.NewPurgeUsecase(db)

	// Nothing selected, nothing deleted.
	report, err := pu.Purge(happydns.PurgeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if *report != (happydns.PurgeReport{}) {
		t.Errorf("empty purge reported %+v", report)
	}
	if _, err := db.GetSession(expired.Id); err != nil {
		t.Errorf("empty purge deleted the expired session: %v", err)
	}

	report, err = pu.Purge(happydns.PurgeOptions{Checkers: true, ExpiredSessions: true, NotificationRecords: true, Compact: true})
	if err != nil {
		t.Fatal(err)
	}
	want := happydns.PurgeReport{Checkers: 1, ExpiredSessions: 1, NotificationRecords: 2, Compacted: false}
	if *report != want {
		t.Errorf("Purge reported %+v, want %+v", *report, want)
	}

	if _, err := db.GetSession(expired.Id); err == nil {
		t.Error("the expired session is still there")
	}
	if _, err := db.GetSession(valid.Id); err != nil {
		t.Errorf("the valid session is gone: %v", err)
	}
	if sessions, err := db.ListUserSessions(owner); err != nil || len(sessions) != 1 {
		t.Errorf("ListUserSessions = %d sessions, %v; want only the valid one", len(sessions), err)
	}
}
