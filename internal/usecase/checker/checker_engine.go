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
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	checkerPkg "git.happydns.org/happyDomain/internal/dnschecker"
	"git.happydns.org/happyDomain/model"
)

// executionCallback is the signature stored under onComplete. Wrapping it in a
// named type lets us hold it via atomic.Pointer without leaking the function
// type spelling everywhere.
type executionCallback func(*happydns.Execution, *happydns.CheckEvaluation)

// ExecutionCallbackSetter is implemented by checker engines that support
// notification callbacks after execution completion.
type ExecutionCallbackSetter interface {
	SetExecutionCallback(func(*happydns.Execution, *happydns.CheckEvaluation))
}

// Engine implements the happydns.CheckerEngine interface.
type Engine struct {
	optionsUC     *CheckerOptionsUsecase
	evalStore     CheckEvaluationStorage
	execStore     ExecutionStorage
	snapStore     ObservationSnapshotStorage
	cacheStore    ObservationCacheStorage
	entryStore    DiscoveryEntryStorage
	obsRefStore   DiscoveryObservationStorage
	relatedLookup checkerPkg.RelatedObservationLookup

	// onComplete is read concurrently by RunExecution from worker goroutines
	// while SetExecutionCallback writes it during app wiring (and potentially
	// later, defensively). atomic.Pointer keeps the load/store race-free.
	onComplete atomic.Pointer[executionCallback]
}

// SetExecutionCallback registers a callback invoked after each successful execution.
func (e *Engine) SetExecutionCallback(cb func(*happydns.Execution, *happydns.CheckEvaluation)) {
	if cb == nil {
		e.onComplete.Store(nil)
		return
	}
	wrapped := executionCallback(cb)
	e.onComplete.Store(&wrapped)
}

// NewCheckerEngine creates a new CheckerEngine implementation. Passing nil
// for entryStore/obsRefStore disables cross-checker discovery; the engine
// then behaves exactly as before the discovery mechanism was introduced.
func NewCheckerEngine(
	optionsUC *CheckerOptionsUsecase,
	evalStore CheckEvaluationStorage,
	execStore ExecutionStorage,
	snapStore ObservationSnapshotStorage,
	cacheStore ObservationCacheStorage,
	entryStore DiscoveryEntryStorage,
	obsRefStore DiscoveryObservationStorage,
) *Engine {
	return &Engine{
		optionsUC:     optionsUC,
		evalStore:     evalStore,
		execStore:     execStore,
		snapStore:     snapStore,
		cacheStore:    cacheStore,
		entryStore:    entryStore,
		obsRefStore:   obsRefStore,
		relatedLookup: newRelatedLookup(entryStore, obsRefStore, snapStore),
	}
}

