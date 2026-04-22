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

package checker

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// --- mock engine ---

type mockEngine struct {
	mu          sync.Mutex
	executions  []*happydns.Execution
	createErr   error
	runErr      error
	runDuration time.Duration
}

func (e *mockEngine) CreateExecution(checkerID string, target happydns.CheckTarget, plan *happydns.CheckPlan) (*happydns.Execution, error) {
	if e.createErr != nil {
		return nil, e.createErr
	}
	id, _ := happydns.NewRandomIdentifier()
	exec := &happydns.Execution{
		Id:        id,
		CheckerID: checkerID,
		Target:    target,
		StartedAt: time.Now(),
		Status:    happydns.ExecutionPending,
	}
	e.mu.Lock()
	e.executions = append(e.executions, exec)
	e.mu.Unlock()
	return exec, nil
}

func (e *mockEngine) RunExecution(ctx context.Context, exec *happydns.Execution, plan *happydns.CheckPlan, runOpts happydns.CheckerOptions) (*happydns.CheckEvaluation, error) {
	if e.runDuration > 0 {
		select {
		case <-time.After(e.runDuration):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if e.runErr != nil {
		return nil, e.runErr
	}
	id, _ := happydns.NewRandomIdentifier()
	return &happydns.CheckEvaluation{Id: id}, nil
}

func (e *mockEngine) executionCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.executions)
}

// --- mock plan store ---

type mockPlanStore struct {
	plans []*happydns.CheckPlan
}

func (s *mockPlanStore) ListAllCheckPlans() (happydns.Iterator[happydns.CheckPlan], error) {
	return &sliceIterator[happydns.CheckPlan]{items: s.plans}, nil
}

