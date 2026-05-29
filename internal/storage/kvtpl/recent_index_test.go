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
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/model"
)

func newStorage(t *testing.T) storage.Storage {
	t.Helper()
	s, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("inmemory.Instantiate: %v", err)
	}
	return s
}

// TestExecutionRecentIndexOrdersNewestFirstWithLimit verifies that the
// time-sortable checker index returns executions newest first and that the
// limit is applied during the scan, not just as a post-sort truncation.
func TestExecutionRecentIndexOrdersNewestFirstWithLimit(t *testing.T) {
	s := newStorage(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	base := time.Now().Truncate(time.Second)
	// Insert out of chronological order to prove ordering comes from the key,
	// not the insertion sequence.
	offsets := []int{2, 0, 4, 1, 3}
	for _, off := range offsets {
		exec := &happydns.Execution{
			CheckerID: "recent_test_checker",
			Target:    target,
			StartedAt: base.Add(time.Duration(off) * time.Minute),
			Status:    happydns.ExecutionDone,
		}
		if err := s.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution: %v", err)
		}
	}

	got, err := s.ListExecutionsByChecker("recent_test_checker", target, 3, nil)
	if err != nil {
		t.Fatalf("ListExecutionsByChecker: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 executions (limit), got %d", len(got))
	}

	// Newest first: offsets 4, 3, 2.
	wantOffsets := []int{4, 3, 2}
	for i, e := range got {
		want := base.Add(time.Duration(wantOffsets[i]) * time.Minute)
		if !e.StartedAt.Equal(want) {
			t.Errorf("position %d: StartedAt = %s, want %s", i, e.StartedAt, want)
		}
		if i > 0 && got[i-1].StartedAt.Before(e.StartedAt) {
			t.Errorf("ordering broken: %s precedes %s", got[i-1].StartedAt, e.StartedAt)
		}
	}
}

// TestExecutionRecentIndexFilterWithLimit checks that the filter predicate is
// honoured while the limit still bounds the number of matches collected.
func TestExecutionRecentIndexFilterWithLimit(t *testing.T) {
	s := newStorage(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String()}

	base := time.Now().Truncate(time.Second)
	for i := 0; i < 6; i++ {
		status := happydns.ExecutionDone
		if i%2 == 0 {
			status = happydns.ExecutionRunning
		}
		exec := &happydns.Execution{
			CheckerID: "filter_test_checker",
			Target:    target,
			StartedAt: base.Add(time.Duration(i) * time.Minute),
			Status:    status,
		}
		if err := s.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution: %v", err)
		}
	}

	done := func(e *happydns.Execution) bool { return e.Status == happydns.ExecutionDone }
	got, err := s.ListExecutionsByUser(uid, 2, done)
	if err != nil {
		t.Fatalf("ListExecutionsByUser: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 filtered executions, got %d", len(got))
	}
	for _, e := range got {
		if e.Status != happydns.ExecutionDone {
			t.Errorf("filter leaked a non-done execution: %v", e.Status)
		}
	}
	// Newest done executions are at odd minutes 5 and 3.
	if !got[0].StartedAt.Equal(base.Add(5*time.Minute)) || !got[1].StartedAt.Equal(base.Add(3*time.Minute)) {
		t.Errorf("unexpected order: %s, %s", got[0].StartedAt, got[1].StartedAt)
	}
}

// TestGetLatestEvaluationReadsMostRecent verifies GetLatestEvaluation returns
// the evaluation with the greatest EvaluatedAt via the time-sortable plan index.
func TestGetLatestEvaluationReadsMostRecent(t *testing.T) {
	s := newStorage(t)

	planID, _ := happydns.NewRandomIdentifier()
	base := time.Now().Truncate(time.Second)
	var newestID happydns.Identifier
	for i, off := range []int{1, 5, 3} {
		eval := &happydns.CheckEvaluation{
			PlanID:      &planID,
			CheckerID:   "eval_test_checker",
			Target:      happydns.CheckTarget{UserId: "u1"},
			EvaluatedAt: base.Add(time.Duration(off) * time.Minute),
		}
		if err := s.CreateEvaluation(eval); err != nil {
			t.Fatalf("CreateEvaluation: %v", err)
		}
		if i == 1 { // offset 5 is the newest
			newestID = eval.Id
		}
	}

	latest, err := s.GetLatestEvaluation(planID)
	if err != nil {
		t.Fatalf("GetLatestEvaluation: %v", err)
	}
	if !latest.Id.Equals(newestID) {
		t.Errorf("GetLatestEvaluation returned %s, want newest %s", latest.Id.String(), newestID.String())
	}
	if !latest.EvaluatedAt.Equal(base.Add(5 * time.Minute)) {
		t.Errorf("latest EvaluatedAt = %s, want %s", latest.EvaluatedAt, base.Add(5*time.Minute))
	}
}
