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

package database_test

import (
	"encoding/json"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	happydns "git.happydns.org/happyDomain/model"
)

func countAll[T any](t *testing.T, list func() (happydns.Iterator[T], error)) int {
	t.Helper()
	iter, err := list()
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	n := 0
	for iter.Next() {
		n++
	}
	return n
}

// seedPurgeData stores one of each piece of checker output, along with the
// data a purge must keep: a check plan, a checker configuration and a
// notification record.
func seedPurgeData(t *testing.T, s storage.Storage) happydns.CheckTarget {
	t.Helper()
	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}
	now := time.Now()

	plan := &happydns.CheckPlan{CheckerID: "purge_checker", Target: target}
	if err := s.CreateCheckPlan(plan); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateCheckerConfiguration("purge_checker", &uid, nil, nil, happydns.CheckerOptions{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateExecution(&happydns.Execution{CheckerID: "purge_checker", PlanID: &plan.Id, Target: target, StartedAt: now, Status: happydns.ExecutionDone}); err != nil {
		t.Fatal(err)
	}
	snap := &happydns.ObservationSnapshot{Target: target, CollectedAt: now, Data: map[happydns.ObservationKey]json.RawMessage{"obs": json.RawMessage(`{}`)}}
	if err := s.CreateSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateEvaluation(&happydns.CheckEvaluation{CheckerID: "purge_checker", PlanID: &plan.Id, Target: target, SnapshotID: snap.Id, EvaluatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutCachedObservation(target, "obs", &happydns.ObservationCacheEntry{SnapshotID: snap.Id, CollectedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceDiscoveryEntries("purge_producer", target, []happydns.DiscoveryEntry{{Type: "t", Ref: "r"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.PutDiscoveryObservationRef(&happydns.DiscoveryObservationRef{ProducerID: "purge_producer", Target: target, Ref: "r", ConsumerID: "purge_checker", Key: "obs", SnapshotID: snap.Id, CollectedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRecord(&happydns.NotificationRecord{UserId: uid, CheckerID: "purge_checker", Target: target, SentAt: now}); err != nil {
		t.Fatal(err)
	}
	return target
}

// A checker purge leaves nothing the checkers produce, and keeps the check
// plans, the checker options and the notification history.
func TestPurgeCheckerHistory(t *testing.T) {
	s := newStorage(t)
	seedPurgeData(t, s)

	n, err := s.PurgeCheckerHistory()
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Error("PurgeCheckerHistory reported no deleted key")
	}

	for name, got := range map[string]int{
		"executions":        countAll(t, s.ListAllExecutions),
		"evaluations":       countAll(t, s.ListAllEvaluations),
		"snapshots":         countAll(t, s.ListAllSnapshots),
		"cached obs":        countAll(t, s.ListAllCachedObservations),
		"discovery entries": countAll(t, s.ListAllDiscoveryEntries),
		"discovery refs":    countAll(t, s.ListAllDiscoveryObservationRefs),
	} {
		if got != 0 {
			t.Errorf("%d %s left after the purge", got, name)
		}
	}
	for name, got := range map[string]int{
		"check plans":          countAll(t, s.ListAllCheckPlans),
		"checker options":      countAll(t, s.ListAllCheckerConfigurations),
		"notification records": countAll(t, s.ListAllRecords),
	} {
		if got != 1 {
			t.Errorf("%d %s left after the purge, want 1", got, name)
		}
	}

	if n, err := s.PurgeCheckerHistory(); err != nil || n != 0 {
		t.Errorf("second PurgeCheckerHistory = %d, %v; want 0, nil", n, err)
	}
}

// Clearing the notification records drops every record with its user index,
// and nothing else.
func TestClearRecords(t *testing.T) {
	s := newStorage(t)
	target := seedPurgeData(t, s)
	uid, _ := happydns.NewIdentifierFromString(target.UserId)

	n, err := s.ClearRecords()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("ClearRecords deleted %d keys, want 2 (record and user index)", n)
	}
	if got := countAll(t, s.ListAllRecords); got != 0 {
		t.Errorf("%d records left", got)
	}
	if recs, err := s.ListRecordsByUser(uid, 10); err != nil || len(recs) != 0 {
		t.Errorf("ListRecordsByUser = %d records, %v; want none", len(recs), err)
	}
	if got := countAll(t, s.ListAllExecutions); got != 1 {
		t.Errorf("%d executions left, want 1", got)
	}
}

// Many keys go through several batches.
func TestPurgeCheckerHistoryManyBatches(t *testing.T) {
	s := newStorage(t)
	target := happydns.CheckTarget{UserId: "u"}
	for range 1200 {
		if err := s.CreateSnapshot(&happydns.ObservationSnapshot{Target: target, CollectedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.PurgeCheckerHistory()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1200 {
		t.Errorf("PurgeCheckerHistory deleted %d keys, want 1200", n)
	}
	if got := countAll(t, s.ListAllSnapshots); got != 0 {
		t.Errorf("%d snapshots left", got)
	}
}
