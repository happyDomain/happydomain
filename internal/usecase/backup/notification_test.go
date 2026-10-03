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

package backup_test

import (
	"encoding/json"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/notifier"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/backup"
	"git.happydns.org/happyDomain/model"
)

// The administrative backup carries the notification channels and
// preferences: restored on a fresh instance, each user finds them back under
// the same identifiers, the preferences still pointing at their channels.
func TestBackupRestoreNotifications(t *testing.T) {
	src, user := seed(t)

	ch := &happydns.NotificationChannel{
		UserId:  user.Id,
		Type:    "webhook",
		Name:    "hook",
		Enabled: true,
		Config:  json.RawMessage(`{"url":"https://example.com/hook","secret":"s3cr3t"}`),
	}
	if err := src.CreateChannel(ch); err != nil {
		t.Fatal(err)
	}
	pref := &happydns.NotificationPreference{UserId: user.Id, ChannelIds: []happydns.Identifier{ch.Id}, MinStatus: 2, Enabled: true}
	if err := src.CreatePreference(pref); err != nil {
		t.Fatal(err)
	}

	dump := backup.NewUsecase(src).Backup()
	if len(dump.Errors) > 0 {
		t.Fatalf("backup errors: %v", dump.Errors)
	}
	if len(dump.NotificationChannels) != 1 || len(dump.NotificationPreferences) != 1 {
		t.Fatalf("backup holds %d channels and %d preferences, want 1 and 1", len(dump.NotificationChannels), len(dump.NotificationPreferences))
	}

	// Through JSON, as the admin API hands it out.
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	var restored happydns.Backup
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	if err := backup.NewUsecase(dst).Restore(&restored); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	chs, err := dst.ListChannelsByUser(user.Id)
	if err != nil || len(chs) != 1 {
		t.Fatalf("ListChannelsByUser = %v, %v; want the channel", chs, err)
	}
	if !chs[0].Id.Equals(ch.Id) || chs[0].Name != "hook" || string(chs[0].Config) != string(ch.Config) {
		t.Errorf("restored channel = %+v, want %+v", chs[0], ch)
	}

	prefs, err := dst.ListPreferencesByUser(user.Id)
	if err != nil || len(prefs) != 1 {
		t.Fatalf("ListPreferencesByUser = %v, %v; want the preference", prefs, err)
	}
	if !prefs[0].Id.Equals(pref.Id) || len(prefs[0].ChannelIds) != 1 || !prefs[0].ChannelIds[0].Equals(ch.Id) || prefs[0].MinStatus != 2 {
		t.Errorf("restored preference = %+v, want %+v", prefs[0], pref)
	}

	// Restored again over itself: still one of each.
	if err := backup.NewUsecase(dst).Restore(&restored); err != nil {
		t.Fatalf("second Restore: %v", err)
	}
	if chs, _ := dst.ListChannelsByUser(user.Id); len(chs) != 1 {
		t.Errorf("%d channels after restoring twice, want 1", len(chs))
	}
}

func exportRegistry() *notifier.Registry {
	r := notifier.NewRegistry()
	r.Register(notifier.Adapt(notifier.NewWebhookSender("https://dash.example.com", nil), nil))
	return r
}

// The export a user downloads carries every channel and preference, but a
// channel's secrets, a webhook signing secret say, are withheld.
func TestBackupUserRedactsChannelSecrets(t *testing.T) {
	db, user := seed(t)
	ch := &happydns.NotificationChannel{UserId: user.Id, Type: "webhook", Name: "hook", Enabled: true, Config: json.RawMessage(`{"url":"https://example.com/hook","secret":"s3cr3t"}`)}
	if err := db.CreateChannel(ch); err != nil {
		t.Fatal(err)
	}
	pref := &happydns.NotificationPreference{UserId: user.Id, ChannelIds: []happydns.Identifier{ch.Id}, MinStatus: 2, Enabled: true}
	if err := db.CreatePreference(pref); err != nil {
		t.Fatal(err)
	}

	ret := backup.NewUsecase(db, backup.WithChannelRedactor(exportRegistry())).BackupUser(user)
	if len(ret.Errors) > 0 {
		t.Fatalf("export errors: %v", ret.Errors)
	}
	if len(ret.NotificationChannels) != 1 || len(ret.NotificationPreferences) != 1 {
		t.Fatalf("export holds %d channels and %d preferences, want 1 and 1", len(ret.NotificationChannels), len(ret.NotificationPreferences))
	}

	got := ret.NotificationChannels[0]
	if got.Name != "hook" || !got.Id.Equals(ch.Id) {
		t.Errorf("exported channel = %+v, want %+v", got, ch)
	}
	if strings.Contains(string(got.Config), "s3cr3t") {
		t.Errorf("the exported channel carries its secret: %s", got.Config)
	}
	if !strings.Contains(string(got.Config), "https://example.com/hook") {
		t.Errorf("the exported channel lost its URL: %s", got.Config)
	}

	// The stored channel is left alone.
	if stored, _ := db.ListChannelsByUser(user.Id); len(stored) != 1 || !strings.Contains(string(stored[0].Config), "s3cr3t") {
		t.Errorf("the stored channel was altered: %+v", stored)
	}
}

