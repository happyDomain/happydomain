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
	"log"
	"sync"
	"time"

	"git.happydns.org/happyDomain/model"
)

// JanitorUserResolver resolves a user from a CheckTarget so the janitor can
// honour per-user retention overrides stored in UserQuota.
type JanitorUserResolver interface {
	GetUser(id happydns.Identifier) (*happydns.User, error)
}

// Janitor periodically prunes old check executions and evaluations according
// to the tiered RetentionPolicy. It is the long-tail enforcement counterpart
// of the cheap hard cap applied at execution-creation time.
type Janitor struct {
	planStore     CheckPlanStorage
	execStore     ExecutionStorage
	evalStore     CheckEvaluationStorage
	snapStore     ObservationSnapshotStorage
	userResolver  JanitorUserResolver
	defaultPolicy RetentionPolicy
	interval      time.Duration

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
}

// NewJanitor builds a Janitor that runs every `interval`. The defaultPolicy
// is applied to executions of users that did not customize their retention
// horizon via UserQuota. evalStore and snapStore may be nil if evaluation
// pruning is not desired.
func NewJanitor(planStore CheckPlanStorage, execStore ExecutionStorage, evalStore CheckEvaluationStorage, snapStore ObservationSnapshotStorage, userResolver JanitorUserResolver, defaultPolicy RetentionPolicy, interval time.Duration) *Janitor {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &Janitor{
		planStore:     planStore,
		execStore:     execStore,
		evalStore:     evalStore,
		snapStore:     snapStore,
		userResolver:  userResolver,
		defaultPolicy: defaultPolicy,
		interval:      interval,
	}
}

// Start launches the janitor loop in a goroutine. It runs an immediate sweep
// once the loop is up.
func (j *Janitor) Start(ctx context.Context) {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	j.cancel = cancel
	j.done = make(chan struct{})
	j.running = true
	j.mu.Unlock()

	go j.loop(ctx)
}

// Stop halts the janitor and waits for the current sweep to finish.
func (j *Janitor) Stop() {
	j.mu.Lock()
	cancel := j.cancel
	done := j.done
	j.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	j.mu.Lock()
	j.running = false
	j.mu.Unlock()
}

func (j *Janitor) loop(ctx context.Context) {
	defer close(j.done)

	// Run immediately, then on the configured interval.
	j.RunOnce(ctx)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.RunOnce(ctx)
		}
	}
}

// RunOnce performs a single sweep over all check plans, applying the per-user
// retention policy to both executions and evaluations. Returns the total
// number of records deleted (executions + evaluations).
func (j *Janitor) RunOnce(ctx context.Context) int {
	iter, err := j.planStore.ListAllCheckPlans()
	if err != nil {
		log.Printf("Janitor: failed to list check plans: %v", err)
		return 0
	}
	defer iter.Close()

	now := time.Now()
	deleted := 0

	// Cache user policies to avoid resolving the same user repeatedly.
	policyByUser := map[string]RetentionPolicy{}

	for iter.Next() {
		select {
		case <-ctx.Done():
			return deleted
		default:
		}

		plan := iter.Item()
		if plan == nil {
			continue
		}

		policy := j.policyForTarget(plan.Target, policyByUser)
		hardCutoff := now.AddDate(0, 0, -policy.RetentionDays)

		// Prune executions using the tiered retention policy.
		execs, err := j.execStore.ListExecutionsByPlan(plan.Id)
		if err != nil {
			log.Printf("Janitor: failed to list executions for plan %s: %v", plan.Id.String(), err)
		} else if len(execs) > 0 {
			// All executions share the same (CheckerID, Target) since they come
			// from a single plan, so Decide's internal grouping is a no-op here.
			_, drop := policy.Decide(execs, now)

			for _, id := range drop {
				if err := j.execStore.DeleteExecution(id); err != nil {
					log.Printf("Janitor: failed to delete execution %s: %v", id.String(), err)
					continue
				}
				deleted++
			}
		}

		// Prune evaluations older than the hard cutoff.
		if j.evalStore != nil {
			deleted += j.pruneEvaluations(plan.Id, hardCutoff)
		}
	}

	if err := iter.Err(); err != nil {
		log.Printf("Janitor: iterator error while walking check plans: %v", err)
	}

	if deleted > 0 {
		log.Printf("Janitor: pruned %d records", deleted)
	}
	return deleted
}

// pruneEvaluations deletes evaluations for the given plan that are older than
// the cutoff, along with their associated snapshots.
func (j *Janitor) pruneEvaluations(planID happydns.Identifier, cutoff time.Time) int {
	evals, err := j.evalStore.ListEvaluationsByPlan(planID)
	if err != nil {
		log.Printf("Janitor: failed to list evaluations for plan %s: %v", planID.String(), err)
		return 0
	}

	deleted := 0
	for _, eval := range evals {
		if eval.EvaluatedAt.Before(cutoff) {
			// Delete the associated snapshot first.
			if j.snapStore != nil && !eval.SnapshotID.IsEmpty() {
				if err := j.snapStore.DeleteSnapshot(eval.SnapshotID); err != nil {
					log.Printf("Janitor: failed to delete snapshot %s: %v", eval.SnapshotID.String(), err)
				}
			}
			if err := j.evalStore.DeleteEvaluation(eval.Id); err != nil {
				log.Printf("Janitor: failed to delete evaluation %s: %v", eval.Id.String(), err)
				continue
			}
			deleted++
		}
	}
	return deleted
}

func (j *Janitor) policyForTarget(target happydns.CheckTarget, cache map[string]RetentionPolicy) RetentionPolicy {
	uid := target.UserId
	if uid == "" || j.userResolver == nil {
		return j.defaultPolicy
	}
	if p, ok := cache[uid]; ok {
		return p
	}
	policy := j.defaultPolicy
	id, err := happydns.NewIdentifierFromString(uid)
	if err == nil {
		if user, err := j.userResolver.GetUser(id); err == nil && user != nil {
			if user.Quota.RetentionDays > 0 {
				policy = DefaultRetentionPolicy(user.Quota.RetentionDays)
			}
		}
	}
	cache[uid] = policy
	return policy
}
