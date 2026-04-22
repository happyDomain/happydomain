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
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"git.happydns.org/happyDomain/model"
)

// --- mock execution store for janitor tests ---

type mockExecStore struct {
	mu    sync.Mutex
	execs map[string][]*happydns.Execution // planID (base64) -> executions
	errs  map[string]error                 // planID (base64) -> error
}

func newMockExecStore() *mockExecStore {
	return &mockExecStore{
		execs: make(map[string][]*happydns.Execution),
		errs:  make(map[string]error),
	}
}

func (s *mockExecStore) addExec(planID happydns.Identifier, exec *happydns.Execution) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := planID.String()
	s.execs[key] = append(s.execs[key], exec)
}

func (s *mockExecStore) setListError(planID happydns.Identifier, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs[planID.String()] = err
}

func (s *mockExecStore) ListExecutionsByPlan(planID happydns.Identifier) ([]*happydns.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := planID.String()
	if err, ok := s.errs[key]; ok {
		return nil, err
	}
	return s.execs[key], nil
}

func (s *mockExecStore) DeleteExecution(execID happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for planKey, execs := range s.execs {
		for i, e := range execs {
			if e.Id.Equals(execID) {
				s.execs[planKey] = append(execs[:i], execs[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("execution %s not found", execID.String())
}

// Unused interface methods.
func (s *mockExecStore) ListAllExecutions() (happydns.Iterator[happydns.Execution], error) {
	return nil, nil
}
func (s *mockExecStore) ListExecutionsByChecker(string, happydns.CheckTarget, int, func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return nil, nil
}
func (s *mockExecStore) ListExecutionsByUser(happydns.Identifier, int, func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return nil, nil
}
func (s *mockExecStore) ListExecutionsByDomain(happydns.Identifier, int, func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return nil, nil
}
func (s *mockExecStore) GetExecution(happydns.Identifier) (*happydns.Execution, error) {
	return nil, nil
}
func (s *mockExecStore) CreateExecution(*happydns.Execution) error                          { return nil }
func (s *mockExecStore) UpdateExecution(*happydns.Execution) error                          { return nil }
func (s *mockExecStore) RestoreExecution(*happydns.Execution) error                         { return nil }
func (s *mockExecStore) DeleteExecutionsByChecker(string, happydns.CheckTarget) error       { return nil }
func (s *mockExecStore) TidyExecutionIndexes() error                                        { return nil }
func (s *mockExecStore) ClearExecutions() error                                             { return nil }

// --- mock user resolver ---

type mockUserResolver struct {
	users map[string]*happydns.User
}

func (r *mockUserResolver) GetUser(id happydns.Identifier) (*happydns.User, error) {
	if u, ok := r.users[id.String()]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user %s not found", id.String())
}

// --- counting wrapper ---

type countingUserResolver struct {
	inner JanitorUserResolver
	calls *int
}

func (r *countingUserResolver) GetUser(id happydns.Identifier) (*happydns.User, error) {
	*r.calls++
	return r.inner.GetUser(id)
}

// --- failing plan store ---

type failingPlanStore struct {
	mockPlanStore
	err error
}

func (s *failingPlanStore) ListAllCheckPlans() (happydns.Iterator[happydns.CheckPlan], error) {
	return nil, s.err
}

// --- helpers ---

func makePlan(id string, userID string) *happydns.CheckPlan {
	return &happydns.CheckPlan{
		Id:        happydns.Identifier(id),
		CheckerID: "ping",
		Target: happydns.CheckTarget{
			UserId:   userID,
			DomainId: "example.com",
		},
	}
}

func makeExec(id string, age time.Duration, now time.Time) *happydns.Execution {
	return &happydns.Execution{
		Id:        happydns.Identifier(id),
		CheckerID: "ping",
		Target:    happydns.CheckTarget{DomainId: "example.com"},
		StartedAt: now.Add(-age),
	}
}

// --- tests ---

func TestJanitor_RunOnce_NoPlans(t *testing.T) {
	ps := &mockPlanStore{}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)

	deleted := j.RunOnce(context.Background())
	if deleted != 0 {
		t.Fatalf("expected 0 deletions, got %d", deleted)
	}
}

func TestJanitor_RunOnce_NoExecutions(t *testing.T) {
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)

	deleted := j.RunOnce(context.Background())
	if deleted != 0 {
		t.Fatalf("expected 0 deletions, got %d", deleted)
	}
}

func TestJanitor_RunOnce_PrunesExpiredExecutions(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()

	// One recent execution (1 hour old) and one expired (100 days old with a 30-day policy).
	es.addExec(plan.Id, makeExec("recent", 1*time.Hour, now))
	es.addExec(plan.Id, makeExec("old", 100*24*time.Hour, now))

	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}

	// Verify the old execution was deleted.
	remaining, _ := es.ListExecutionsByPlan(plan.Id)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining execution, got %d", len(remaining))
	}
	if !remaining[0].Id.Equals(happydns.Identifier("recent")) {
		t.Fatalf("expected 'recent' to survive, got %s", remaining[0].Id.String())
	}
}

func TestJanitor_RunOnce_PerUserRetentionOverride(t *testing.T) {
	now := time.Now()
	userID := happydns.Identifier("user1")
	plan := makePlan("plan1", userID.String())
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()

	// Execution 20 days old. System default is 30 days (would keep), but user override is 10 days (should drop).
	es.addExec(plan.Id, makeExec("exec1", 20*24*time.Hour, now))

	resolver := &mockUserResolver{
		users: map[string]*happydns.User{
			userID.String(): {
				Id:    userID,
				Quota: happydns.UserQuota{RetentionDays: 10},
			},
		},
	}

	j := NewJanitor(ps, es, nil, nil, resolver, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion (user retention=10d), got %d", deleted)
	}
}

func TestJanitor_RunOnce_UserCacheAvoidsRepeatedLookups(t *testing.T) {
	now := time.Now()
	userID := happydns.Identifier("user1")

	// Two plans for the same user.
	plan1 := makePlan("plan1", userID.String())
	plan2 := makePlan("plan2", userID.String())
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan1, plan2}}
	es := newMockExecStore()

	es.addExec(plan1.Id, makeExec("e1", 20*24*time.Hour, now))
	es.addExec(plan2.Id, makeExec("e2", 20*24*time.Hour, now))

	calls := 0
	resolver := &countingUserResolver{
		inner: &mockUserResolver{
			users: map[string]*happydns.User{
				userID.String(): {
					Id:    userID,
					Quota: happydns.UserQuota{RetentionDays: 10},
				},
			},
		},
		calls: &calls,
	}

	j := NewJanitor(ps, es, nil, nil, resolver, DefaultRetentionPolicy(30), time.Hour)
	j.RunOnce(context.Background())

	if calls != 1 {
		t.Fatalf("expected user resolver to be called once (cached), got %d", calls)
	}
}