func (s *mockPlanStore) ListCheckPlansByTarget(target happydns.CheckTarget) ([]*happydns.CheckPlan, error) {
	var result []*happydns.CheckPlan
	for _, p := range s.plans {
		if p.Target.String() == target.String() {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *mockPlanStore) ListCheckPlansByChecker(string) ([]*happydns.CheckPlan, error) {
	return nil, nil
}
func (s *mockPlanStore) ListCheckPlansByUser(happydns.Identifier) ([]*happydns.CheckPlan, error) {
	return nil, nil
}
func (s *mockPlanStore) GetCheckPlan(id happydns.Identifier) (*happydns.CheckPlan, error) {
	for _, p := range s.plans {
		if p.Id.Equals(id) {
			return p, nil
		}
	}
	return nil, happydns.ErrCheckPlanNotFound
}
func (s *mockPlanStore) CreateCheckPlan(plan *happydns.CheckPlan) error {
	id, _ := happydns.NewRandomIdentifier()
	plan.Id = id
	s.plans = append(s.plans, plan)
	return nil
}
func (s *mockPlanStore) UpdateCheckPlan(plan *happydns.CheckPlan) error  { return nil }
func (s *mockPlanStore) RestoreCheckPlan(plan *happydns.CheckPlan) error { return nil }
func (s *mockPlanStore) DeleteCheckPlan(happydns.Identifier) error       { return nil }
func (s *mockPlanStore) TidyCheckPlanIndexes() error                    { return nil }
func (s *mockPlanStore) ClearCheckPlans() error                         { return nil }

// --- mock domain lister ---

type mockDomainLister struct {
	domains []*happydns.Domain
}

func (d *mockDomainLister) ListAllDomains() (happydns.Iterator[happydns.Domain], error) {
	return &sliceIterator[happydns.Domain]{items: d.domains}, nil
}

// --- mock zone getter ---

type mockZoneGetter struct {
	zones map[string]*happydns.ZoneMessage
}

func (z *mockZoneGetter) GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error) {
	zm, ok := z.zones[id.String()]
	if !ok {
		return nil, happydns.ErrZoneNotFound
	}
	return zm, nil
}

// --- mock state store ---

type mockStateStore struct {
	mu      sync.Mutex
	lastRun time.Time
}

func (s *mockStateStore) GetLastSchedulerRun() (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRun, nil
}

func (s *mockStateStore) SetLastSchedulerRun(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRun = t
	return nil
}

// --- sliceIterator ---

type sliceIterator[T any] struct {
	items []*T
	idx   int
	cur   *T
}

func (it *sliceIterator[T]) Next() bool {
	if it.idx >= len(it.items) {
		return false
	}
	it.cur = it.items[it.idx]
	it.idx++
	return true
}
func (it *sliceIterator[T]) NextWithError() bool { return it.Next() }
func (it *sliceIterator[T]) Item() *T            { return it.cur }
func (it *sliceIterator[T]) DropItem() error      { return nil }
func (it *sliceIterator[T]) Key() string           { return "" }
func (it *sliceIterator[T]) Raw() any              { return nil }
func (it *sliceIterator[T]) Err() error            { return nil }
func (it *sliceIterator[T]) Close()                {}

// --- helper to build a scheduler with mock deps ---

func newTestScheduler(engine happydns.CheckerEngine, domains []*happydns.Domain) (*Scheduler, *mockPlanStore, *mockStateStore) {
	ps := &mockPlanStore{}
	dl := &mockDomainLister{domains: domains}
	zg := &mockZoneGetter{zones: make(map[string]*happydns.ZoneMessage)}
	ss := &mockStateStore{}
	sched := NewScheduler(engine, 2, ps, dl, zg, ss, nil, nil)
	return sched, ps, ss
}

// --- computeNextRun tests (preserved from original) ---

func TestComputeNextRun_ZeroLastActive(t *testing.T) {
	interval := 1 * time.Hour
	offset := 10 * time.Minute

	nextRun := computeNextRun(interval, offset, time.Time{})
	now := time.Now()

	if !nextRun.After(now) {
		t.Errorf("expected nextRun (%v) to be in the future (now=%v)", nextRun, now)
	}
	if nextRun.After(now.Add(interval)) {
		t.Errorf("expected nextRun (%v) to be within one interval from now (%v)", nextRun, now.Add(interval))
	}
}

func TestComputeNextRun_RecentLastActive_NoRerun(t *testing.T) {
	interval := 1 * time.Hour
	offset := computeOffset("test-checker", "test-target", interval)
	now := time.Now()

	// lastActive is very recent; the current slot was already executed.
	lastActive := now.Add(-1 * time.Minute)

	nextRun := computeNextRun(interval, offset, lastActive)

	if !nextRun.After(now) {
		t.Errorf("expected nextRun (%v) to be in the future when lastActive is recent (now=%v)", nextRun, now)
	}
}

func TestComputeNextRun_OldLastActive_CatchUp(t *testing.T) {
	interval := 1 * time.Hour
	offset := 0 * time.Minute
	now := time.Now()

	// lastActive is several hours ago; there should be a missed slot.
	lastActive := now.Add(-3 * time.Hour)

	nextRun := computeNextRun(interval, offset, lastActive)

	// The missed slot should be scheduled at now (catch-up).
	if nextRun.After(now.Add(1 * time.Second)) {
		t.Errorf("expected nextRun (%v) to be approximately now (%v) for catch-up", nextRun, now)
	}
	if nextRun.Before(now.Add(-1 * time.Second)) {
		t.Errorf("expected nextRun (%v) to be approximately now (%v) for catch-up", nextRun, now)
	}
}

// --- Scheduler lifecycle tests ---

func TestScheduler_StartStop(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)

	status := sched.GetStatus()
	if !status.Running {
		t.Error("expected scheduler to be running after Start")
	}

	sched.Stop()

	status = sched.GetStatus()
	if status.Running {
		t.Error("expected scheduler to be stopped after Stop")
	}
}

func TestScheduler_StopIdempotent(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	// Stop without Start should not panic.
	sched.Stop()
	sched.Stop()
}

func TestScheduler_SetEnabled(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	ctx := context.Background()

	// Start via SetEnabled.
	sched.SetEnabled(ctx, true)
	status := sched.GetStatus()
	if !status.Running {
		t.Error("expected scheduler to be running after SetEnabled(true)")
	}

	// Stop via SetEnabled.
	sched.SetEnabled(ctx, false)
	status = sched.GetStatus()
	if status.Running {
		t.Error("expected scheduler to be stopped after SetEnabled(false)")
	}

	// Restart via SetEnabled (this verifies the fixed context bug).
	sched.SetEnabled(ctx, true)
	status = sched.GetStatus()
	if !status.Running {
		t.Fatal("expected scheduler to be running after re-enable via SetEnabled(true)")
	}

	// Give it a moment and verify it's still running (not exited due to cancelled context).
	time.Sleep(50 * time.Millisecond)
	status = sched.GetStatus()
	if !status.Running {
		t.Error("scheduler exited prematurely after re-enable; likely using a cancelled context")
	}

	sched.Stop()
}

