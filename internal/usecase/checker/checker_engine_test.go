// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// testObservationProvider returns static test data.
type testObservationProvider struct{}

func (p *testObservationProvider) Key() happydns.ObservationKey {
	return "test_obs"
}

func (p *testObservationProvider) Collect(ctx context.Context, opts happydns.CheckerOptions) (any, error) {
	return map[string]any{"value": 42}, nil
}

// testCheckRule produces a state based on observations.
type testCheckRule struct {
	name   string
	status happydns.Status
}

func (r *testCheckRule) Name() string        { return r.name }
func (r *testCheckRule) Description() string { return "test rule: " + r.name }

func (r *testCheckRule) Evaluate(ctx context.Context, obs happydns.ObservationGetter, opts happydns.CheckerOptions) happydns.CheckState {
	var data map[string]any
	if err := obs.Get(ctx, "test_obs", &data); err != nil {
		return happydns.CheckState{Status: happydns.StatusError, Message: err.Error()}
	}
	return happydns.CheckState{Status: r.status, Message: r.name + " passed", Code: r.name}
}

func TestCheckerEngine_RunOK(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	// Register test provider and checker.
	checker.RegisterObservationProvider(&testObservationProvider{})
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker",
		Name: "Test Checker",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_ok", status: happydns.StatusOK},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	exec, err := engine.CreateExecution("test_checker", target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	eval, err := engine.RunExecution(context.Background(), exec, nil, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	if eval == nil {
		t.Fatal("RunExecution() returned nil evaluation")
	}

	if exec.Result.Status != happydns.StatusOK {
		t.Errorf("expected status OK, got %s", exec.Result.Status)
	}

	if len(eval.States) != 1 {
		t.Errorf("expected 1 state, got %d", len(eval.States))
	}

	// Verify execution was persisted.
	execs, err := store.ListExecutionsByChecker("test_checker", target, 0)
	if err != nil {
		t.Fatalf("ListExecutionsByChecker() returned error: %v", err)
	}
	if len(execs) != 1 {
		t.Errorf("expected 1 execution, got %d", len(execs))
	}

	// Verify the execution ended as Done.
	for _, ex := range execs {
		if ex.Status != happydns.ExecutionDone {
			t.Errorf("expected execution status Done, got %d", ex.Status)
		}
	}
}