func TestJanitor_RunOnce_NilUserResolverUsesDefault(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "user1")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()

	// 20 days old with a 30-day default policy: should be kept.
	es.addExec(plan.Id, makeExec("exec1", 20*24*time.Hour, now))

	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 0 {
		t.Fatalf("expected 0 deletions (within default 30d retention), got %d", deleted)
	}
}

func TestJanitor_RunOnce_ListPlanError(t *testing.T) {
	ps := &failingPlanStore{err: errors.New("storage down")}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)

	deleted := j.RunOnce(context.Background())
	if deleted != 0 {
		t.Fatalf("expected 0 on plan listing error, got %d", deleted)
	}
}

func TestJanitor_RunOnce_ListExecErrorContinues(t *testing.T) {
	now := time.Now()
	plan1 := makePlan("plan1", "")
	plan2 := makePlan("plan2", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan1, plan2}}
	es := newMockExecStore()

	// plan1 returns an error; plan2 has a deletable execution.
	es.setListError(plan1.Id, errors.New("corrupt index"))
	es.addExec(plan2.Id, makeExec("old", 100*24*time.Hour, now))

	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion (plan1 error should be skipped), got %d", deleted)
	}
}

func TestJanitor_RunOnce_ContextCancellation(t *testing.T) {
	now := time.Now()
	var plans []*happydns.CheckPlan
	es := newMockExecStore()

	// Create many plans with expired executions.
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("plan%d", i)
		plan := makePlan(id, "")
		plans = append(plans, plan)
		es.addExec(plan.Id, makeExec(fmt.Sprintf("exec%d", i), 100*24*time.Hour, now))
	}
	ps := &mockPlanStore{plans: plans}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(ctx)

	// Should have stopped early - not all 100 should be deleted.
	if deleted >= 100 {
		t.Fatalf("expected early exit from cancellation, but all %d were deleted", deleted)
	}
}

func TestJanitor_StartStop(t *testing.T) {
	ps := &mockPlanStore{}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), 50*time.Millisecond)

	ctx := context.Background()
	j.Start(ctx)

	// Let it run a couple of ticks.
	time.Sleep(150 * time.Millisecond)

	j.Stop()

	// Verify it actually stopped by checking that Stop doesn't hang.
}

func TestJanitor_DoubleStartIsNoop(t *testing.T) {
	ps := &mockPlanStore{}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)

	ctx := context.Background()
	j.Start(ctx)
	j.Start(ctx) // should not panic or start a second goroutine

	j.Stop()
}

