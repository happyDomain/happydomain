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
	"time"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

// checkerEngine implements the happydns.CheckerEngine interface.
type checkerEngine struct {
	optionsUC     *CheckerOptionsUsecase
	evalStore     CheckEvaluationStorage
	execStore     ExecutionStorage
	snapStore     ObservationSnapshotStorage
	cacheStore    ObservationCacheStorage
	entryStore    DiscoveryEntryStorage
	obsRefStore   DiscoveryObservationStorage
	relatedLookup checkerPkg.RelatedObservationLookup
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
) happydns.CheckerEngine {
	return &checkerEngine{
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

	// If an endpoint is configured, override observation providers with HTTP transport.
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

// RelatedLookup exposes the engine's Related resolver so controllers can
// build ReportContexts with cross-checker observations pre-resolved. Returns
// nil when discovery storage is not wired.
func (e *checkerEngine) RelatedLookup() checkerPkg.RelatedObservationLookup {
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
