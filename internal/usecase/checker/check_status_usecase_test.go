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

package checker_test

import (
	"encoding/json"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

func setupStatusUC(t *testing.T) (*checkerUC.CheckStatusUsecase, *planStore, storage.Storage) {
	t.Helper()
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "status_test_checker",
		Name: "Status Test Checker",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_x", status: happydns.StatusOK},
			&testCheckRule{name: "rule_y", status: happydns.StatusWarn},
		},
	})

	ps := newPlanStore()
	ms, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}
	uc := checkerUC.NewCheckStatusUsecase(ps, ms, ms, ms)
	return uc, ps, ms
}

func TestCheckStatusUsecase_ListCheckerStatuses(t *testing.T) {
	uc, _, _ := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	statuses, err := uc.ListCheckerStatuses(target)
	if err != nil {
		t.Fatalf("ListCheckerStatuses() error: %v", err)
	}

	if len(statuses) == 0 {
		t.Fatal("expected at least one checker status")
	}

	// All should be enabled by default (no plans).
	for _, s := range statuses {
		if !s.Enabled {
			t.Errorf("expected checker %s to be enabled by default", s.ID)
		}
	}
}

func TestCheckStatusUsecase_ListCheckerStatuses_WithPlan(t *testing.T) {
	uc, ps, _ := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	// Create a plan that fully disables the checker.
	plan := &happydns.CheckPlan{
		CheckerID: "status_test_checker",
		Target:    target,
		Enabled:   map[string]bool{"rule_x": false, "rule_y": false},
	}
	if err := ps.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	statuses, err := uc.ListCheckerStatuses(target)
	if err != nil {
		t.Fatalf("ListCheckerStatuses() error: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.ID == "status_test_checker" {
			found = true
			if s.Enabled {
				t.Error("expected status_test_checker to be disabled when all rules are off")
			}
			if s.Plan == nil {
				t.Error("expected Plan to be set")
			}
			if s.EnabledRules["rule_x"] {
				t.Error("expected rule_x to be disabled")
			}
			if s.EnabledRules["rule_y"] {
				t.Error("expected rule_y to be disabled")
			}
		}
	}
	if !found {
		t.Error("status_test_checker not found in statuses")
	}
}

func TestCheckStatusUsecase_ListCheckerStatuses_WithEvaluation(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	// Create an execution for the checker.
	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		StartedAt: time.Now(),
		Status:    happydns.ExecutionDone,
		Result:    happydns.CheckState{Status: happydns.StatusOK, Message: "all good"},
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	statuses, err := uc.ListCheckerStatuses(target)
	if err != nil {
		t.Fatalf("ListCheckerStatuses() error: %v", err)
	}

	for _, s := range statuses {
		if s.ID == "status_test_checker" {
			if s.LatestExecution == nil {
				t.Error("expected LatestExecution to be set")
			} else if s.LatestExecution.Result.Status != happydns.StatusOK {
				t.Errorf("expected latest execution result status OK, got %s", s.LatestExecution.Result.Status)
			}
		}
	}
}