func TestJanitor_StopBeforeStartIsNoop(t *testing.T) {
	ps := &mockPlanStore{}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)

	// Should not panic or hang.
	j.Stop()
}

func TestJanitor_DefaultInterval(t *testing.T) {
	ps := &mockPlanStore{}
	es := newMockExecStore()
	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), 0)

	if j.interval != 6*time.Hour {
		t.Fatalf("expected default interval 6h, got %v", j.interval)
	}
}

func TestJanitor_RunOnce_MultiplePlansMultipleUsers(t *testing.T) {
	now := time.Now()
	user1 := happydns.Identifier("user1")
	user2 := happydns.Identifier("user2")

	plan1 := makePlan("plan1", user1.String())
	plan2 := makePlan("plan2", user2.String())
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan1, plan2}}
	es := newMockExecStore()

	// user1 has retention=10d, exec at 15 days -> should be pruned.
	es.addExec(plan1.Id, makeExec("u1_exec", 15*24*time.Hour, now))

	// user2 has retention=30d, exec at 15 days -> should be kept.
	es.addExec(plan2.Id, makeExec("u2_exec", 15*24*time.Hour, now))

	resolver := &mockUserResolver{
		users: map[string]*happydns.User{
			user1.String(): {Id: user1, Quota: happydns.UserQuota{RetentionDays: 10}},
			user2.String(): {Id: user2, Quota: happydns.UserQuota{RetentionDays: 30}},
		},
	}

	j := NewJanitor(ps, es, nil, nil, resolver, DefaultRetentionPolicy(365), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion (user1 only), got %d", deleted)
	}

	remaining1, _ := es.ListExecutionsByPlan(plan1.Id)
	if len(remaining1) != 0 {
		t.Fatalf("expected user1's exec to be deleted, got %d remaining", len(remaining1))
	}

	remaining2, _ := es.ListExecutionsByPlan(plan2.Id)
	if len(remaining2) != 1 {
		t.Fatalf("expected user2's exec to be kept, got %d remaining", len(remaining2))
	}
}

// --- mock evaluation store for janitor tests ---

type mockEvalStore struct {
	mu    sync.Mutex
	evals map[string][]*happydns.CheckEvaluation // planID (base64) -> evaluations
}

func newMockEvalStore() *mockEvalStore {
	return &mockEvalStore{
		evals: make(map[string][]*happydns.CheckEvaluation),
	}
}

func (s *mockEvalStore) addEval(planID happydns.Identifier, eval *happydns.CheckEvaluation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := planID.String()
	s.evals[key] = append(s.evals[key], eval)
}

func (s *mockEvalStore) ListEvaluationsByPlan(planID happydns.Identifier) ([]*happydns.CheckEvaluation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evals[planID.String()], nil
}