func TestCheckerEngine_RunWarn(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker_warn",
		Name: "Test Checker Warn",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_ok", status: happydns.StatusOK},
			&testCheckRule{name: "rule_warn", status: happydns.StatusWarn},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	exec, err := engine.CreateExecution("test_checker_warn", target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}
	eval, err := engine.RunExecution(context.Background(), exec, nil, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	// Worst status aggregation: WARN should win over OK.
	if exec.Result.Status != happydns.StatusWarn {
		t.Errorf("expected aggregated status WARN, got %s", exec.Result.Status)
	}

	if len(eval.States) != 2 {
		t.Errorf("expected 2 states, got %d", len(eval.States))
	}
}

func TestCheckerEngine_RunPerRuleDisable(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker_per_rule",
		Name: "Test Checker Per Rule",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_a", status: happydns.StatusOK},
			&testCheckRule{name: "rule_b", status: happydns.StatusWarn},
			&testCheckRule{name: "rule_c", status: happydns.StatusCrit},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	// Disable rule_b and rule_c, only rule_a should run.
	plan := &happydns.CheckPlan{
		CheckerID: "test_checker_per_rule",
		Target:    target,
		Enabled: map[string]bool{
			"rule_a": true,
			"rule_b": false,
			"rule_c": false,
		},
	}

	exec, err := engine.CreateExecution("test_checker_per_rule", target, plan)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}
	eval, err := engine.RunExecution(context.Background(), exec, plan, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	if len(eval.States) != 1 {
		t.Fatalf("expected 1 state (only rule_a), got %d", len(eval.States))
	}

	if exec.Result.Status != happydns.StatusOK {
		t.Errorf("expected status OK (only rule_a active), got %s", exec.Result.Status)
	}

	if eval.States[0].Code != "rule_a" {
		t.Errorf("expected rule_a state, got code %s", eval.States[0].Code)
	}
}

func TestCheckPlan_IsFullyDisabled(t *testing.T) {
	// Nil map = not disabled.
	p := &happydns.CheckPlan{}
	if p.IsFullyDisabled() {
		t.Error("nil map should not be fully disabled")
	}

	// All false = disabled.
	p.Enabled = map[string]bool{"a": false, "b": false}
	if !p.IsFullyDisabled() {
		t.Error("all-false map should be fully disabled")
	}

	// Mixed = not disabled.
	p.Enabled = map[string]bool{"a": true, "b": false}
	if p.IsFullyDisabled() {
		t.Error("mixed map should not be fully disabled")
	}
}

func TestCheckPlan_IsRuleEnabled(t *testing.T) {
	// Nil map = all enabled.
	p := &happydns.CheckPlan{}
	if !p.IsRuleEnabled("any") {
		t.Error("nil map should enable all rules")
	}

	// Missing key = enabled.
	p.Enabled = map[string]bool{"a": false}
	if !p.IsRuleEnabled("b") {
		t.Error("missing key should be enabled")
	}

	// Explicit false = disabled.
	if p.IsRuleEnabled("a") {
		t.Error("explicit false should be disabled")
	}

	// Explicit true = enabled.
	p.Enabled["c"] = true
	if !p.IsRuleEnabled("c") {
		t.Error("explicit true should be enabled")
	}
}

func TestCheckerEngine_RunNotFound(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}
	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String()}

	_, err = engine.CreateExecution("nonexistent_checker", target, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent checker")
	}
}

func TestCheckerEngine_RunWithScheduledTrigger(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker_sched",
		Name: "Test Checker Scheduled",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_sched", status: happydns.StatusOK},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	planID, _ := happydns.NewRandomIdentifier()
	plan := &happydns.CheckPlan{
		Id:        planID,
		CheckerID: "test_checker_sched",
		Target:    target,
	}

	exec, err := engine.CreateExecution("test_checker_sched", target, plan)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	// Verify the trigger is set to Schedule when plan is provided.
	if exec.Trigger.Type != happydns.TriggerSchedule {
		t.Errorf("expected TriggerSchedule, got %v", exec.Trigger.Type)
	}
	if exec.PlanID == nil || !exec.PlanID.Equals(planID) {
		t.Errorf("expected PlanID %s, got %v", planID, exec.PlanID)
	}

	eval, err := engine.RunExecution(context.Background(), exec, plan, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}
	if eval == nil {
		t.Fatal("expected non-nil evaluation")
	}
}