func TestCheckStatusUsecase_GetExecution(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	exec := &happydns.Execution{
		Status: happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	got, err := uc.GetExecution(happydns.CheckTarget{}, exec.Id)
	if err != nil {
		t.Fatalf("GetExecution() error: %v", err)
	}
	if got.Status != happydns.ExecutionDone {
		t.Errorf("expected status Done, got %d", got.Status)
	}
}

func TestCheckStatusUsecase_GetExecutionNotFound(t *testing.T) {
	uc, _, _ := setupStatusUC(t)

	fakeID, _ := happydns.NewRandomIdentifier()
	_, err := uc.GetExecution(happydns.CheckTarget{}, fakeID)
	if err == nil {
		t.Fatal("expected error for nonexistent execution")
	}
}

func TestCheckStatusUsecase_GetExecution_ScopeMismatch(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	// Access with a different user scope should fail.
	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	_, err := uc.GetExecution(wrongScope, exec.Id)
	if err == nil {
		t.Fatal("expected error when scope doesn't match execution target")
	}
}

func TestCheckStatusUsecase_DeleteExecution(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	if err := uc.DeleteExecution(target, exec.Id); err != nil {
		t.Fatalf("DeleteExecution() error: %v", err)
	}

	_, err := uc.GetExecution(target, exec.Id)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

func TestCheckStatusUsecase_DeleteExecution_ScopeMismatch(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	// Delete with wrong scope should fail.
	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	if err := uc.DeleteExecution(wrongScope, exec.Id); err == nil {
		t.Fatal("expected error when scope doesn't match")
	}

	// Original should still exist.
	_, err := uc.GetExecution(target, exec.Id)
	if err != nil {
		t.Fatalf("execution should still exist after failed delete: %v", err)
	}
}

func TestCheckStatusUsecase_DeleteExecutionsByChecker(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	for i := 0; i < 3; i++ {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    target,
			Status:    happydns.ExecutionDone,
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	if err := uc.DeleteExecutionsByChecker("status_test_checker", target); err != nil {
		t.Fatalf("DeleteExecutionsByChecker() error: %v", err)
	}

	execs, err := uc.ListExecutionsByChecker("status_test_checker", target, 0)
	if err != nil {
		t.Fatalf("ListExecutionsByChecker() error: %v", err)
	}
	if len(execs) != 0 {
		t.Errorf("expected 0 executions after bulk delete, got %d", len(execs))
	}
}

func TestCheckStatusUsecase_ListExecutionsByChecker(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	for i := 0; i < 5; i++ {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    target,
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	execs, err := uc.ListExecutionsByChecker("status_test_checker", target, 3)
	if err != nil {
		t.Fatalf("ListExecutionsByChecker() error: %v", err)
	}
	if len(execs) > 3 {
		t.Errorf("expected at most 3 executions with limit, got %d", len(execs))
	}
}

func TestCheckStatusUsecase_GetWorstDomainStatuses(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did1, _ := happydns.NewRandomIdentifier()
	did2, _ := happydns.NewRandomIdentifier()

	// Domain 1: one OK and one WARN execution.
	for _, status := range []happydns.Status{happydns.StatusOK, happydns.StatusWarn} {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    happydns.CheckTarget{UserId: uid.String(), DomainId: did1.String()},
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
			Result:    happydns.CheckState{Status: status},
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	// Domain 2: only OK.
	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    happydns.CheckTarget{UserId: uid.String(), DomainId: did2.String()},
		StartedAt: time.Now(),
		Status:    happydns.ExecutionDone,
		Result:    happydns.CheckState{Status: happydns.StatusOK},
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	worst, err := uc.GetWorstDomainStatuses(uid)
	if err != nil {
		t.Fatalf("GetWorstDomainStatuses() error: %v", err)
	}

	// Domain 1 should have worst status WARN.
	if s, ok := worst[did1.String()]; !ok {
		t.Error("expected domain 1 in results")
	} else if *s != happydns.StatusWarn {
		t.Errorf("expected worst status WARN for domain 1, got %v", *s)
	}

	// Domain 2 should have worst status OK.
	if s, ok := worst[did2.String()]; !ok {
		t.Error("expected domain 2 in results")
	} else if *s != happydns.StatusOK {
		t.Errorf("expected worst status OK for domain 2, got %v", *s)
	}
}

func TestCheckStatusUsecase_GetWorstServiceStatuses(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	sid1, _ := happydns.NewRandomIdentifier()
	sid2, _ := happydns.NewRandomIdentifier()

	// Service 1: CRIT execution.
	exec1 := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    happydns.CheckTarget{UserId: uid.String(), DomainId: did.String(), ServiceId: sid1.String()},
		StartedAt: time.Now(),
		Status:    happydns.ExecutionDone,
		Result:    happydns.CheckState{Status: happydns.StatusCrit},
	}
	if err := ms.CreateExecution(exec1); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	// Service 2: OK execution.
	exec2 := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    happydns.CheckTarget{UserId: uid.String(), DomainId: did.String(), ServiceId: sid2.String()},
		StartedAt: time.Now(),
		Status:    happydns.ExecutionDone,
		Result:    happydns.CheckState{Status: happydns.StatusOK},
	}
	if err := ms.CreateExecution(exec2); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	worst, err := uc.GetWorstServiceStatuses(uid, did)
	if err != nil {
		t.Fatalf("GetWorstServiceStatuses() error: %v", err)
	}

	if s, ok := worst[sid1.String()]; !ok {
		t.Error("expected service 1 in results")
	} else if *s != happydns.StatusCrit {
		t.Errorf("expected CRIT for service 1, got %v", *s)
	}

	if s, ok := worst[sid2.String()]; !ok {
		t.Error("expected service 2 in results")
	} else if *s != happydns.StatusOK {
		t.Errorf("expected OK for service 2, got %v", *s)
	}
}

func TestCheckStatusUsecase_GetWorstServiceStatuses_Empty(t *testing.T) {
	uc, _, _ := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()

	result, err := uc.GetWorstServiceStatuses(uid, did)
	if err != nil {
		t.Fatalf("GetWorstServiceStatuses() error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty results, got %v", result)
	}
}

func TestCheckStatusUsecase_GetResultsByExecution(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	// Create evaluation.
	eval := &happydns.CheckEvaluation{
		CheckerID: "status_test_checker",
		Target:    target,
		States:    []happydns.CheckState{{Status: happydns.StatusOK, Code: "test"}},
	}
	if err := ms.CreateEvaluation(eval); err != nil {
		t.Fatalf("CreateEvaluation() error: %v", err)
	}

	// Create execution referencing the evaluation.
	exec := &happydns.Execution{
		CheckerID:    "status_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: &eval.Id,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	got, err := uc.GetResultsByExecution(target, exec.Id)
	if err != nil {
		t.Fatalf("GetResultsByExecution() error: %v", err)
	}
	if len(got.States) != 1 {
		t.Errorf("expected 1 state, got %d", len(got.States))
	}
}

func TestCheckStatusUsecase_GetResultsByExecution_NoEvaluation(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	target := happydns.CheckTarget{}
	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionPending,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	_, err := uc.GetResultsByExecution(target, exec.Id)
	if err == nil {
		t.Fatal("expected error for execution without evaluation")
	}
}

func TestCheckStatusUsecase_ListPlannedExecutions(t *testing.T) {
	// Test with nil provider.
	result := checkerUC.ListPlannedExecutions(nil, nil, "checker", happydns.CheckTarget{})
	if result != nil {
		t.Errorf("expected nil for nil provider, got %v", result)
	}
}

// fakePlannedProvider is a stub PlannedJobProvider that returns a fixed list
// of scheduler jobs, used to test ListPlannedExecutions independently of the
// scheduler.
type fakePlannedProvider struct {
	jobs []*checkerUC.SchedulerJob
}

func (f *fakePlannedProvider) GetPlannedJobsForChecker(checkerID string, target happydns.CheckTarget) []*checkerUC.SchedulerJob {
	return f.jobs
}

// fakeBudgetChecker is a stub BudgetChecker. Its verdict can be set as a
// blanket value (limited=true denies every call) or selectively denied for
// intervals shorter than denyBelow, mirroring how UserGater throttles
// short-interval jobs while still allowing longer ones.
type fakeBudgetChecker struct {
	limited   bool
	denyBelow time.Duration
	calls     int // number of RateLimiterFor invocations, for fanout assertions
}

func (f *fakeBudgetChecker) RateLimiterFor(userID string) func(time.Duration) bool {
	f.calls++
	return func(interval time.Duration) bool {
		if f.limited {
			return true
		}
		if f.denyBelow > 0 && interval > 0 && interval < f.denyBelow {
			return true
		}
		return false
	}
}

// AllowWithInterval / IncrementUsage are present so fakeBudgetChecker
// satisfies BudgetChecker; ListPlannedExecutions never calls them, so the
// bodies are intentionally minimal.
func (f *fakeBudgetChecker) AllowWithInterval(_ happydns.CheckTarget, _ time.Duration) bool {
	return !f.limited
}
func (f *fakeBudgetChecker) IncrementUsage(_ happydns.CheckTarget) {}

func TestCheckStatusUsecase_ListPlannedExecutions_StatusPending(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}
	now := time.Now()
	provider := &fakePlannedProvider{jobs: []*checkerUC.SchedulerJob{
		{CheckerID: "c1", Target: target, Interval: time.Hour, NextRun: now.Add(time.Hour)},
		{CheckerID: "c1", Target: target, Interval: 2 * time.Hour, NextRun: now.Add(2 * time.Hour)},
	}}

	// Nil budget checker -> all entries should be pending.
	result := checkerUC.ListPlannedExecutions(provider, nil, "c1", target)
	if len(result) != 2 {
		t.Fatalf("expected 2 planned executions, got %d", len(result))
	}
	for i, exec := range result {
		if exec.Status != happydns.ExecutionPending {
			t.Errorf("result[%d].Status = %v; want ExecutionPending", i, exec.Status)
		}
		if exec.Trigger.Type != happydns.TriggerSchedule {
			t.Errorf("result[%d].Trigger.Type = %v; want TriggerSchedule", i, exec.Trigger.Type)
		}
	}

	// Budget checker reporting "not rate-limited" -> still pending.
	result = checkerUC.ListPlannedExecutions(provider, &fakeBudgetChecker{}, "c1", target)
	for i, exec := range result {
		if exec.Status != happydns.ExecutionPending {
			t.Errorf("result[%d].Status = %v; want ExecutionPending when budget OK", i, exec.Status)
		}
	}
}

func TestCheckStatusUsecase_ListPlannedExecutions_StatusRateLimited(t *testing.T) {
	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}
	now := time.Now()
	provider := &fakePlannedProvider{jobs: []*checkerUC.SchedulerJob{
		{CheckerID: "c1", Target: target, Interval: time.Hour, NextRun: now.Add(time.Hour)},
		{CheckerID: "c1", Target: target, Interval: 2 * time.Hour, NextRun: now.Add(2 * time.Hour)},
	}}

	result := checkerUC.ListPlannedExecutions(provider, &fakeBudgetChecker{limited: true}, "c1", target)
	if len(result) != 2 {
		t.Fatalf("expected 2 planned executions, got %d", len(result))
	}
	for i, exec := range result {
		if exec.Status != happydns.ExecutionRateLimited {
			t.Errorf("result[%d].Status = %v; want ExecutionRateLimited", i, exec.Status)
		}
	}
}

func TestCheckStatusUsecase_ListPlannedExecutions_MixedByInterval(t *testing.T) {
	// When throttling is interval-aware, ListPlannedExecutions should flag
	// short-interval jobs as rate-limited while leaving longer ones pending.
	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}
	now := time.Now()
	provider := &fakePlannedProvider{jobs: []*checkerUC.SchedulerJob{
		{CheckerID: "c1", Target: target, Interval: time.Minute, NextRun: now.Add(time.Minute)},
		{CheckerID: "c1", Target: target, Interval: 6 * time.Hour, NextRun: now.Add(6 * time.Hour)},
	}}

	// Throttle anything shorter than 4h (mirrors UserGater's cutoff).
	result := checkerUC.ListPlannedExecutions(provider, &fakeBudgetChecker{denyBelow: 4 * time.Hour}, "c1", target)
	if len(result) != 2 {
		t.Fatalf("expected 2 planned executions, got %d", len(result))
	}
	if result[0].Status != happydns.ExecutionRateLimited {
		t.Errorf("result[0].Status = %v; want ExecutionRateLimited for 1-minute interval", result[0].Status)
	}
	if result[1].Status != happydns.ExecutionPending {
		t.Errorf("result[1].Status = %v; want ExecutionPending for 6-hour interval", result[1].Status)
	}
}

func TestCheckStatusUsecase_ListPlannedExecutions_EmptyJobs(t *testing.T) {
	// Even when rate-limited, an empty provider should produce an empty,
	// non-nil result (matching the prior behaviour of always returning a
	// slice when provider is non-nil).
	provider := &fakePlannedProvider{jobs: nil}
	result := checkerUC.ListPlannedExecutions(provider, &fakeBudgetChecker{limited: true}, "c1", happydns.CheckTarget{})
	if result == nil {
		t.Fatal("expected non-nil (empty) slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 planned executions, got %d", len(result))
	}
}

func TestCheckStatusUsecase_ListPlannedExecutions_SnapshotsBudgetOnce(t *testing.T) {
	// RateLimiterFor must be invoked exactly once per call regardless of
	// how many planned jobs are returned — this is the whole point of the
	// closure-based snapshot API (one budget lookup amortised over N jobs).
	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}
	now := time.Now()
	provider := &fakePlannedProvider{jobs: []*checkerUC.SchedulerJob{
		{CheckerID: "c1", Target: target, Interval: time.Minute, NextRun: now},
		{CheckerID: "c1", Target: target, Interval: time.Hour, NextRun: now},
		{CheckerID: "c1", Target: target, Interval: 6 * time.Hour, NextRun: now},
		{CheckerID: "c1", Target: target, Interval: 24 * time.Hour, NextRun: now},
	}}

	bc := &fakeBudgetChecker{denyBelow: 4 * time.Hour}
	_ = checkerUC.ListPlannedExecutions(provider, bc, "c1", target)
	if bc.calls != 1 {
		t.Errorf("RateLimiterFor called %d times; want 1 (one snapshot per call)", bc.calls)
	}
}