// CreateExecution validates the checker and creates a pending Execution record.
func (e *Engine) CreateExecution(checkerID string, target happydns.CheckTarget, plan *happydns.CheckPlan) (*happydns.Execution, error) {
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
func (e *Engine) RunExecution(ctx context.Context, exec *happydns.Execution, plan *happydns.CheckPlan, runOpts happydns.CheckerOptions) (*happydns.CheckEvaluation, error) {
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

	// Fire notification callback. The callback decides synchronously whether
	// to notify and advances state, but actual sender invocations are
	// dispatched to a worker pool so a slow channel cannot wedge the engine.
	if cb := e.onComplete.Load(); cb != nil {
		(*cb)(exec, eval)
	}

	return eval, nil
}

func (e *Engine) runPipeline(ctx context.Context, def *happydns.CheckerDefinition, target happydns.CheckTarget, plan *happydns.CheckPlan, planID *happydns.Identifier, runOpts happydns.CheckerOptions) (happydns.CheckState, *happydns.CheckEvaluation, error) {
	// Resolve options (stored + run + auto-fill).
	mergedOpts, injectedEntries, err := e.optionsUC.BuildMergedCheckerOptionsWithAutoFill(def.ID, happydns.TargetIdentifier(target.UserId), happydns.TargetIdentifier(target.DomainId), happydns.TargetIdentifier(target.ServiceId), runOpts)
	if err != nil {
		return happydns.CheckState{}, nil, fmt.Errorf("resolving options: %w", err)
	}

	// Build observation cache lookup for cross-checker reuse.
	var cacheLookup checkerPkg.ObservationCacheLookup
	if e.cacheStore != nil {
		cacheLookup = func(target happydns.CheckTarget, key happydns.ObservationKey) (json.RawMessage, time.Time, error) {
			entry, err := e.cacheStore.GetCachedObservation(target, key)
			if err != nil {
				return nil, time.Time{}, err
			}
			snap, err := e.snapStore.GetSnapshot(entry.SnapshotID)
			if err != nil {
				return nil, time.Time{}, err
			}
			raw, ok := snap.Data[key]
			if !ok {
				return nil, time.Time{}, fmt.Errorf("observation %q not in snapshot", key)
			}
			return raw, entry.CollectedAt, nil
		}
	}

	var freshness time.Duration
	if plan != nil && plan.Interval != nil {
		freshness = *plan.Interval
	} else if plan != nil && def.Interval != nil {
		freshness = def.Interval.Default
	}

	// Create observation context for lazy data collection.
	obsCtx := checkerPkg.NewObservationContext(target, mergedOpts, cacheLookup, freshness)

	if e.relatedLookup != nil {
		obsCtx.SetRelatedLookup(def.ID, e.relatedLookup)
	}

	// If an endpoint is configured, override observation providers with HTTP
	// transport. The "endpoint" AdminOpt (added by RegisterExternalizableChecker)
	// may be set in the DB or by a -checker-<id>-endpoint CLI flag; both feed
	// into mergedOpts above, with the CLI value winning.
	if endpoint, ok := mergedOpts["endpoint"].(string); ok && endpoint != "" {
		for _, key := range def.ObservationKeys {
			obsCtx.SetProviderOverride(key, checkerPkg.NewHTTPObservationProvider(key, endpoint))
		}
	}

	// Evaluate all rules, skipping disabled ones.
	states := make([]happydns.CheckState, 0, len(def.Rules))
	for _, rule := range def.Rules {
		if plan != nil && !plan.IsRuleEnabled(rule.Name()) {
			continue
		}
		ruleStates := rule.Evaluate(ctx, obsCtx, mergedOpts)
		if len(ruleStates) == 0 {
			ruleStates = []happydns.CheckState{{
				Status:  happydns.StatusUnknown,
				Message: "rule returned no state",
			}}
		}
		for i := range ruleStates {
			ruleStates[i].RuleName = rule.Name()
		}
		states = append(states, ruleStates...)
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

	// Update observation cache pointers for cross-checker reuse.
	if e.cacheStore != nil {
		for key := range snap.Data {
			if err := e.cacheStore.PutCachedObservation(target, key, &happydns.ObservationCacheEntry{
				SnapshotID:  snap.Id,
				CollectedAt: snap.CollectedAt,
			}); err != nil {
				log.Printf("warning: failed to cache observation %q for target %s: %v", key, target.String(), err)
			}
		}
	}

	// Always replace, including with an empty slice, so stale entries vanish.
	if e.entryStore != nil {
		var published []happydns.DiscoveryEntry
		for _, list := range obsCtx.Entries() {
			published = append(published, list...)
		}
		if err := e.entryStore.ReplaceDiscoveryEntries(def.ID, target, published); err != nil {
			log.Printf("warning: failed to replace discovery entries for %s on %s: %v", def.ID, target.String(), err)
		}
	}

	// Persist the consumer→entry lineage: for each entry that was fed into
	// this run via AutoFillDiscoveryEntries, link every observation we just
	// stored to the original producer's (producer, target, ref) tuple. A
	// later GetRelated call from the producer walks these refs.
	if e.obsRefStore != nil && len(injectedEntries) > 0 && len(snap.Data) > 0 {
		for _, entry := range injectedEntries {
			base := happydns.DiscoveryObservationRef{
				ProducerID:  entry.ProducerID,
				Target:      entry.Target,
				Ref:         entry.Ref,
				ConsumerID:  def.ID,
				SnapshotID:  snap.Id,
				CollectedAt: snap.CollectedAt,
			}
			for key := range snap.Data {
				ref := base
				ref.Key = key
				if err := e.obsRefStore.PutDiscoveryObservationRef(&ref); err != nil {
					log.Printf("warning: failed to persist observation ref for %s/%s: %v", entry.ProducerID, entry.Ref, err)
				}
			}
		}
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

// RecoverStaleExecutions scans all executions and marks any still in Pending
// or Running state as Failed. It is intended to be called at startup to
// reconcile state left over from a previous process that crashed or was
// killed mid-execution: without it, the affected executions would remain
// "running" forever in the UI. Returns the number of executions updated.
func (e *Engine) RecoverStaleExecutions(ctx context.Context) (int, error) {
	iter, err := e.execStore.ListAllExecutions()
	if err != nil {
		return 0, fmt.Errorf("listing executions: %w", err)
	}
	defer iter.Close()

	n := 0
	for iter.Next() {
		exec := iter.Item()
		if exec.Status != happydns.ExecutionPending && exec.Status != happydns.ExecutionRunning {
			continue
		}
		endTime := time.Now()
		exec.Status = happydns.ExecutionFailed
		exec.EndedAt = &endTime
		if exec.Error == "" {
			exec.Error = "execution interrupted by server restart"
		}
		if err := e.execStore.UpdateExecution(exec); err != nil {
			log.Printf("CheckerEngine: failed to recover stale execution %s: %v", exec.Id.String(), err)
			continue
		}
		n++
	}
	return n, nil
}

// RelatedLookup exposes the engine's Related resolver so controllers can
// build ReportContexts with cross-checker observations pre-resolved. Returns
// nil when discovery storage is not wired.
func (e *Engine) RelatedLookup() checkerPkg.RelatedObservationLookup {
	return e.relatedLookup
}

// newRelatedLookup builds the RelatedObservationLookup closure once at engine
// construction time. Returns nil when any required store is absent.
func newRelatedLookup(entryStore DiscoveryEntryStorage, obsRefStore DiscoveryObservationStorage, snapStore ObservationSnapshotStorage) checkerPkg.RelatedObservationLookup {
	if entryStore == nil || obsRefStore == nil || snapStore == nil {
		return nil
	}
	return func(_ context.Context, producerCheckerID string, target happydns.CheckTarget, key happydns.ObservationKey) ([]happydns.RelatedObservation, error) {
		entries, err := entryStore.ListDiscoveryEntriesByProducer(producerCheckerID, target)
		if err != nil {
			return nil, err
		}
		var out []happydns.RelatedObservation
		snapCache := make(map[string]*happydns.ObservationSnapshot)
		for _, entry := range entries {
			refs, err := obsRefStore.ListDiscoveryObservationRefs(producerCheckerID, target, entry.Ref)
			if err != nil {
				continue
			}
			for _, r := range refs {
				if r.Key != key {
					continue
				}
				snapID := r.SnapshotID.String()
				snap, ok := snapCache[snapID]
				if !ok {
					snap, err = snapStore.GetSnapshot(r.SnapshotID)
					if err != nil {
						// Snapshot gone (TTL) — skip silently; implicit GC.
						continue
					}
					snapCache[snapID] = snap
				}
				data, ok := snap.Data[r.Key]
				if !ok {
					continue
				}
				out = append(out, happydns.RelatedObservation{
					CheckerID:   r.ConsumerID,
					Key:         r.Key,
					Data:        data,
					CollectedAt: r.CollectedAt,
					Ref:         r.Ref,
				})
			}
		}
		return out, nil
	}
}
