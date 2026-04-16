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
	"encoding/json"
	"log"
	"slices"
	"time"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

// worstStatusMaxExecs is the maximum number of executions fetched when
// computing worst-status aggregations.  It prevents unbounded memory usage
// on long-lived accounts while being generous enough for any realistic
// scenario.
const worstStatusMaxExecs = 10000

// CheckStatusUsecase handles aggregation of checker statuses and evaluation/execution queries.
type CheckStatusUsecase struct {
	planStore CheckPlanStorage
	evalStore CheckEvaluationStorage
	execStore ExecutionStorage
	snapStore ObservationSnapshotStorage
}

// NewCheckStatusUsecase creates a new CheckStatusUsecase.
func NewCheckStatusUsecase(planStore CheckPlanStorage, evalStore CheckEvaluationStorage, execStore ExecutionStorage, snapStore ObservationSnapshotStorage) *CheckStatusUsecase {
	return &CheckStatusUsecase{
		planStore: planStore,
		evalStore: evalStore,
		execStore: execStore,
		snapStore: snapStore,
	}
}

// BudgetChecker exposes the per-user daily quota to the API layer.
//
// RateLimiterFor hands out a snapshot predicate that reports whether a
// scheduled execution of a given interval would be denied by the user's
// daily budget — either because the hard MaxChecksPerDay cap is reached,
// or because interval-aware throttling is skipping short-interval jobs to
// protect rarer ones. The snapshot is resolved once per user so callers
// can evaluate many intervals without repeated lookups.
//
// AllowWithInterval and IncrementUsage let the manual-trigger endpoint
// enforce the same quota as the scheduler: check before creating the
// execution, increment on success. All three are implemented by
// *UserGater.
type BudgetChecker interface {
	RateLimiterFor(userID string) func(time.Duration) bool
	AllowWithInterval(target happydns.CheckTarget, interval time.Duration) bool
	IncrementUsage(target happydns.CheckTarget)
}

// ListPlannedExecutions returns synthetic Execution records for upcoming scheduled jobs.
// Returns nil if provider is nil. When budgetChecker is non-nil, each planned
// execution is marked with status ExecutionRateLimited instead of
// ExecutionPending if a job of that interval would be denied by the user's
// daily budget at the moment of the call.
func ListPlannedExecutions(provider PlannedJobProvider, budgetChecker BudgetChecker, checkerID string, target happydns.CheckTarget) []*happydns.Execution {
	if provider == nil {
		return nil
	}
	jobs := provider.GetPlannedJobsForChecker(checkerID, target)
	result := make([]*happydns.Execution, 0, len(jobs))

	// Resolve the user's rate-limit predicate once, so the loop below does
	// not reacquire the UserGater's budget lock for every planned job.
	isLimited := func(time.Duration) bool { return false }
	if budgetChecker != nil {
		isLimited = budgetChecker.RateLimiterFor(target.UserId)
	}

	for _, job := range jobs {
		status := happydns.ExecutionPending
		if isLimited(job.Interval) {
			status = happydns.ExecutionRateLimited
		}
		exec := &happydns.Execution{
			CheckerID: job.CheckerID,
			PlanID:    job.PlanID,
			Target:    job.Target,
			Trigger:   happydns.TriggerInfo{Type: happydns.TriggerSchedule},
			StartedAt: job.NextRun,
			Status:    status,
		}
		result = append(result, exec)
	}
	return result
}

// ListCheckerStatuses aggregates checkers, plans, and latest evaluations into a status list.
func (u *CheckStatusUsecase) ListCheckerStatuses(target happydns.CheckTarget) ([]happydns.CheckerStatus, error) {
	checkers := checkerPkg.GetCheckers()
	plans, err := u.planStore.ListCheckPlansByTarget(target)
	if err != nil {
		return nil, err
	}

	planByChecker := make(map[string]*happydns.CheckPlan)
	for _, p := range plans {
		planByChecker[p.CheckerID] = p
	}

	var result []happydns.CheckerStatus
	for _, def := range checkers {
		switch target.Scope() {
		case happydns.CheckScopeDomain:
			if !def.Availability.ApplyToDomain {
				continue
			}
		case happydns.CheckScopeService:
			if !def.Availability.ApplyToService {
				continue
			}
			if len(def.Availability.LimitToServices) > 0 && target.ServiceType != "" {
				if !slices.Contains(def.Availability.LimitToServices, target.ServiceType) {
					continue
				}
			}
		}

		status := happydns.CheckerStatus{
			CheckerDefinition: def,
			Plan:              planByChecker[def.ID],
			Enabled:           true,
		}

		enabledRules := make(map[string]bool, len(def.Rules))
		for _, rule := range def.Rules {
			enabledRules[rule.Name()] = true
		}
		if status.Plan != nil {
			status.Enabled = !status.Plan.IsFullyDisabled()
			for ruleName := range enabledRules {
				enabledRules[ruleName] = status.Plan.IsRuleEnabled(ruleName)
			}
		}
		status.EnabledRules = enabledRules

		execs, err := u.execStore.ListExecutionsByChecker(def.ID, target, 1, nil)
		if err != nil {
			log.Printf("ListCheckerStatuses: failed to fetch latest execution for checker %s: %v", def.ID, err)
		} else if len(execs) > 0 {
			status.LatestExecution = execs[0]
		}

		result = append(result, status)
	}

	if result == nil {
		result = []happydns.CheckerStatus{}
	}
	return result, nil
}