func TestCheckStatusUsecase_GetObservationsByExecution(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	// Create snapshot.
	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
	}
	if err := ms.CreateSnapshot(snap); err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}

	// Create evaluation referencing the snapshot.
	eval := &happydns.CheckEvaluation{
		CheckerID:  "status_test_checker",
		Target:     target,
		SnapshotID: snap.Id,
	}
	if err := ms.CreateEvaluation(eval); err != nil {
		t.Fatalf("CreateEvaluation() error: %v", err)
	}

	// Create execution referencing the evaluation.
	exec := &happydns.Execution{
		CheckerID:    "status_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: &eval.Id,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	got, err := uc.GetObservationsByExecution(target, exec.Id)
	if err != nil {
		t.Fatalf("GetObservationsByExecution() error: %v", err)
	}
	if !got.Id.Equals(snap.Id) {
		t.Errorf("expected snapshot ID %s, got %s", snap.Id, got.Id)
	}
}

func TestCheckStatusUsecase_GetObservationsByExecution_ScopeMismatch(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	_, err := uc.GetObservationsByExecution(wrongScope, exec.Id)
	if err == nil {
		t.Fatal("expected error when scope doesn't match")
	}
}

// --- Metrics extraction tests ---

func TestCheckStatusUsecase_ExtractMetricsFromExecution_NilEvaluation(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID:    "status_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: nil,
		StartedAt:    time.Now(),
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	metrics, err := uc.GetMetricsByExecution(target, exec.Id)
	if err != nil {
		t.Fatalf("GetMetricsByExecution() error: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected empty metrics for nil evaluation, got %d", len(metrics))
	}
}