func (s *mockEvalStore) DeleteEvaluation(evalID happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for planKey, evals := range s.evals {
		for i, e := range evals {
			if e.Id.Equals(evalID) {
				s.evals[planKey] = append(evals[:i], evals[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("evaluation %s not found", evalID.String())
}

// Unused interface methods.
func (s *mockEvalStore) ListAllEvaluations() (happydns.Iterator[happydns.CheckEvaluation], error) {
	return nil, nil
}
func (s *mockEvalStore) ListEvaluationsByChecker(string, happydns.CheckTarget, int) ([]*happydns.CheckEvaluation, error) {
	return nil, nil
}
func (s *mockEvalStore) GetEvaluation(happydns.Identifier) (*happydns.CheckEvaluation, error) {
	return nil, nil
}
func (s *mockEvalStore) GetLatestEvaluation(happydns.Identifier) (*happydns.CheckEvaluation, error) {
	return nil, nil
}
func (s *mockEvalStore) CreateEvaluation(*happydns.CheckEvaluation) error                    { return nil }
func (s *mockEvalStore) RestoreEvaluation(*happydns.CheckEvaluation) error                   { return nil }
func (s *mockEvalStore) DeleteEvaluationsByChecker(string, happydns.CheckTarget) error       { return nil }
func (s *mockEvalStore) TidyEvaluationIndexes() error                                        { return nil }
func (s *mockEvalStore) ClearEvaluations() error                                             { return nil }

// --- mock snapshot store for janitor tests ---

type mockSnapStore struct {
	mu       sync.Mutex
	deleted  []string // snapshot IDs that were deleted
	failNext bool
}

func newMockSnapStore() *mockSnapStore {
	return &mockSnapStore{}
}

func (s *mockSnapStore) DeleteSnapshot(snapID happydns.Identifier) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext {
		s.failNext = false
		return fmt.Errorf("snapshot %s delete failed", snapID.String())
	}
	s.deleted = append(s.deleted, snapID.String())
	return nil
}

func (s *mockSnapStore) deletedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.deleted)
}

// Unused interface methods.
func (s *mockSnapStore) ListAllSnapshots() (happydns.Iterator[happydns.ObservationSnapshot], error) {
	return nil, nil
}
func (s *mockSnapStore) GetSnapshot(happydns.Identifier) (*happydns.ObservationSnapshot, error) {
	return nil, nil
}
func (s *mockSnapStore) CreateSnapshot(*happydns.ObservationSnapshot) error { return nil }
func (s *mockSnapStore) ClearSnapshots() error                              { return nil }

// --- helpers ---

func makeEval(id string, snapID string, age time.Duration, now time.Time, planID happydns.Identifier) *happydns.CheckEvaluation {
	pid := planID
	return &happydns.CheckEvaluation{
		Id:          happydns.Identifier(id),
		PlanID:      &pid,
		CheckerID:   "ping",
		Target:      happydns.CheckTarget{DomainId: "example.com"},
		SnapshotID:  happydns.Identifier(snapID),
		EvaluatedAt: now.Add(-age),
	}
}

// --- evaluation pruning tests ---

func TestJanitor_RunOnce_PrunesExpiredEvaluations(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()
	evs := newMockEvalStore()
	ss := newMockSnapStore()

	evs.addEval(plan.Id, makeEval("recent_eval", "snap1", 1*time.Hour, now, plan.Id))
	evs.addEval(plan.Id, makeEval("old_eval", "snap2", 100*24*time.Hour, now, plan.Id))

	j := NewJanitor(ps, es, evs, ss, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}

	remaining, _ := evs.ListEvaluationsByPlan(plan.Id)
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining evaluation, got %d", len(remaining))
	}
	if !remaining[0].Id.Equals(happydns.Identifier("recent_eval")) {
		t.Fatalf("expected 'recent_eval' to survive, got %s", remaining[0].Id.String())
	}

	if ss.deletedCount() != 1 {
		t.Fatalf("expected 1 snapshot deleted, got %d", ss.deletedCount())
	}
}

func TestJanitor_RunOnce_PrunesBothExecutionsAndEvaluations(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()
	evs := newMockEvalStore()
	ss := newMockSnapStore()

	es.addExec(plan.Id, makeExec("old_exec", 100*24*time.Hour, now))
	evs.addEval(plan.Id, makeEval("old_eval", "snap1", 100*24*time.Hour, now, plan.Id))

	j := NewJanitor(ps, es, evs, ss, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 2 {
		t.Fatalf("expected 2 deletions (1 exec + 1 eval), got %d", deleted)
	}
}

func TestJanitor_RunOnce_EvalPruningRespectsPerUserRetention(t *testing.T) {
	now := time.Now()
	userID := happydns.Identifier("user1")
	plan := makePlan("plan1", userID.String())
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()
	evs := newMockEvalStore()
	ss := newMockSnapStore()

	// Evaluation 20 days old. System default is 30 days (would keep), but user override is 10 days (should drop).
	evs.addEval(plan.Id, makeEval("eval1", "snap1", 20*24*time.Hour, now, plan.Id))

	resolver := &mockUserResolver{
		users: map[string]*happydns.User{
			userID.String(): {
				Id:    userID,
				Quota: happydns.UserQuota{RetentionDays: 10},
			},
		},
	}

	j := NewJanitor(ps, es, evs, ss, resolver, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	if deleted != 1 {
		t.Fatalf("expected 1 deletion (user retention=10d), got %d", deleted)
	}
}

func TestJanitor_RunOnce_NilEvalStoreSkipsEvalPruning(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()

	es.addExec(plan.Id, makeExec("old", 100*24*time.Hour, now))

	j := NewJanitor(ps, es, nil, nil, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	// Should only delete the execution, not panic on nil evalStore.
	if deleted != 1 {
		t.Fatalf("expected 1 deletion, got %d", deleted)
	}
}

func TestJanitor_RunOnce_SnapshotDeleteFailureContinues(t *testing.T) {
	now := time.Now()
	plan := makePlan("plan1", "")
	ps := &mockPlanStore{plans: []*happydns.CheckPlan{plan}}
	es := newMockExecStore()
	evs := newMockEvalStore()
	ss := newMockSnapStore()
	ss.failNext = true

	evs.addEval(plan.Id, makeEval("old_eval", "snap1", 100*24*time.Hour, now, plan.Id))

	j := NewJanitor(ps, es, evs, ss, nil, DefaultRetentionPolicy(30), time.Hour)
	deleted := j.RunOnce(context.Background())

	// Evaluation should still be deleted even if snapshot deletion fails.
	if deleted != 1 {
		t.Fatalf("expected 1 deletion despite snapshot failure, got %d", deleted)
	}
}