func TestCheckerEngine_RunExecution_CheckerDisappeared(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker_disappear",
		Name: "Test Checker Disappear",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_d", status: happydns.StatusOK},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String()}

	exec, err := engine.CreateExecution("test_checker_disappear", target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	// Simulate the checker being unregistered between Create and Run
	// by using a fake checker ID on the execution.
	exec.CheckerID = "vanished_checker"

	_, err = engine.RunExecution(context.Background(), exec, nil, nil)
	if err == nil {
		t.Fatal("expected error when checker has disappeared")
	}

	// The execution should be marked as failed.
	persisted, err := store.GetExecution(exec.Id)
	if err != nil {
		t.Fatalf("GetExecution() returned error: %v", err)
	}
	if persisted.Status != happydns.ExecutionFailed {
		t.Errorf("expected execution status Failed, got %d", persisted.Status)
	}
}

func TestCheckerEngine_RunPopulatesObservationCache(t *testing.T) {
	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	checker.RegisterObservationProvider(&testObservationProvider{})
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "test_checker_cache",
		Name: "Test Checker Cache",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_cache", status: happydns.StatusOK},
		},
	})

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	exec, err := engine.CreateExecution("test_checker_cache", target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	_, err = engine.RunExecution(context.Background(), exec, nil, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	// Verify observation cache was populated for the "test_obs" key.
	entry, err := store.GetCachedObservation(target, "test_obs")
	if err != nil {
		t.Fatalf("GetCachedObservation() returned error: %v", err)
	}
	if entry.SnapshotID.IsEmpty() {
		t.Error("expected non-empty snapshot ID in cache entry")
	}
	if entry.CollectedAt.IsZero() {
		t.Error("expected non-zero CollectedAt in cache entry")
	}

	// Verify the cached snapshot actually exists and contains the data.
	snap, err := store.GetSnapshot(entry.SnapshotID)
	if err != nil {
		t.Fatalf("GetSnapshot() returned error: %v", err)
	}
	if _, ok := snap.Data["test_obs"]; !ok {
		t.Error("expected 'test_obs' key in snapshot data")
	}
}

func TestCheckerEngine_RunWithEndpointOverride(t *testing.T) {
	// Start a fake remote checker that responds to POST /collect.
	var gotRequest happydns.ExternalCollectRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/collect" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Data: json.RawMessage(`{"value":99}`),
		})
	}))
	defer srv.Close()

	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	const checkerID = "test_checker_endpoint"
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   checkerID,
		Name: "Test Checker Endpoint",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		ObservationKeys: []happydns.ObservationKey{"test_obs"},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_endpoint", status: happydns.StatusOK},
		},
	})

	// Store admin-level configuration with the endpoint pointing to our test server.
	if err := store.UpdateCheckerConfiguration(checkerID, nil, nil, nil, happydns.CheckerOptions{
		"endpoint": srv.URL,
	}); err != nil {
		t.Fatalf("UpdateCheckerConfiguration() returned error: %v", err)
	}

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	exec, err := engine.CreateExecution(checkerID, target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	eval, err := engine.RunExecution(context.Background(), exec, nil, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	if eval == nil {
		t.Fatal("RunExecution() returned nil evaluation")
	}

	// The engine should have delegated to the HTTP endpoint.
	if gotRequest.Key != "test_obs" {
		t.Errorf("remote received Key = %q, want %q", gotRequest.Key, "test_obs")
	}

	if exec.Result.Status != happydns.StatusOK {
		t.Errorf("expected status OK, got %s", exec.Result.Status)
	}
}

func TestCheckerEngine_RunWithEndpointOverride_RemoteFailure(t *testing.T) {
	// Start a remote checker that always returns an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(happydns.ExternalCollectResponse{
			Error: "remote collector is down",
		})
	}))
	defer srv.Close()

	store, err := inmemory.Instantiate()
	if err != nil {
		t.Fatalf("Instantiate() returned error: %v", err)
	}

	const checkerID = "test_checker_endpoint_fail"
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   checkerID,
		Name: "Test Checker Endpoint Fail",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		ObservationKeys: []happydns.ObservationKey{"test_obs"},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_endpoint_fail", status: happydns.StatusOK},
		},
	})

	if err := store.UpdateCheckerConfiguration(checkerID, nil, nil, nil, happydns.CheckerOptions{
		"endpoint": srv.URL,
	}); err != nil {
		t.Fatalf("UpdateCheckerConfiguration() returned error: %v", err)
	}

	optionsUC := checkerUC.NewCheckerOptionsUsecase(store, nil)
	engine := checkerUC.NewCheckerEngine(optionsUC, store, store, store, store)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	exec, err := engine.CreateExecution(checkerID, target, nil)
	if err != nil {
		t.Fatalf("CreateExecution() returned error: %v", err)
	}

	eval, err := engine.RunExecution(context.Background(), exec, nil, nil)
	if err != nil {
		t.Fatalf("RunExecution() returned error: %v", err)
	}

	// The rule should report an error state because observation collection failed.
	if exec.Result.Status != happydns.StatusError {
		t.Errorf("expected status Error, got %s", exec.Result.Status)
	}

	if len(eval.States) != 1 {
		t.Fatalf("expected 1 state, got %d", len(eval.States))
	}
}