func TestScheduler_Gate(t *testing.T) {
	engine := &mockEngine{}
	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()

	domain := &happydns.Domain{
		Id:         did,
		Owner:      uid,
		DomainName: "gate-test.example.",
	}

	var gated atomic.Int32
	ps := &mockPlanStore{}
	dl := &mockDomainLister{domains: []*happydns.Domain{domain}}
	zg := &mockZoneGetter{zones: make(map[string]*happydns.ZoneMessage)}
	ss := &mockStateStore{}
	sched := NewScheduler(engine, 2, ps, dl, zg, ss, func(target happydns.CheckTarget, interval time.Duration) bool {
		gated.Add(1)
		return false // block all jobs
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sched.Start(ctx)
	defer sched.Stop()

	// Wait briefly for the scheduler to attempt to run jobs.
	time.Sleep(200 * time.Millisecond)

	// The gate should have been called but no executions should have run.
	if engine.executionCount() > 0 {
		t.Error("expected no executions when gate blocks all jobs")
	}
}

// injectJob pushes a SchedulerJob directly into a running scheduler's queue
// and wakes the loop so the new job is observed promptly. It must be called
// after Start (Start resets the queue via buildQueue).
func injectJob(t *testing.T, sched *Scheduler, job *SchedulerJob) {
	t.Helper()
	sched.mu.Lock()
	heap.Push(&sched.queue, job)
	sched.mu.Unlock()
	select {
	case sched.wake <- struct{}{}:
	default:
	}
}

func TestScheduler_OnExecute_CalledOnSuccess(t *testing.T) {
	engine := &mockEngine{}
	ps := &mockPlanStore{}
	dl := &mockDomainLister{domains: nil}
	zg := &mockZoneGetter{zones: make(map[string]*happydns.ZoneMessage)}
	ss := &mockStateStore{}

	var onExecCalls atomic.Int32
	var lastTarget atomic.Value // happydns.CheckTarget
	sched := NewScheduler(engine, 2, ps, dl, zg, ss, nil, func(target happydns.CheckTarget) {
		onExecCalls.Add(1)
		lastTarget.Store(target)
	})

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	sched.Start(t.Context())
	defer sched.Stop()

	// Inject a due job with a long interval so reschedule does not re-run
	// it within the test window.
	injectJob(t, sched, &SchedulerJob{
		CheckerID: "test-checker",
		Target:    target,
		Interval:  time.Hour,
		NextRun:   time.Now(),
	})

	// Wait for the scheduler to pick up and execute the job.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && onExecCalls.Load() == 0 {
		time.Sleep(20 * time.Millisecond)
	}

	if got := onExecCalls.Load(); got < 1 {
		t.Fatalf("expected onExecute to be called at least once, got %d", got)
	}
	if got := engine.executionCount(); got < 1 {
		t.Errorf("expected at least one execution created, got %d", got)
	}
	stored, _ := lastTarget.Load().(happydns.CheckTarget)
	if stored.UserId != target.UserId || stored.DomainId != target.DomainId {
		t.Errorf("expected onExecute target=%+v, got %+v", target, stored)
	}
}

func TestScheduler_OnExecute_NotCalledWhenCreateFails(t *testing.T) {
	engine := &mockEngine{createErr: happydns.ErrExecutionNotFound}
	ps := &mockPlanStore{}
	dl := &mockDomainLister{domains: nil}
	zg := &mockZoneGetter{zones: make(map[string]*happydns.ZoneMessage)}
	ss := &mockStateStore{}

	var onExecCalls atomic.Int32
	sched := NewScheduler(engine, 2, ps, dl, zg, ss, nil, func(target happydns.CheckTarget) {
		onExecCalls.Add(1)
	})

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	sched.Start(t.Context())
	defer sched.Stop()

	// Inject a due job that will fail at CreateExecution. Use a long
	// interval so we do not repeatedly attempt within the test window.
	injectJob(t, sched, &SchedulerJob{
		CheckerID: "test-checker",
		Target:    target,
		Interval:  time.Hour,
		NextRun:   time.Now(),
	})

	// Wait long enough for the scheduler to attempt the job at least once.
	time.Sleep(250 * time.Millisecond)

	if got := onExecCalls.Load(); got != 0 {
		t.Errorf("expected onExecute not to be called when CreateExecution fails, got %d", got)
	}
	if got := engine.executionCount(); got != 0 {
		t.Errorf("expected no executions created when CreateExecution fails, got %d", got)
	}
}

func TestScheduler_OnExecute_NotCalledWhenGateDenies(t *testing.T) {
	// onExecute should also be skipped when the gate blocks a job — the
	// usage counter must only move for jobs that actually produced an
	// execution.
	engine := &mockEngine{}
	ps := &mockPlanStore{}
	dl := &mockDomainLister{domains: nil}
	zg := &mockZoneGetter{zones: make(map[string]*happydns.ZoneMessage)}
	ss := &mockStateStore{}

	var onExecCalls atomic.Int32
	sched := NewScheduler(engine, 2, ps, dl, zg, ss,
		func(target happydns.CheckTarget, interval time.Duration) bool { return false },
		func(target happydns.CheckTarget) { onExecCalls.Add(1) },
	)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	sched.Start(t.Context())
	defer sched.Stop()

	injectJob(t, sched, &SchedulerJob{
		CheckerID: "test-checker",
		Target:    target,
		Interval:  time.Hour,
		NextRun:   time.Now(),
	})

	time.Sleep(200 * time.Millisecond)

	if got := onExecCalls.Load(); got != 0 {
		t.Errorf("expected onExecute not to be called when gate denies, got %d", got)
	}
	if got := engine.executionCount(); got != 0 {
		t.Errorf("expected no executions when gate denies, got %d", got)
	}
}

func TestScheduler_GetStatus_Empty(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	status := sched.GetStatus()
	if status.Running {
		t.Error("expected not running before Start")
	}
	if status.JobCount != 0 {
		t.Errorf("expected 0 jobs, got %d", status.JobCount)
	}
	if len(status.NextJobs) != 0 {
		t.Errorf("expected 0 next jobs, got %d", len(status.NextJobs))
	}
}

func TestScheduler_RebuildQueue(t *testing.T) {
	engine := &mockEngine{}
	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()

	domain := &happydns.Domain{
		Id:         did,
		Owner:      uid,
		DomainName: "rebuild.example.",
	}

	sched, _, _ := newTestScheduler(engine, []*happydns.Domain{domain})

	count := sched.RebuildQueue()
	if count == 0 {
		// No checkers registered, so 0 is expected.
		// This test verifies RebuildQueue doesn't panic.
	}

	status := sched.GetStatus()
	if status.JobCount != count {
		t.Errorf("expected JobCount %d, got %d", count, status.JobCount)
	}
}

func TestScheduler_NotifyDomainRemoved(t *testing.T) {
	engine := &mockEngine{}
	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()

	domain := &happydns.Domain{
		Id:         did,
		Owner:      uid,
		DomainName: "remove-test.example.",
	}

	sched, _, _ := newTestScheduler(engine, []*happydns.Domain{domain})

	// Build the queue so jobs exist.
	sched.mu.Lock()
	sched.buildQueue()
	initialCount := sched.queue.Len()
	sched.mu.Unlock()

	// Remove the domain.
	sched.NotifyDomainRemoved(did)

	sched.mu.RLock()
	afterCount := sched.queue.Len()
	sched.mu.RUnlock()

	if initialCount > 0 && afterCount >= initialCount {
		t.Errorf("expected jobs to decrease after domain removal, was %d, now %d", initialCount, afterCount)
	}

	// Verify no jobs reference the removed domain.
	sched.mu.RLock()
	for _, job := range sched.queue {
		if job.Target.DomainId == did.String() {
			t.Errorf("found job referencing removed domain %s", did)
		}
	}
	sched.mu.RUnlock()
}

func TestScheduler_GetPlannedJobsForChecker(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	// Manually push a job into the queue.
	sched.mu.Lock()
	job := &SchedulerJob{
		CheckerID: "test-checker",
		Target:    target,
		Interval:  time.Hour,
		NextRun:   time.Now().Add(time.Hour),
	}
	heap.Push(&sched.queue, job)
	sched.mu.Unlock()

	jobs := sched.GetPlannedJobsForChecker("test-checker", target)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 planned job, got %d", len(jobs))
	}
	if jobs[0].CheckerID != "test-checker" {
		t.Errorf("expected checker ID test-checker, got %s", jobs[0].CheckerID)
	}

	// Different checker should return empty.
	jobs2 := sched.GetPlannedJobsForChecker("other-checker", target)
	if len(jobs2) != 0 {
		t.Errorf("expected 0 planned jobs for other checker, got %d", len(jobs2))
	}
}

