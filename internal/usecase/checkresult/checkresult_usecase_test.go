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

package checkresult_test

import (
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	checkresultUC "git.happydns.org/happyDomain/internal/usecase/checkresult"
	"git.happydns.org/happyDomain/model"
)

// newTestDB creates a fresh in-memory storage for each test.
func newTestDB(t *testing.T) storage.Storage {
	t.Helper()
	mem, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}
	db, err := kv.NewKVDatabase(mem)
	if err != nil {
		t.Fatalf("failed to create KV database: %v", err)
	}
	return db
}

func newTestCheckResultUsecase(db storage.Storage, maxResults int) *checkresultUC.CheckResultUsecase {
	return checkresultUC.NewCheckResultUsecase(db, &happydns.Options{MaxResultsPerCheck: maxResults})
}

// ---------------------------------------------------------------------------
// CreateCheckResult tests
// ---------------------------------------------------------------------------

func Test_CreateCheckResult_DefaultMaxResults(t *testing.T) {
	db := newTestDB(t)
	// MaxResultsPerCheck=0 → default 100; verify results are stored correctly.
	uc := newTestCheckResultUsecase(db, 0)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	result := &happydns.CheckResult{
		CheckerName: "checker",
		CheckType:   happydns.CheckScopeDomain,
		TargetId:    targetId,
		OwnerId:     ownerId,
		ExecutedAt:  time.Now(),
	}

	if err := uc.CreateCheckResult(result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Id.IsEmpty() {
		t.Error("expected result to have a non-empty ID after create")
	}

	stored, err := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId, 0)
	if err != nil {
		t.Fatalf("unexpected error listing results: %v", err)
	}
	if len(stored) != 1 {
		t.Errorf("expected 1 stored result, got %d", len(stored))
	}
}

func Test_CreateCheckResult_CustomMaxResults(t *testing.T) {
	db := newTestDB(t)
	// MaxResultsPerCheck=3: pre-seed 5 results, create 1 more → expect 3 to remain.
	uc := newTestCheckResultUsecase(db, 3)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	for i := range 5 {
		r := &happydns.CheckResult{
			CheckerName: "checker",
			CheckType:   happydns.CheckScopeDomain,
			TargetId:    targetId,
			OwnerId:     ownerId,
			ExecutedAt:  time.Now().Add(-time.Duration(5-i) * time.Minute),
		}
		if err := db.CreateCheckResult(r); err != nil {
			t.Fatalf("failed to seed result: %v", err)
		}
	}

	// Creating one more via the usecase triggers retention → prune to 3.
	if err := uc.CreateCheckResult(&happydns.CheckResult{
		CheckerName: "checker",
		CheckType:   happydns.CheckScopeDomain,
		TargetId:    targetId,
		OwnerId:     ownerId,
		ExecutedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remaining, err := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId, 0)
	if err != nil {
		t.Fatalf("unexpected error listing results: %v", err)
	}
	if len(remaining) != 3 {
		t.Errorf("expected 3 results after retention (MaxResultsPerCheck=3), got %d", len(remaining))
	}
}

func Test_CreateCheckResult_StoresResult(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	result := &happydns.CheckResult{
		CheckerName: "checker",
		CheckType:   happydns.CheckScopeDomain,
		TargetId:    targetId,
		OwnerId:     ownerId,
		ExecutedAt:  time.Now(),
	}

	if err := uc.CreateCheckResult(result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Id.IsEmpty() {
		t.Error("expected result to have a non-empty ID after create")
	}

	// Verify retrievable from storage.
	fetched, err := db.GetCheckResult("checker", happydns.CheckScopeDomain, targetId, result.Id)
	if err != nil {
		t.Fatalf("expected result to be retrievable: %v", err)
	}
	if fetched.Id.IsEmpty() {
		t.Error("retrieved result has empty ID")
	}
}

// ---------------------------------------------------------------------------
// ListCheckResultsByTarget tests
// ---------------------------------------------------------------------------

func Test_ListCheckResultsByTarget_DefaultLimit(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 100)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	for i := range 8 {
		r := &happydns.CheckResult{
			CheckerName: "checker",
			CheckType:   happydns.CheckScopeDomain,
			TargetId:    targetId,
			OwnerId:     ownerId,
			ExecutedAt:  time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := db.CreateCheckResult(r); err != nil {
			t.Fatalf("failed to seed result: %v", err)
		}
	}

	results, err := uc.ListCheckResultsByTarget("checker", happydns.CheckScopeDomain, targetId, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) > 5 {
		t.Errorf("expected at most 5 results (default limit), got %d", len(results))
	}
}

func Test_ListCheckResultsByTarget_CustomLimit(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 100)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	for i := range 4 {
		r := &happydns.CheckResult{
			CheckerName: "checker",
			CheckType:   happydns.CheckScopeDomain,
			TargetId:    targetId,
			OwnerId:     ownerId,
			ExecutedAt:  time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := db.CreateCheckResult(r); err != nil {
			t.Fatalf("failed to seed result: %v", err)
		}
	}

	results, err := uc.ListCheckResultsByTarget("checker", happydns.CheckScopeDomain, targetId, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results with limit=2, got %d", len(results))
	}
}

// ---------------------------------------------------------------------------
// DeleteAllCheckResults tests
// ---------------------------------------------------------------------------

func Test_DeleteAllCheckResults_Empty(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()

	if err := uc.DeleteAllCheckResults("checker", happydns.CheckScopeDomain, targetId); err != nil {
		t.Fatalf("unexpected error on empty store: %v", err)
	}
}

func Test_DeleteAllCheckResults_OnlyTargetDeleted(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 100)

	targetId1, _ := happydns.NewRandomIdentifier()
	targetId2, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	for range 3 {
		r := &happydns.CheckResult{
			CheckerName: "checker",
			CheckType:   happydns.CheckScopeDomain,
			TargetId:    targetId1,
			OwnerId:     ownerId,
			ExecutedAt:  time.Now(),
		}
		if err := db.CreateCheckResult(r); err != nil {
			t.Fatalf("failed to seed targetId1 result: %v", err)
		}
	}
	for range 2 {
		r := &happydns.CheckResult{
			CheckerName: "checker",
			CheckType:   happydns.CheckScopeDomain,
			TargetId:    targetId2,
			OwnerId:     ownerId,
			ExecutedAt:  time.Now(),
		}
		if err := db.CreateCheckResult(r); err != nil {
			t.Fatalf("failed to seed targetId2 result: %v", err)
		}
	}

	if err := uc.DeleteAllCheckResults("checker", happydns.CheckScopeDomain, targetId1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remaining1, _ := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId1, 0)
	if len(remaining1) != 0 {
		t.Errorf("expected 0 results for targetId1 after delete, got %d", len(remaining1))
	}

	remaining2, _ := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId2, 0)
	if len(remaining2) != 2 {
		t.Errorf("expected 2 results for targetId2 to remain, got %d", len(remaining2))
	}
}