// A channel of a type no sender knows cannot be told secret from not: its
// configuration is withheld whole rather than exported in clear.
func TestBackupUserWithholdsUnknownChannelConfig(t *testing.T) {
	db, user := seed(t)
	if err := db.CreateChannel(&happydns.NotificationChannel{UserId: user.Id, Type: "legacy", Config: json.RawMessage(`{"token":"s3cr3t"}`)}); err != nil {
		t.Fatal(err)
	}

	ret := backup.NewUsecase(db, backup.WithChannelRedactor(exportRegistry())).BackupUser(user)
	if len(ret.NotificationChannels) != 1 {
		t.Fatalf("export holds %d channels, want 1", len(ret.NotificationChannels))
	}
	if strings.Contains(string(ret.NotificationChannels[0].Config), "s3cr3t") {
		t.Errorf("the exported channel carries its secret: %s", ret.NotificationChannels[0].Config)
	}
}

// Without a way to redact, no channel leaves.
func TestBackupUserWithoutRedactorCarriesNoChannel(t *testing.T) {
	db, user := seed(t)
	if err := db.CreateChannel(&happydns.NotificationChannel{UserId: user.Id, Type: "webhook", Config: json.RawMessage(`{"url":"https://example.com","secret":"s3cr3t"}`)}); err != nil {
		t.Fatal(err)
	}

	ret := backup.NewUsecase(db).BackupUser(user)
	if len(ret.NotificationChannels) != 0 {
		t.Errorf("the user export carries %d channels", len(ret.NotificationChannels))
	}
}

// The notification states (acknowledgements included) and records travel in
// the backup, and come back under the same user.
func TestBackupRestoreNotificationStatesAndRecords(t *testing.T) {
	src, user := seed(t)

	target := happydns.CheckTarget{UserId: user.Id.String(), DomainId: "d1"}
	state := &happydns.NotificationState{CheckerID: "ping", Target: target, UserId: user.Id, LastStatus: 3, Acknowledged: true, AcknowledgedBy: "me", Annotation: "known"}
	if err := src.PutState(state); err != nil {
		t.Fatal(err)
	}
	rec := &happydns.NotificationRecord{UserId: user.Id, ChannelType: "webhook", CheckerID: "ping", Target: target, OldStatus: 1, NewStatus: 3, Success: true}
	if err := src.CreateRecord(rec); err != nil {
		t.Fatal(err)
	}

	dump := backup.NewUsecase(src).Backup()
	if len(dump.Errors) > 0 {
		t.Fatalf("backup errors: %v", dump.Errors)
	}
	if len(dump.NotificationStates) != 1 || len(dump.NotificationRecords) != 1 {
		t.Fatalf("backup holds %d states and %d records, want 1 and 1", len(dump.NotificationStates), len(dump.NotificationRecords))
	}

	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatal(err)
	}
	var restored happydns.Backup
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}

	dst, err := inmemory.Instantiate()
	if err != nil {
		t.Fatal(err)
	}
	if err := backup.NewUsecase(dst).Restore(&restored); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	got, err := dst.GetState("ping", target, user.Id)
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if !got.Acknowledged || got.AcknowledgedBy != "me" || got.Annotation != "known" || got.LastStatus != 3 {
		t.Errorf("restored state = %+v, want %+v", got, state)
	}

	recs, err := dst.ListRecordsByUser(user.Id, 0)
	if err != nil || len(recs) != 1 {
		t.Fatalf("ListRecordsByUser = %v, %v; want the record", recs, err)
	}
	if !recs[0].Id.Equals(rec.Id) || recs[0].NewStatus != 3 || !recs[0].Success {
		t.Errorf("restored record = %+v, want %+v", recs[0], rec)
	}

	if err := backup.NewUsecase(dst).Restore(&restored); err != nil {
		t.Fatalf("second Restore: %v", err)
	}
	if recs, _ := dst.ListRecordsByUser(user.Id, 0); len(recs) != 1 {
		t.Errorf("%d records after restoring twice, want 1", len(recs))
	}
}

// BackupUser carries only the states and records of that user.
func TestBackupUserCarriesItsStatesAndRecords(t *testing.T) {
	src, user := seed(t)
	other := happydns.Identifier{0x77}

	for _, owner := range []happydns.Identifier{user.Id, other} {
		if err := src.PutState(&happydns.NotificationState{CheckerID: "ping", UserId: owner}); err != nil {
			t.Fatal(err)
		}
		if err := src.CreateRecord(&happydns.NotificationRecord{UserId: owner, CheckerID: "ping"}); err != nil {
			t.Fatal(err)
		}
	}

	ret := backup.NewUsecase(src).BackupUser(user)
	if len(ret.Errors) > 0 {
		t.Fatalf("backup errors: %v", ret.Errors)
	}
	if len(ret.NotificationStates) != 1 || !ret.NotificationStates[0].UserId.Equals(user.Id) {
		t.Errorf("NotificationStates = %v, want only the user's", ret.NotificationStates)
	}
	if len(ret.NotificationRecords) != 1 || !ret.NotificationRecords[0].UserId.Equals(user.Id) {
		t.Errorf("NotificationRecords = %v, want only the user's", ret.NotificationRecords)
	}
}