// --- Queue tests ---

func TestSchedulerQueue_HeapOrder(t *testing.T) {
	q := &SchedulerQueue{}
	heap.Init(q)

	now := time.Now()
	heap.Push(q, &SchedulerJob{CheckerID: "c", NextRun: now.Add(3 * time.Hour)})
	heap.Push(q, &SchedulerJob{CheckerID: "a", NextRun: now.Add(1 * time.Hour)})
	heap.Push(q, &SchedulerJob{CheckerID: "b", NextRun: now.Add(2 * time.Hour)})

	first := heap.Pop(q).(*SchedulerJob)
	if first.CheckerID != "a" {
		t.Errorf("expected first popped job to be 'a', got %s", first.CheckerID)
	}
	second := heap.Pop(q).(*SchedulerJob)
	if second.CheckerID != "b" {
		t.Errorf("expected second popped job to be 'b', got %s", second.CheckerID)
	}
	third := heap.Pop(q).(*SchedulerJob)
	if third.CheckerID != "c" {
		t.Errorf("expected third popped job to be 'c', got %s", third.CheckerID)
	}
}

func TestSchedulerQueue_Peek(t *testing.T) {
	q := &SchedulerQueue{}
	heap.Init(q)

	if q.Peek() != nil {
		t.Error("expected Peek on empty queue to return nil")
	}

	now := time.Now()
	heap.Push(q, &SchedulerJob{CheckerID: "x", NextRun: now.Add(time.Hour)})
	heap.Push(q, &SchedulerJob{CheckerID: "y", NextRun: now.Add(time.Minute)})

	peeked := q.Peek()
	if peeked.CheckerID != "y" {
		t.Errorf("expected Peek to return earliest job 'y', got %s", peeked.CheckerID)
	}
	// Peek should not remove the item.
	if q.Len() != 2 {
		t.Errorf("expected queue length 2 after Peek, got %d", q.Len())
	}
}