// ---------------------------------------------------------------------------
// CreateCheckExecution tests
// ---------------------------------------------------------------------------

func Test_CreateCheckExecution_SetsStartedAt(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	execution := &happydns.CheckExecution{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
	}

	before := time.Now()
	if err := uc.CreateCheckExecution(execution); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if execution.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
	if execution.StartedAt.Before(before) || execution.StartedAt.After(after) {
		t.Errorf("StartedAt %v not in expected range [%v, %v]", execution.StartedAt, before, after)
	}
}

func Test_CreateCheckExecution_PreservesStartedAt(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()
	specificTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	execution := &happydns.CheckExecution{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		StartedAt:   specificTime,
	}

	if err := uc.CreateCheckExecution(execution); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !execution.StartedAt.Equal(specificTime) {
		t.Errorf("expected StartedAt to be preserved as %v, got %v", specificTime, execution.StartedAt)
	}
}

// ---------------------------------------------------------------------------
// CompleteCheckExecution tests
// ---------------------------------------------------------------------------

func Test_CompleteCheckExecution_SetsStatus(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()
	resultId, _ := happydns.NewRandomIdentifier()

	execution := &happydns.CheckExecution{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Status:      happydns.CheckExecutionRunning,
		StartedAt:   time.Now().Add(-time.Second),
	}
	if err := uc.CreateCheckExecution(execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	before := time.Now()
	if err := uc.CompleteCheckExecution(execution.Id, resultId); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	updated, err := db.GetCheckExecution(execution.Id)
	if err != nil {
		t.Fatalf("failed to retrieve execution: %v", err)
	}
	if updated.Status != happydns.CheckExecutionCompleted {
		t.Errorf("expected status Completed, got %v", updated.Status)
	}
	if updated.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	} else if updated.CompletedAt.Before(before) || updated.CompletedAt.After(after) {
		t.Errorf("CompletedAt %v not in expected range [%v, %v]", *updated.CompletedAt, before, after)
	}
	if updated.ResultId == nil || !updated.ResultId.Equals(resultId) {
		t.Error("expected ResultId to be set to the provided resultId")
	}
}

func Test_CompleteCheckExecution_NotFound(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	nonExistentId, _ := happydns.NewRandomIdentifier()
	resultId, _ := happydns.NewRandomIdentifier()

	if err := uc.CompleteCheckExecution(nonExistentId, resultId); err == nil {
		t.Fatal("expected error for non-existent execution ID")
	}
}

// ---------------------------------------------------------------------------
// FailCheckExecution tests
// ---------------------------------------------------------------------------

func Test_FailCheckExecution_CreatesErrorResult(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()

	execution := &happydns.CheckExecution{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Status:      happydns.CheckExecutionRunning,
		StartedAt:   time.Now().Add(-time.Second),
	}
	if err := uc.CreateCheckExecution(execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	if err := uc.FailCheckExecution(execution.Id, "something went wrong"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the check result was created.
	results, err := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId, 0)
	if err != nil {
		t.Fatalf("failed to list results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 check result to be created on failure, got %d", len(results))
	}
	result := results[0]
	if result.Status != happydns.CheckResultStatusCritical {
		t.Errorf("expected Status=KO, got %v", result.Status)
	}
	if result.Error != "something went wrong" {
		t.Errorf("expected Error='something went wrong', got %q", result.Error)
	}

	// Verify the execution status was updated.
	updated, err := db.GetCheckExecution(execution.Id)
	if err != nil {
		t.Fatalf("failed to retrieve execution: %v", err)
	}
	if updated.Status != happydns.CheckExecutionFailed {
		t.Errorf("expected execution status Failed, got %v", updated.Status)
	}
}

func Test_FailCheckExecution_ScheduledCheckFlag(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckResultUsecase(db, 10)

	targetId, _ := happydns.NewRandomIdentifier()
	ownerId, _ := happydns.NewRandomIdentifier()
	scheduleId, _ := happydns.NewRandomIdentifier()

	execution := &happydns.CheckExecution{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		ScheduleId:  &scheduleId,
		Status:      happydns.CheckExecutionRunning,
		StartedAt:   time.Now().Add(-time.Second),
	}
	if err := uc.CreateCheckExecution(execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	if err := uc.FailCheckExecution(execution.Id, "scheduled check failed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := db.ListCheckResults("checker", happydns.CheckScopeDomain, targetId, 0)
	if err != nil {
		t.Fatalf("failed to list results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].ScheduledCheck {
		t.Error("expected ScheduledCheck=true when execution has a non-nil ScheduleId")
	}
}