// GetExecution returns a specific execution by ID after verifying scope ownership.
func (u *CheckStatusUsecase) GetExecution(scope happydns.CheckTarget, execID happydns.Identifier) (*happydns.Execution, error) {
	exec, err := u.execStore.GetExecution(execID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, exec.Target) {
		return nil, happydns.ErrExecutionNotFound
	}
	return exec, nil
}

// ListExecutionsByChecker returns executions for a checker on a target, up to limit.
func (u *CheckStatusUsecase) ListExecutionsByChecker(checkerID string, target happydns.CheckTarget, limit int) ([]*happydns.Execution, error) {
	return u.execStore.ListExecutionsByChecker(checkerID, target, limit, nil)
}

// GetObservationsByExecution returns the observation snapshot for an execution after verifying scope.
func (u *CheckStatusUsecase) GetObservationsByExecution(scope happydns.CheckTarget, execID happydns.Identifier) (*happydns.ObservationSnapshot, error) {
	exec, err := u.execStore.GetExecution(execID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, exec.Target) {
		return nil, happydns.ErrExecutionNotFound
	}
	return u.snapshotForExecution(exec)
}

// DeleteExecution deletes an execution record by ID after verifying scope ownership.
func (u *CheckStatusUsecase) DeleteExecution(scope happydns.CheckTarget, execID happydns.Identifier) error {
	exec, err := u.execStore.GetExecution(execID)
	if err != nil {
		return err
	}
	if !targetMatchesResource(scope, exec.Target) {
		return happydns.ErrExecutionNotFound
	}
	return u.execStore.DeleteExecution(execID)
}

// DeleteExecutionsByChecker deletes all executions for a checker on a target.
func (u *CheckStatusUsecase) DeleteExecutionsByChecker(checkerID string, target happydns.CheckTarget) error {
	return u.execStore.DeleteExecutionsByChecker(checkerID, target)
}

// worstStatuses groups executions by a key extracted via keyFn, keeps only
// the latest execution per (key, checker) pair, and returns the worst status
// per key.
func worstStatuses(execs []*happydns.Execution, keyFn func(*happydns.Execution) string) map[string]*happydns.Status {
	type groupKey struct {
		key     string
		checker string
	}
	latest := map[groupKey]*happydns.Execution{}
	for _, exec := range execs {
		k := keyFn(exec)
		if k == "" || exec.Status != happydns.ExecutionDone {
			continue
		}
		gk := groupKey{key: k, checker: exec.CheckerID}
		if prev, ok := latest[gk]; !ok || exec.StartedAt.After(prev.StartedAt) {
			latest[gk] = exec
		}
	}

	worst := map[string]*happydns.Status{}
	for gk, exec := range latest {
		s := exec.Result.Status
		if s == happydns.StatusUnknown {
			continue
		}
		if prev, ok := worst[gk.key]; !ok || s > *prev {
			worst[gk.key] = &s
		}
	}

	if len(worst) == 0 {
		return nil
	}
	return worst
}

// GetWorstDomainStatuses fetches all executions for a user and returns the worst
// (most critical) status per domain. It keeps only the latest execution per
// (domain, checker) pair and reports the worst status among them.
func (u *CheckStatusUsecase) GetWorstDomainStatuses(userId happydns.Identifier) (map[string]*happydns.Status, error) {
	execs, err := u.execStore.ListExecutionsByUser(userId, worstStatusMaxExecs, nil)
	if err != nil {
		return nil, err
	}
	return worstStatuses(execs, func(e *happydns.Execution) string {
		return e.Target.DomainId
	}), nil
}

// GetWorstServiceStatuses returns the worst check status for each service in the zone.
// It fetches all executions for the domain in a single query, then aggregates
// the worst status per service in memory.
func (u *CheckStatusUsecase) GetWorstServiceStatuses(userId happydns.Identifier, domainId happydns.Identifier) (map[string]*happydns.Status, error) {
	execs, err := u.execStore.ListExecutionsByDomain(domainId, worstStatusMaxExecs, nil)
	if err != nil {
		return nil, err
	}
	return worstStatuses(execs, func(e *happydns.Execution) string {
		return e.Target.ServiceId
	}), nil
}