// --- spreadOverdueJobs tests ---

func TestSpreadOverdueJobs(t *testing.T) {
	engine := &mockEngine{}
	sched, _, _ := newTestScheduler(engine, nil)

	now := time.Now()

	// Add overdue jobs.
	sched.mu.Lock()
	for i := 0; i < 5; i++ {
		heap.Push(&sched.queue, &SchedulerJob{
			CheckerID: "overdue",
			Target:    happydns.CheckTarget{UserId: "u", DomainId: "d"},
			Interval:  time.Hour,
			NextRun:   now.Add(-time.Duration(i+1) * time.Hour),
		})
	}
	sched.spreadOverdueJobs()
	sched.mu.Unlock()

	// All jobs should now be in the future (or at now).
	sched.mu.RLock()
	for _, job := range sched.queue {
		if job.NextRun.Before(now.Add(-time.Second)) {
			t.Errorf("expected job to be rescheduled to now or later, got %v", job.NextRun)
		}
	}
	sched.mu.RUnlock()
}

// --- effectiveInterval tests ---

func TestEffectiveInterval_Defaults(t *testing.T) {
	sched, _, _ := newTestScheduler(&mockEngine{}, nil)

	// No interval spec, no plan -> defaultInterval.
	def := &happydns.CheckerDefinition{}
	got := sched.effectiveInterval(def, nil)
	if got != defaultInterval {
		t.Errorf("expected %v, got %v", defaultInterval, got)
	}
}

func TestEffectiveInterval_DefDefault(t *testing.T) {
	sched, _, _ := newTestScheduler(&mockEngine{}, nil)

	def := &happydns.CheckerDefinition{
		Interval: &happydns.CheckIntervalSpec{
			Default: 2 * time.Hour,
			Min:     1 * time.Hour,
			Max:     12 * time.Hour,
		},
	}
	got := sched.effectiveInterval(def, nil)
	if got != 2*time.Hour {
		t.Errorf("expected 2h, got %v", got)
	}
}

