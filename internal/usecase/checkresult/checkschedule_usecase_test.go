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
	"fmt"
	"testing"
	"time"

	"git.happydns.org/happyDomain/internal/storage"
	checkresultUC "git.happydns.org/happyDomain/internal/usecase/checkresult"
	"git.happydns.org/happyDomain/model"
)

// ---------------------------------------------------------------------------
// mockCheckerUsecase – minimal CheckerUsecase backed by a fixed map.
// ---------------------------------------------------------------------------

type mockCheckerUsecase struct {
	checkers map[string]happydns.Checker
}

func (m *mockCheckerUsecase) ListCheckers() (*map[string]happydns.Checker, error) {
	return &m.checkers, nil
}

func (m *mockCheckerUsecase) GetChecker(name string) (happydns.Checker, error) {
	c, ok := m.checkers[name]
	if !ok {
		return nil, fmt.Errorf("checker not found: %s", name)
	}
	return c, nil
}

func (m *mockCheckerUsecase) GetCheckerOptions(name string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (*happydns.CheckerOptions, error) {
	return nil, nil
}

func (m *mockCheckerUsecase) BuildMergedCheckerOptions(name string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, runOpts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
	return runOpts, nil
}

func (m *mockCheckerUsecase) GetStoredCheckerOptionsNoDefault(name string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier) (happydns.CheckerOptions, error) {
	return make(happydns.CheckerOptions), nil
}

func (m *mockCheckerUsecase) SetCheckerOptions(name string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	return nil
}

func (m *mockCheckerUsecase) OverwriteSomeCheckerOptions(name string, userid *happydns.Identifier, domainid *happydns.Identifier, serviceid *happydns.Identifier, opts happydns.CheckerOptions) error {
	return nil
}

// ---------------------------------------------------------------------------
// mockDomainChecker – Checker with configurable Availability.
// ---------------------------------------------------------------------------

type mockDomainChecker struct {
	name         string
	applyDomain  bool
	applyService bool
}

func (m *mockDomainChecker) ID() string   { return m.name }
func (m *mockDomainChecker) Name() string { return m.name }
func (m *mockDomainChecker) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{ApplyToDomain: m.applyDomain, ApplyToService: m.applyService}
}
func (m *mockDomainChecker) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{}
}
func (m *mockDomainChecker) RunCheck(opts happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Constructor helper
// ---------------------------------------------------------------------------

func newTestCheckScheduleUsecase(db storage.Storage, checkerUC happydns.CheckerUsecase) *checkresultUC.CheckScheduleUsecase {
	return checkresultUC.NewCheckScheduleUsecase(db, &happydns.Options{}, db, checkerUC)
}

// seedSchedule creates and stores a schedule in the db, returning it with its assigned ID.
func seedSchedule(t *testing.T, db storage.Storage, interval time.Duration) *happydns.CheckerSchedule {
	t.Helper()
	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	s := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Interval:    interval,
		Enabled:     true,
		NextRun:     time.Now().Add(interval),
	}
	if err := db.CreateCheckerSchedule(s); err != nil {
		t.Fatalf("failed to seed schedule: %v", err)
	}
	return s
}

// ---------------------------------------------------------------------------
// CreateSchedule tests
// ---------------------------------------------------------------------------

func Test_CreateSchedule_DefaultInterval_Domain(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Enabled:     true,
	}

	if err := uc.CreateSchedule(schedule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule.Interval != checkresultUC.DefaultDomainCheckInterval {
		t.Errorf("expected default domain interval %v, got %v", checkresultUC.DefaultDomainCheckInterval, schedule.Interval)
	}
}

func Test_CreateSchedule_DefaultInterval_Service(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeService,
		TargetId:    targetId,
		Enabled:     true,
	}

	if err := uc.CreateSchedule(schedule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule.Interval != checkresultUC.DefaultServiceCheckInterval {
		t.Errorf("expected default service interval %v, got %v", checkresultUC.DefaultServiceCheckInterval, schedule.Interval)
	}
}