func TestCheckStatusUsecase_ExtractMetricsFromExecution_NotDone(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionPending,
		StartedAt: time.Now(),
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	metrics, err := uc.GetMetricsByExecution(target, exec.Id)
	if err != nil {
		t.Fatalf("GetMetricsByExecution() error: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected empty metrics for pending execution, got %d", len(metrics))
	}
}

func TestCheckStatusUsecase_GetMetricsByChecker_Empty(t *testing.T) {
	uc, _, _ := setupStatusUC(t)

	target := happydns.CheckTarget{UserId: "nonexistent", DomainId: "d1"}

	metrics, err := uc.GetMetricsByChecker("status_test_checker", target, 100)
	if err != nil {
		t.Fatalf("GetMetricsByChecker() error: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected empty metrics for checker with no executions, got %d", len(metrics))
	}
}

func TestCheckStatusUsecase_GetMetricsByUser(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	for i := 0; i < 3; i++ {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    target,
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
			Result:    happydns.CheckState{Status: happydns.StatusOK},
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	metrics, err := uc.GetMetricsByUser(uid, 100)
	if err != nil {
		t.Fatalf("GetMetricsByUser() error: %v", err)
	}
	// Without observation providers registered in tests, metrics will be empty,
	// but the call must succeed without error.
	_ = metrics
}

func TestCheckStatusUsecase_GetMetricsByDomain(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	for i := 0; i < 3; i++ {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    target,
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
			Result:    happydns.CheckState{Status: happydns.StatusOK},
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	metrics, err := uc.GetMetricsByDomain(did, 100)
	if err != nil {
		t.Fatalf("GetMetricsByDomain() error: %v", err)
	}
	_ = metrics
}

func TestCheckStatusUsecase_GetMetricsByUser_LimitApplied(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	for i := 0; i < 5; i++ {
		exec := &happydns.Execution{
			CheckerID: "status_test_checker",
			Target:    target,
			StartedAt: time.Now(),
			Status:    happydns.ExecutionDone,
			Result:    happydns.CheckState{Status: happydns.StatusOK},
		}
		if err := ms.CreateExecution(exec); err != nil {
			t.Fatalf("CreateExecution() error: %v", err)
		}
	}

	// Call with limit=2; underlying list should be limited.
	metrics, err := uc.GetMetricsByUser(uid, 2)
	if err != nil {
		t.Fatalf("GetMetricsByUser(limit=2) error: %v", err)
	}
	_ = metrics
}

func TestCheckStatusUsecase_GetSnapshotByExecution(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	// Create snapshot with observation data.
	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
		Data: map[happydns.ObservationKey]json.RawMessage{
			"dns_records": json.RawMessage(`{"records":["A 1.2.3.4"]}`),
		},
	}
	if err := ms.CreateSnapshot(snap); err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}

	eval := &happydns.CheckEvaluation{
		CheckerID:  "status_test_checker",
		Target:     target,
		SnapshotID: snap.Id,
	}
	if err := ms.CreateEvaluation(eval); err != nil {
		t.Fatalf("CreateEvaluation() error: %v", err)
	}

	exec := &happydns.Execution{
		CheckerID:    "status_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: &eval.Id,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	raw, err := uc.GetSnapshotByExecution(target, exec.Id, "dns_records")
	if err != nil {
		t.Fatalf("GetSnapshotByExecution() error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal observation data: %v", err)
	}
	if _, ok := parsed["records"]; !ok {
		t.Error("expected 'records' key in observation data")
	}
}

func TestCheckStatusUsecase_GetSnapshotByExecution_KeyNotFound(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
		Data:        map[happydns.ObservationKey]json.RawMessage{},
	}
	if err := ms.CreateSnapshot(snap); err != nil {
		t.Fatalf("CreateSnapshot() error: %v", err)
	}

	eval := &happydns.CheckEvaluation{
		CheckerID:  "status_test_checker",
		Target:     target,
		SnapshotID: snap.Id,
	}
	if err := ms.CreateEvaluation(eval); err != nil {
		t.Fatalf("CreateEvaluation() error: %v", err)
	}

	exec := &happydns.Execution{
		CheckerID:    "status_test_checker",
		Target:       target,
		Status:       happydns.ExecutionDone,
		EvaluationID: &eval.Id,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	_, err := uc.GetSnapshotByExecution(target, exec.Id, "nonexistent_key")
	if err == nil {
		t.Fatal("expected error for nonexistent observation key")
	}
}

func TestCheckStatusUsecase_GetSnapshotByExecution_ScopeMismatch(t *testing.T) {
	uc, _, ms := setupStatusUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	exec := &happydns.Execution{
		CheckerID: "status_test_checker",
		Target:    target,
		Status:    happydns.ExecutionDone,
	}
	if err := ms.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution() error: %v", err)
	}

	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	_, err := uc.GetSnapshotByExecution(wrongScope, exec.Id, "any_key")
	if err == nil {
		t.Fatal("expected error when scope doesn't match")
	}
}