func TestEffectiveInterval_PlanOverride(t *testing.T) {
	sched, _, _ := newTestScheduler(&mockEngine{}, nil)

	def := &happydns.CheckerDefinition{
		Interval: &happydns.CheckIntervalSpec{
			Default: 2 * time.Hour,
			Min:     1 * time.Hour,
			Max:     12 * time.Hour,
		},
	}
	interval := 6 * time.Hour
	plan := &happydns.CheckPlan{Interval: &interval}
	got := sched.effectiveInterval(def, plan)
	if got != 6*time.Hour {
		t.Errorf("expected 6h, got %v", got)
	}
}

func TestEffectiveInterval_ClampMin(t *testing.T) {
	sched, _, _ := newTestScheduler(&mockEngine{}, nil)

	def := &happydns.CheckerDefinition{
		Interval: &happydns.CheckIntervalSpec{
			Default: 2 * time.Hour,
			Min:     1 * time.Hour,
			Max:     12 * time.Hour,
		},
	}
	interval := 10 * time.Minute // below min
	plan := &happydns.CheckPlan{Interval: &interval}
	got := sched.effectiveInterval(def, plan)
	if got != 1*time.Hour {
		t.Errorf("expected clamped to 1h, got %v", got)
	}
}

func TestEffectiveInterval_ClampMax(t *testing.T) {
	sched, _, _ := newTestScheduler(&mockEngine{}, nil)

	def := &happydns.CheckerDefinition{
		Interval: &happydns.CheckIntervalSpec{
			Default: 2 * time.Hour,
			Min:     1 * time.Hour,
			Max:     12 * time.Hour,
		},
	}
	interval := 24 * time.Hour // above max
	plan := &happydns.CheckPlan{Interval: &interval}
	got := sched.effectiveInterval(def, plan)
	if got != 12*time.Hour {
		t.Errorf("expected clamped to 12h, got %v", got)
	}
}

// --- buildPlanIndex tests ---

func TestBuildPlanIndex(t *testing.T) {
	target := happydns.CheckTarget{UserId: "u1", DomainId: "d1"}
	plans := []*happydns.CheckPlan{
		{
			CheckerID: "c1",
			Target:    target,
			Enabled:   map[string]bool{"r1": false, "r2": false},
		},
		{
			CheckerID: "c2",
			Target:    target,
			Enabled:   map[string]bool{"r1": true},
		},
	}

	disabled, planMap := buildPlanIndex(plans)

	key1 := "c1|" + target.String()
	key2 := "c2|" + target.String()

	if !disabled[key1] {
		t.Error("expected c1 to be in disabled set")
	}
	if disabled[key2] {
		t.Error("expected c2 to NOT be in disabled set")
	}
	if planMap[key1] != plans[0] {
		t.Error("expected planMap to contain c1 plan")
	}
	if planMap[key2] != plans[1] {
		t.Error("expected planMap to contain c2 plan")
	}
}

// --- computeJitter tests ---

func TestComputeJitter_Deterministic(t *testing.T) {
	now := time.Now()
	interval := time.Hour

	j1 := computeJitter("c1", "t1", now, interval)
	j2 := computeJitter("c1", "t1", now, interval)

	if j1 != j2 {
		t.Errorf("expected deterministic jitter, got %v and %v", j1, j2)
	}

	// Different inputs should (usually) produce different jitter.
	j3 := computeJitter("c2", "t1", now, interval)
	// Not guaranteed to differ, but very likely.
	_ = j3
}

func TestComputeJitter_BoundedByInterval(t *testing.T) {
	now := time.Now()
	interval := time.Hour
	maxJitter := interval / 20

	j := computeJitter("c1", "t1", now, interval)
	if j < 0 || j >= maxJitter {
		t.Errorf("expected jitter in [0, %v), got %v", maxJitter, j)
	}
}

func TestComputeJitter_ZeroInterval(t *testing.T) {
	j := computeJitter("c1", "t1", time.Now(), 0)
	if j != 0 {
		t.Errorf("expected 0 jitter for zero interval, got %v", j)
	}
}

// --- computeOffset tests ---

func TestComputeOffset_Deterministic(t *testing.T) {
	interval := time.Hour
	o1 := computeOffset("c1", "t1", interval)
	o2 := computeOffset("c1", "t1", interval)
	if o1 != o2 {
		t.Errorf("expected deterministic offset, got %v and %v", o1, o2)
	}
}

func TestComputeOffset_WithinInterval(t *testing.T) {
	interval := time.Hour
	o := computeOffset("c1", "t1", interval)
	if o < 0 || o >= interval {
		t.Errorf("expected offset in [0, %v), got %v", interval, o)
	}
}
