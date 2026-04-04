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

package checker

import (
	"context"
	"fmt"
	"log"
	"time"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

// checkerEngine implements the happydns.CheckerEngine interface.
type checkerEngine struct {
	optionsUC  *CheckerOptionsUsecase
	evalStore  CheckEvaluationStorage
	execStore  ExecutionStorage
	snapStore  ObservationSnapshotStorage
}

// NewCheckerEngine creates a new CheckerEngine implementation.
func NewCheckerEngine(
	optionsUC *CheckerOptionsUsecase,
	evalStore CheckEvaluationStorage,
	execStore ExecutionStorage,
	snapStore ObservationSnapshotStorage,
) happydns.CheckerEngine {
	return &checkerEngine{
		optionsUC: optionsUC,
		evalStore: evalStore,
		execStore: execStore,
		snapStore: snapStore,
	}
}

// CreateExecution validates the checker and creates a pending Execution record.
func (e *checkerEngine) CreateExecution(checkerID string, target happydns.CheckTarget, plan *happydns.CheckPlan) (*happydns.Execution, error) {
	if checkerPkg.FindChecker(checkerID) == nil {
		return nil, fmt.Errorf("%w: %s", happydns.ErrCheckerNotFound, checkerID)
	}

	// Determine trigger info.
	trigger := happydns.TriggerInfo{Type: happydns.TriggerManual}
	var planID *happydns.Identifier
	if plan != nil {
		planID = &plan.Id
		trigger.PlanID = planID
		trigger.Type = happydns.TriggerSchedule
	}

	// Create execution record.
	exec := &happydns.Execution{
		CheckerID: checkerID,
		PlanID:    planID,
		Target:    target,
		Trigger:   trigger,
		StartedAt: time.Now(),
		Status:    happydns.ExecutionPending,
	}
	if err := e.execStore.CreateExecution(exec); err != nil {
		return nil, fmt.Errorf("creating execution: %w", err)
	}

	return exec, nil
}

// RunExecution takes an existing execution and runs the checker pipeline.
func (e *checkerEngine) RunExecution(ctx context.Context, exec *happydns.Execution, plan *happydns.CheckPlan, runOpts happydns.CheckerOptions) (*happydns.CheckEvaluation, error) {
	log.Printf("CheckerEngine: running checker %s on %s", exec.CheckerID, exec.Target.String())

	def := checkerPkg.FindChecker(exec.CheckerID)
	if def == nil {
		endTime := time.Now()
		exec.Status = happydns.ExecutionFailed
		exec.EndedAt = &endTime
		exec.Error = fmt.Sprintf("checker not found: %s", exec.CheckerID)
		if err := e.execStore.UpdateExecution(exec); err != nil {
			log.Printf("CheckerEngine: failed to update execution: %v", err)
		}
		return nil, fmt.Errorf("%w: %s", happydns.ErrCheckerNotFound, exec.CheckerID)
	}

	// Mark as running.
	exec.Status = happydns.ExecutionRunning
	if err := e.execStore.UpdateExecution(exec); err != nil {
		log.Printf("CheckerEngine: failed to update execution: %v", err)
	}

	// Run the pipeline and handle failure.
	result, eval, err := e.runPipeline(ctx, def, exec.Target, plan, exec.PlanID, runOpts)
	if err != nil {
		log.Printf("CheckerEngine: checker %s on %s failed: %v", exec.CheckerID, exec.Target.String(), err)
		endTime := time.Now()
		exec.Status = happydns.ExecutionFailed
		exec.EndedAt = &endTime
		exec.Error = err.Error()
		if err := e.execStore.UpdateExecution(exec); err != nil {
			log.Printf("CheckerEngine: failed to update execution: %v", err)
		}
		return nil, err
	}

	// Mark as done.
	endTime := time.Now()
	exec.Status = happydns.ExecutionDone
	exec.EndedAt = &endTime
	exec.Result = result
	exec.EvaluationID = &eval.Id
	if err := e.execStore.UpdateExecution(exec); err != nil {
		log.Printf("CheckerEngine: failed to update execution: %v", err)
	}

	return eval, nil
}

func (e *checkerEngine) runPipeline(ctx context.Context, def *happydns.CheckerDefinition, target happydns.CheckTarget, plan *happydns.CheckPlan, planID *happydns.Identifier, runOpts happydns.CheckerOptions) (happydns.CheckState, *happydns.CheckEvaluation, error) {
	// Resolve options (stored + run + auto-fill).
	mergedOpts, err := e.optionsUC.BuildMergedCheckerOptionsWithAutoFill(def.ID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), runOpts)
	if err != nil {
		return happydns.CheckState{}, nil, fmt.Errorf("resolving options: %w", err)
	}

	// Create observation context for lazy data collection.
	obsCtx := checkerPkg.NewObservationContext(target, mergedOpts)

	// Evaluate all rules, skipping disabled ones.
	states := make([]happydns.CheckState, 0, len(def.Rules))
	for _, rule := range def.Rules {
		if plan != nil && !plan.IsRuleEnabled(rule.Name()) {
			continue
		}
		state := rule.Evaluate(ctx, obsCtx, mergedOpts)
		if state.Code == "" {
			state.Code = rule.Name()
		}
		states = append(states, state)
	}

	// Aggregate results.
	aggregator := def.Aggregator
	if aggregator == nil {
		aggregator = checkerPkg.WorstStatusAggregator{}
	}
	result := aggregator.Aggregate(states)

	// Persist observation snapshot.
	snap := &happydns.ObservationSnapshot{
		Target:      target,
		CollectedAt: time.Now(),
		Data:        obsCtx.Data(),
	}
	if err := e.snapStore.CreateSnapshot(snap); err != nil {
		return happydns.CheckState{}, nil, fmt.Errorf("creating snapshot: %w", err)
	}

	// Persist evaluation.
	eval := &happydns.CheckEvaluation{
		PlanID:      planID,
		CheckerID:   def.ID,
		Target:      target,
		SnapshotID:  snap.Id,
		EvaluatedAt: time.Now(),
		States:      states,
	}
	if err := e.evalStore.CreateEvaluation(eval); err != nil {
		return happydns.CheckState{}, nil, fmt.Errorf("creating evaluation: %w", err)
	}

	return result, eval, nil
}
