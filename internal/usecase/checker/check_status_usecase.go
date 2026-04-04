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
	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

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

// ListPlannedExecutions returns synthetic Execution records for upcoming scheduled jobs.
// Returns nil if provider is nil.
func ListPlannedExecutions(provider PlannedJobProvider, checkerID string, target happydns.CheckTarget) []*happydns.Execution {
	if provider == nil {
		return nil
	}
	jobs := provider.GetPlannedJobsForChecker(checkerID, target)
	result := make([]*happydns.Execution, 0, len(jobs))
	for _, job := range jobs {
		exec := &happydns.Execution{
			CheckerID: job.CheckerID,
			PlanID:    job.PlanID,
			Target:    job.Target,
			Trigger:   happydns.TriggerInfo{Type: happydns.TriggerSchedule},
			StartedAt: job.NextRun,
			Status:    happydns.ExecutionPending,
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

		execs, err := u.execStore.ListExecutionsByChecker(def.ID, target, 1)
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
	return u.execStore.ListExecutionsByChecker(checkerID, target, limit)
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

// GetWorstServiceStatuses returns the worst check status for each service in the zone.
// It iterates all services and all registered checkers, fetching the latest execution
// for each (service, checker) pair, and returns the worst status per service.
func (u *CheckStatusUsecase) GetWorstServiceStatuses(userId happydns.Identifier, domainId happydns.Identifier, zone *happydns.Zone) (map[string]*happydns.Status, error) {
	checkers := checkerPkg.GetCheckers()
	if len(checkers) == 0 {
		return nil, nil
	}

	result := make(map[string]*happydns.Status)
	for subdomain := range zone.Services {
		for _, svc := range zone.Services[subdomain] {
			target := happydns.CheckTarget{
				UserId:    &userId,
				DomainId:  &domainId,
				ServiceId: &svc.Id,
			}
			var worst *happydns.Status
			for _, def := range checkers {
				execs, err := u.execStore.ListExecutionsByChecker(def.ID, target, 1)
				if err != nil || len(execs) == 0 {
					continue
				}
				s := execs[0].Result.Status
				if worst == nil || s > *worst {
					worst = &s
				}
			}
			if worst != nil {
				result[svc.Id.String()] = worst
			}
		}
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
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