// GetResultsByExecution returns the evaluation (with per-rule states) for an execution after verifying scope.
func (u *CheckStatusUsecase) GetResultsByExecution(scope happydns.CheckTarget, execID happydns.Identifier) (*happydns.CheckEvaluation, error) {
	exec, err := u.execStore.GetExecution(execID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, exec.Target) {
		return nil, happydns.ErrExecutionNotFound
	}
	if exec.EvaluationID == nil {
		return nil, happydns.ErrCheckEvaluationNotFound
	}
	return u.evalStore.GetEvaluation(*exec.EvaluationID)
}

// snapshotForExecution returns the observation snapshot associated with an execution.
func (u *CheckStatusUsecase) snapshotForExecution(exec *happydns.Execution) (*happydns.ObservationSnapshot, error) {
	if exec.EvaluationID == nil {
		return nil, happydns.ErrCheckEvaluationNotFound
	}

	eval, err := u.evalStore.GetEvaluation(*exec.EvaluationID)
	if err != nil {
		return nil, err
	}

	return u.snapStore.GetSnapshot(eval.SnapshotID)
}

// extractMetricsFromExecution extracts metrics from a single execution's snapshot.
func (u *CheckStatusUsecase) extractMetricsFromExecution(exec *happydns.Execution) ([]happydns.CheckMetric, error) {
	if exec.Status != happydns.ExecutionDone || exec.EvaluationID == nil {
		return nil, nil
	}

	snap, err := u.snapshotForExecution(exec)
	if err != nil {
		return nil, err
	}

	return checkerPkg.GetAllMetrics(snap)
}

// extractMetricsFromExecutions extracts metrics from a list of executions.
func (u *CheckStatusUsecase) extractMetricsFromExecutions(execs []*happydns.Execution) ([]happydns.CheckMetric, error) {
	var allMetrics []happydns.CheckMetric
	for _, exec := range execs {
		metrics, err := u.extractMetricsFromExecution(exec)
		if err != nil {
			log.Printf("extractMetricsFromExecutions: exec %s: %v", exec.Id.String(), err)
			continue
		}
		allMetrics = append(allMetrics, metrics...)
	}
	return allMetrics, nil
}

// doneExecution is a filter predicate for ListExecutionsBy* that keeps only
// executions that have completed successfully and can produce metrics.
func doneExecution(e *happydns.Execution) bool {
	return e.Status == happydns.ExecutionDone && e.EvaluationID != nil
}

// GetMetricsByExecution extracts metrics from a single execution's snapshot after verifying scope.
func (u *CheckStatusUsecase) GetMetricsByExecution(scope happydns.CheckTarget, execID happydns.Identifier) ([]happydns.CheckMetric, error) {
	exec, err := u.execStore.GetExecution(execID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, exec.Target) {
		return nil, happydns.ErrExecutionNotFound
	}
	return u.extractMetricsFromExecution(exec)
}

// GetMetricsByChecker extracts metrics from recent executions of a checker on a target.
func (u *CheckStatusUsecase) GetMetricsByChecker(checkerID string, target happydns.CheckTarget, limit int) ([]happydns.CheckMetric, error) {
	execs, err := u.execStore.ListExecutionsByChecker(checkerID, target, limit, doneExecution)
	if err != nil {
		return nil, err
	}
	return u.extractMetricsFromExecutions(execs)
}

// GetMetricsByUser extracts metrics from recent executions for a user across all checkers.
func (u *CheckStatusUsecase) GetMetricsByUser(userId happydns.Identifier, limit int) ([]happydns.CheckMetric, error) {
	execs, err := u.execStore.ListExecutionsByUser(userId, limit, doneExecution)
	if err != nil {
		return nil, err
	}
	return u.extractMetricsFromExecutions(execs)
}

// GetMetricsByDomain extracts metrics from recent executions for a domain (including services).
func (u *CheckStatusUsecase) GetMetricsByDomain(domainId happydns.Identifier, limit int) ([]happydns.CheckMetric, error) {
	execs, err := u.execStore.ListExecutionsByDomain(domainId, limit, doneExecution)
	if err != nil {
		return nil, err
	}
	return u.extractMetricsFromExecutions(execs)
}

// GetSnapshotByExecution returns the raw observation data for a single key from an execution after verifying scope.
func (u *CheckStatusUsecase) GetSnapshotByExecution(scope happydns.CheckTarget, execID happydns.Identifier, obsKey string) (json.RawMessage, error) {
	snap, err := u.GetObservationsByExecution(scope, execID)
	if err != nil {
		return nil, err
	}

	raw, ok := snap.Data[obsKey]
	if !ok {
		return nil, happydns.ErrSnapshotNotFound
	}

	return raw, nil
}