func Test_CreateSchedule_MinimumIntervalRejected(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Interval:    4 * time.Minute,
		Enabled:     true,
	}

	if err := uc.CreateSchedule(schedule); err == nil {
		t.Fatal("expected error for interval below minimum (4 minutes)")
	}
}

func Test_CreateSchedule_ExactMinimumAccepted(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Interval:    checkresultUC.MinimumCheckInterval,
		Enabled:     true,
	}

	if err := uc.CreateSchedule(schedule); err != nil {
		t.Errorf("expected no error for exactly minimum interval, got: %v", err)
	}
}

func Test_CreateSchedule_NextRunSetWhenZero(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	interval := 30 * time.Minute

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Interval:    interval,
		Enabled:     true,
	}

	before := time.Now()
	if err := uc.CreateSchedule(schedule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if schedule.NextRun.IsZero() {
		t.Fatal("expected NextRun to be set")
	}
	// NextRun = now + rand(0, interval), so NextRun is in [before, before+interval).
	if schedule.NextRun.Before(before) || schedule.NextRun.After(before.Add(interval)) {
		t.Errorf("NextRun %v not in expected range [%v, %v+interval]", schedule.NextRun, before, before)
	}
}

func Test_CreateSchedule_NextRunPreserved(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	specificNextRun := time.Now().Add(3 * time.Hour)

	schedule := &happydns.CheckerSchedule{
		CheckerName: "checker",
		OwnerId:     ownerId,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    targetId,
		Interval:    30 * time.Minute,
		Enabled:     true,
		NextRun:     specificNextRun,
	}

	if err := uc.CreateSchedule(schedule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !schedule.NextRun.Equal(specificNextRun) {
		t.Errorf("expected NextRun to be preserved as %v, got %v", specificNextRun, schedule.NextRun)
	}
}

// ---------------------------------------------------------------------------
// UpdateSchedule tests
// ---------------------------------------------------------------------------

func Test_UpdateSchedule_PreservesLastRun(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	lastRun := time.Now().Add(-30 * time.Minute)
	existing.LastRun = &lastRun
	// Store LastRun into the db.
	if err := db.UpdateCheckerSchedule(existing); err != nil {
		t.Fatalf("failed to persist LastRun: %v", err)
	}

	// Update without setting LastRun → should be preserved from existing.
	update := *existing
	update.LastRun = nil

	if err := uc.UpdateSchedule(&update); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	if stored.LastRun == nil {
		t.Error("expected LastRun to be preserved from existing schedule")
	} else if !stored.LastRun.Equal(lastRun) {
		t.Errorf("expected LastRun %v, got %v", lastRun, *stored.LastRun)
	}
}

func Test_UpdateSchedule_RecalculatesNextRun(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	lastRun := time.Now().Add(-20 * time.Minute)
	existing.LastRun = &lastRun
	if err := db.UpdateCheckerSchedule(existing); err != nil {
		t.Fatalf("failed to persist LastRun: %v", err)
	}

	newInterval := 2 * time.Hour
	update := *existing
	update.Interval = newInterval

	if err := uc.UpdateSchedule(&update); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	expectedNextRun := lastRun.Add(newInterval)
	if !stored.NextRun.Equal(expectedNextRun) {
		t.Errorf("expected NextRun=%v (LastRun+newInterval), got %v", expectedNextRun, stored.NextRun)
	}
}

func Test_UpdateSchedule_NextRunFromNowWhenNoLastRun(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	// Ensure no LastRun in stored version.
	existing.LastRun = nil
	if err := db.UpdateCheckerSchedule(existing); err != nil {
		t.Fatalf("failed to persist cleared LastRun: %v", err)
	}

	newInterval := 2 * time.Hour
	update := *existing
	update.Interval = newInterval

	before := time.Now()
	if err := uc.UpdateSchedule(&update); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	lowerBound := before.Add(newInterval)
	upperBound := after.Add(newInterval)
	if stored.NextRun.Before(lowerBound) || stored.NextRun.After(upperBound) {
		t.Errorf("expected NextRun in [%v, %v], got %v", lowerBound, upperBound, stored.NextRun)
	}
}

func Test_UpdateSchedule_MinimumIntervalRejected(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	update := *existing
	update.Interval = 2 * time.Minute

	if err := uc.UpdateSchedule(&update); err == nil {
		t.Fatal("expected error for interval below minimum")
	}
}

// ---------------------------------------------------------------------------
// EnableSchedule / DisableSchedule tests
// ---------------------------------------------------------------------------

func Test_EnableSchedule_SetsEnabled(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	existing.Enabled = false
	existing.NextRun = time.Now().Add(time.Hour) // future, no reset needed
	if err := db.UpdateCheckerSchedule(existing); err != nil {
		t.Fatalf("failed to persist disabled schedule: %v", err)
	}

	if err := uc.EnableSchedule(existing.Id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	if !stored.Enabled {
		t.Error("expected Enabled=true after EnableSchedule")
	}
}

func Test_EnableSchedule_ResetsNextRunIfPast(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	existing.Enabled = false
	existing.NextRun = time.Now().Add(-time.Hour) // in the past
	if err := db.UpdateCheckerSchedule(existing); err != nil {
		t.Fatalf("failed to persist past NextRun: %v", err)
	}

	before := time.Now()
	if err := uc.EnableSchedule(existing.Id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	lowerBound := before.Add(existing.Interval)
	upperBound := after.Add(existing.Interval)
	if stored.NextRun.Before(lowerBound) || stored.NextRun.After(upperBound) {
		t.Errorf("expected NextRun in [%v, %v] after enable, got %v", lowerBound, upperBound, stored.NextRun)
	}
}

func Test_DisableSchedule_SetsDisabled(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	// It's already Enabled=true from seedSchedule.

	if err := uc.DisableSchedule(existing.Id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := db.GetCheckerSchedule(existing.Id)
	if err != nil {
		t.Fatalf("failed to retrieve schedule: %v", err)
	}
	if stored.Enabled {
		t.Error("expected Enabled=false after DisableSchedule")
	}
}

// ---------------------------------------------------------------------------
// ListDueSchedules tests
// ---------------------------------------------------------------------------

func Test_ListDueSchedules_FiltersDisabledAndFuture(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()

	pastTime := time.Now().Add(-time.Minute)
	futureTime := time.Now().Add(time.Hour)

	makeAndStore := func(enabled bool, nextRun time.Time) *happydns.CheckerSchedule {
		s := &happydns.CheckerSchedule{
			CheckerName: "checker",
			OwnerId:     ownerId,
			TargetType:  happydns.CheckScopeDomain,
			TargetId:    targetId,
			Interval:    time.Hour,
			Enabled:     enabled,
			NextRun:     nextRun,
		}
		if err := db.CreateCheckerSchedule(s); err != nil {
			t.Fatalf("failed to create schedule: %v", err)
		}
		return s
	}

	enabledDue := makeAndStore(true, pastTime)
	_ = makeAndStore(false, pastTime)    // disabled + past → not returned
	_ = makeAndStore(true, futureTime)   // enabled + future → not returned

	due, err := uc.ListDueSchedules()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("expected 1 due schedule, got %d", len(due))
	}
	if len(due) > 0 && !due[0].Id.Equals(enabledDue.Id) {
		t.Errorf("expected the enabled+past schedule to be returned")
	}
}

func Test_ListDueSchedules_Empty(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	due, err := uc.ListDueSchedules()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due schedules, got %d", len(due))
	}
}

// ---------------------------------------------------------------------------
// ListUpcomingSchedules tests
// ---------------------------------------------------------------------------

func Test_ListUpcomingSchedules_SortedAscending(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	now := time.Now()

	// Insert in reverse order (far future first) to ensure sorting is needed.
	for i := 3; i >= 1; i-- {
		s := &happydns.CheckerSchedule{
			CheckerName: "checker",
			OwnerId:     ownerId,
			TargetType:  happydns.CheckScopeDomain,
			TargetId:    targetId,
			Interval:    time.Hour,
			Enabled:     true,
			NextRun:     now.Add(time.Duration(i) * time.Hour),
		}
		if err := db.CreateCheckerSchedule(s); err != nil {
			t.Fatalf("failed to create schedule: %v", err)
		}
	}

	upcoming, err := uc.ListUpcomingSchedules(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(upcoming); i++ {
		if upcoming[i].NextRun.Before(upcoming[i-1].NextRun) {
			t.Errorf("schedules not in ascending order at index %d: %v > %v",
				i, upcoming[i-1].NextRun, upcoming[i].NextRun)
		}
	}
}

func Test_ListUpcomingSchedules_LimitApplied(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	now := time.Now()

	for i := range 5 {
		s := &happydns.CheckerSchedule{
			CheckerName: "checker",
			OwnerId:     ownerId,
			TargetType:  happydns.CheckScopeDomain,
			TargetId:    targetId,
			Interval:    time.Hour,
			Enabled:     true,
			NextRun:     now.Add(time.Duration(i) * time.Hour),
		}
		if err := db.CreateCheckerSchedule(s); err != nil {
			t.Fatalf("failed to create schedule: %v", err)
		}
	}

	upcoming, err := uc.ListUpcomingSchedules(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(upcoming) != 3 {
		t.Errorf("expected 3 schedules with limit=3, got %d", len(upcoming))
	}
}

// ---------------------------------------------------------------------------
// ValidateScheduleOwnership tests
// ---------------------------------------------------------------------------

func Test_ValidateScheduleOwnership_Match(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)

	if err := uc.ValidateScheduleOwnership(existing.Id, existing.OwnerId); err != nil {
		t.Errorf("expected no error for matching owner, got: %v", err)
	}
}

func Test_ValidateScheduleOwnership_Mismatch(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	existing := seedSchedule(t, db, time.Hour)
	wrongUserId, _ := happydns.NewRandomIdentifier()

	if err := uc.ValidateScheduleOwnership(existing.Id, wrongUserId); err == nil {
		t.Fatal("expected error for wrong owner")
	}
}

// ---------------------------------------------------------------------------
// RescheduleOverdueChecks tests
// ---------------------------------------------------------------------------

func createOverdueSchedules(t *testing.T, db storage.Storage, n int) []*happydns.CheckerSchedule {
	t.Helper()
	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	pastTime := time.Now().Add(-2 * time.Hour)

	schedules := make([]*happydns.CheckerSchedule, n)
	for i := range n {
		s := &happydns.CheckerSchedule{
			CheckerName: "checker",
			OwnerId:     ownerId,
			TargetType:  happydns.CheckScopeDomain,
			TargetId:    targetId,
			Interval:    time.Hour,
			Enabled:     true,
			NextRun:     pastTime,
		}
		if err := db.CreateCheckerSchedule(s); err != nil {
			t.Fatalf("failed to create overdue schedule: %v", err)
		}
		schedules[i] = s
	}
	return schedules
}

func Test_RescheduleOverdueChecks_FewOverdue_NoChange(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	schedules := createOverdueSchedules(t, db, 5)
	originalNextRuns := make([]time.Time, len(schedules))
	for i, s := range schedules {
		originalNextRuns[i] = s.NextRun
	}

	count, err := uc.RescheduleOverdueChecks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count=0 for fewer than 10 overdue schedules, got %d", count)
	}

	// NextRun should not have changed.
	for i, s := range schedules {
		stored, err := db.GetCheckerSchedule(s.Id)
		if err != nil {
			t.Fatalf("failed to retrieve schedule: %v", err)
		}
		if !stored.NextRun.Equal(originalNextRuns[i]) {
			t.Errorf("schedule[%d] NextRun changed when it should not have", i)
		}
	}
}

func Test_RescheduleOverdueChecks_ManyOverdue_Rescheduled(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	schedules := createOverdueSchedules(t, db, 15)

	before := time.Now()
	count, err := uc.RescheduleOverdueChecks()
	after := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 15 {
		t.Errorf("expected count=15, got %d", count)
	}

	// All schedules should now have NextRun in [before, before+MinimumCheckInterval].
	for i, s := range schedules {
		stored, err := db.GetCheckerSchedule(s.Id)
		if err != nil {
			t.Fatalf("failed to retrieve schedule[%d]: %v", i, err)
		}
		if !stored.NextRun.After(before) {
			t.Errorf("schedule[%d] NextRun %v should be after %v", i, stored.NextRun, before)
		}
		upperBound := after.Add(checkresultUC.MinimumCheckInterval)
		if stored.NextRun.After(upperBound) {
			t.Errorf("schedule[%d] NextRun %v should not exceed %v", i, stored.NextRun, upperBound)
		}
	}
}

func Test_RescheduleOverdueChecks_FutureSchedulesIgnored(t *testing.T) {
	db := newTestDB(t)
	uc := newTestCheckScheduleUsecase(db, nil)

	overdue := createOverdueSchedules(t, db, 15)
	_ = overdue // created in db

	// Add 3 future enabled schedules.
	ownerId, _ := happydns.NewRandomIdentifier()
	targetId, _ := happydns.NewRandomIdentifier()
	futureTime := time.Now().Add(2 * time.Hour)
	var futureSchedules []*happydns.CheckerSchedule
	for range 3 {
		s := &happydns.CheckerSchedule{
			CheckerName: "checker",
			OwnerId:     ownerId,
			TargetType:  happydns.CheckScopeDomain,
			TargetId:    targetId,
			Interval:    time.Hour,
			Enabled:     true,
			NextRun:     futureTime,
		}
		if err := db.CreateCheckerSchedule(s); err != nil {
			t.Fatalf("failed to create future schedule: %v", err)
		}
		futureSchedules = append(futureSchedules, s)
	}

	if _, err := uc.RescheduleOverdueChecks(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Future schedules should retain their original NextRun.
	for i, s := range futureSchedules {
		stored, err := db.GetCheckerSchedule(s.Id)
		if err != nil {
			t.Fatalf("failed to retrieve future schedule[%d]: %v", i, err)
		}
		if !stored.NextRun.Equal(futureTime) {
			t.Errorf("future schedule[%d] NextRun changed from %v to %v", i, futureTime, stored.NextRun)
		}
	}
}

// ---------------------------------------------------------------------------
// DiscoverAndEnsureSchedules tests
// ---------------------------------------------------------------------------

func createDomain(t *testing.T, db storage.Storage, name string) *happydns.Domain {
	t.Helper()
	ownerId, _ := happydns.NewRandomIdentifier()
	domain := &happydns.Domain{
		Owner:      ownerId,
		DomainName: name,
	}
	if err := db.CreateDomain(domain); err != nil {
		t.Fatalf("failed to create domain %s: %v", name, err)
	}
	return domain
}

func Test_DiscoverAndEnsureSchedules_CreatesForMissingPlugin(t *testing.T) {
	db := newTestDB(t)
	domain := createDomain(t, db, "example.com.")
	checkerUC := &mockCheckerUsecase{
		checkers: map[string]happydns.Checker{
			"domain-checker": &mockDomainChecker{name: "domain-checker", applyDomain: true},
		},
	}
	uc := newTestCheckScheduleUsecase(db, checkerUC)

	if err := uc.DiscoverAndEnsureSchedules(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schedules, err := db.ListCheckerSchedulesByTarget(happydns.CheckScopeDomain, domain.Id)
	if err != nil {
		t.Fatalf("failed to list schedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("expected 1 schedule created, got %d", len(schedules))
	}
}

func Test_DiscoverAndEnsureSchedules_SkipsExistingSchedule(t *testing.T) {
	db := newTestDB(t)
	domain := createDomain(t, db, "example.com.")
	checkerUC := &mockCheckerUsecase{
		checkers: map[string]happydns.Checker{
			"domain-checker": &mockDomainChecker{name: "domain-checker", applyDomain: true},
		},
	}

	// Pre-seed a schedule for this domain + checker.
	pre := &happydns.CheckerSchedule{
		CheckerName: "domain-checker",
		OwnerId:     domain.Owner,
		TargetType:  happydns.CheckScopeDomain,
		TargetId:    domain.Id,
		Interval:    24 * time.Hour,
		Enabled:     true,
		NextRun:     time.Now().Add(time.Hour),
	}
	if err := db.CreateCheckerSchedule(pre); err != nil {
		t.Fatalf("failed to seed schedule: %v", err)
	}

	uc := newTestCheckScheduleUsecase(db, checkerUC)
	if err := uc.DiscoverAndEnsureSchedules(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schedules, err := db.ListCheckerSchedulesByTarget(happydns.CheckScopeDomain, domain.Id)
	if err != nil {
		t.Fatalf("failed to list schedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Errorf("expected 1 schedule (no duplicate), got %d", len(schedules))
	}
}

func Test_DiscoverAndEnsureSchedules_SkipsServiceOnlyChecker(t *testing.T) {
	db := newTestDB(t)
	domain := createDomain(t, db, "example.com.")
	checkerUC := &mockCheckerUsecase{
		checkers: map[string]happydns.Checker{
			"service-only": &mockDomainChecker{name: "service-only", applyDomain: false, applyService: true},
		},
	}
	uc := newTestCheckScheduleUsecase(db, checkerUC)

	if err := uc.DiscoverAndEnsureSchedules(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schedules, err := db.ListCheckerSchedulesByTarget(happydns.CheckScopeDomain, domain.Id)
	if err != nil {
		t.Fatalf("failed to list schedules: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules for service-only checker, got %d", len(schedules))
	}
}

func Test_DiscoverAndEnsureSchedules_NilDependencies(t *testing.T) {
	db := newTestDB(t)
	// Both domainLister and checkerUsecase are nil → returns nil, no panic.
	uc := checkresultUC.NewCheckScheduleUsecase(db, &happydns.Options{}, nil, nil)

	if err := uc.DiscoverAndEnsureSchedules(); err != nil {
		t.Errorf("expected nil error for nil dependencies, got: %v", err)
	}
}

func Test_DiscoverAndEnsureSchedules_MultipleDomains(t *testing.T) {
	db := newTestDB(t)
	createDomain(t, db, "alpha.com.")
	createDomain(t, db, "beta.com.")
	createDomain(t, db, "gamma.com.")

	checkerUC := &mockCheckerUsecase{
		checkers: map[string]happydns.Checker{
			"checker-1": &mockDomainChecker{name: "checker-1", applyDomain: true},
			"checker-2": &mockDomainChecker{name: "checker-2", applyDomain: true},
		},
	}
	uc := newTestCheckScheduleUsecase(db, checkerUC)

	if err := uc.DiscoverAndEnsureSchedules(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 domains × 2 checkers = 6 schedules.
	enabled, err := db.ListEnabledCheckerSchedules()
	if err != nil {
		t.Fatalf("failed to list schedules: %v", err)
	}
	if len(enabled) != 6 {
		t.Errorf("expected 6 schedules (3 domains × 2 checkers), got %d", len(enabled))
	}
}
