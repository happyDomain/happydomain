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
	"time"

	"git.happydns.org/happyDomain/model"
)

// SchedulerStateStorage provides persistence for scheduler state (e.g. last run time).
type SchedulerStateStorage interface {
	GetLastSchedulerRun() (time.Time, error)
	SetLastSchedulerRun(t time.Time) error
}

// DomainLister is the minimal interface needed by the scheduler to enumerate domains.
type DomainLister interface {
	ListAllDomains() (happydns.Iterator[happydns.Domain], error)
}

// ZoneGetter is the minimal interface needed by the scheduler to load zones for service discovery.
type ZoneGetter interface {
	GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error)
}

// CheckAutoFillStorage provides access to domain, zone and user data
// needed to resolve auto-fill field values at execution time.
type CheckAutoFillStorage interface {
	GetDomain(id happydns.Identifier) (*happydns.Domain, error)
	GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error)
	ListDomains(u *happydns.User) ([]*happydns.Domain, error)
	GetUser(id happydns.Identifier) (*happydns.User, error)
}

// CheckPlanStorage provides persistence for CheckPlan entities.
type CheckPlanStorage interface {
	ListAllCheckPlans() (happydns.Iterator[happydns.CheckPlan], error)
	ListCheckPlansByTarget(target happydns.CheckTarget) ([]*happydns.CheckPlan, error)
	ListCheckPlansByChecker(checkerID string) ([]*happydns.CheckPlan, error)
	ListCheckPlansByUser(userId happydns.Identifier) ([]*happydns.CheckPlan, error)
	GetCheckPlan(planID happydns.Identifier) (*happydns.CheckPlan, error)
	CreateCheckPlan(plan *happydns.CheckPlan) error
	UpdateCheckPlan(plan *happydns.CheckPlan) error
	RestoreCheckPlan(plan *happydns.CheckPlan) error
	DeleteCheckPlan(planID happydns.Identifier) error
	TidyCheckPlanIndexes() error
	ClearCheckPlans() error
}

// CheckerOptionsStorage provides persistence for checker options at different levels.
type CheckerOptionsStorage interface {
	ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptionsPositional], error)
	ListCheckerConfiguration(checkerName string) ([]*happydns.CheckerOptionsPositional, error)
	GetCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) ([]*happydns.CheckerOptionsPositional, error)
	UpdateCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error
	DeleteCheckerConfiguration(checkerName string, userId *happydns.Identifier, domainId *happydns.Identifier, serviceId *happydns.Identifier) error
	ClearCheckerConfigurations() error
}

// CheckEvaluationStorage provides persistence for check evaluation results.
type CheckEvaluationStorage interface {
	ListAllEvaluations() (happydns.Iterator[happydns.CheckEvaluation], error)
	ListEvaluationsByPlan(planID happydns.Identifier) ([]*happydns.CheckEvaluation, error)
	ListEvaluationsByChecker(checkerID string, target happydns.CheckTarget, limit int) ([]*happydns.CheckEvaluation, error)
	GetEvaluation(evalID happydns.Identifier) (*happydns.CheckEvaluation, error)
	GetLatestEvaluation(planID happydns.Identifier) (*happydns.CheckEvaluation, error)
	CreateEvaluation(eval *happydns.CheckEvaluation) error
	RestoreEvaluation(eval *happydns.CheckEvaluation) error
	DeleteEvaluation(evalID happydns.Identifier) error
	DeleteEvaluationsByChecker(checkerID string, target happydns.CheckTarget) error
	TidyEvaluationIndexes() error
	ClearEvaluations() error
}

// ExecutionStorage provides persistence for execution records.
type ExecutionStorage interface {
	ListAllExecutions() (happydns.Iterator[happydns.Execution], error)
	ListExecutionsByPlan(planID happydns.Identifier) ([]*happydns.Execution, error)
	ListExecutionsByChecker(checkerID string, target happydns.CheckTarget, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error)
	ListExecutionsByUser(userId happydns.Identifier, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error)
	ListExecutionsByDomain(domainId happydns.Identifier, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error)
	GetExecution(execID happydns.Identifier) (*happydns.Execution, error)
	CreateExecution(exec *happydns.Execution) error
	UpdateExecution(exec *happydns.Execution) error
	RestoreExecution(exec *happydns.Execution) error
	DeleteExecution(execID happydns.Identifier) error
	DeleteExecutionsByChecker(checkerID string, target happydns.CheckTarget) error
	TidyExecutionIndexes() error
	ClearExecutions() error
}

// PlannedJobProvider exposes upcoming scheduler jobs from the in-memory queue.
type PlannedJobProvider interface {
	GetPlannedJobsForChecker(checkerID string, target happydns.CheckTarget) []*SchedulerJob
}

// ObservationSnapshotStorage provides persistence for observation snapshots.
type ObservationSnapshotStorage interface {
	ListAllSnapshots() (happydns.Iterator[happydns.ObservationSnapshot], error)
	GetSnapshot(snapID happydns.Identifier) (*happydns.ObservationSnapshot, error)
	CreateSnapshot(snap *happydns.ObservationSnapshot) error
	DeleteSnapshot(snapID happydns.Identifier) error
	ClearSnapshots() error
}

// ObservationCacheStorage provides a lightweight cache mapping (target, observation key)
// to the snapshot that holds the most recent data.
type ObservationCacheStorage interface {
	ListAllCachedObservations() (happydns.Iterator[happydns.ObservationCacheEntry], error)
	GetCachedObservation(target happydns.CheckTarget, key happydns.ObservationKey) (*happydns.ObservationCacheEntry, error)
	PutCachedObservation(target happydns.CheckTarget, key happydns.ObservationKey, entry *happydns.ObservationCacheEntry) error
}

// DiscoveryEntryStorage persists DiscoveryEntry records published by a
// producer checker for a given target. Entries are replaced atomically on
// each collection cycle (see docs/checker-discovery.md).
type DiscoveryEntryStorage interface {
	// ListDiscoveryEntriesByTarget returns entries published for a target
	// across all producers — used to fill AutoFillDiscoveryEntries options.
	ListDiscoveryEntriesByTarget(target happydns.CheckTarget) ([]*happydns.StoredDiscoveryEntry, error)
	// ListDiscoveryEntriesByProducer returns entries a producer published
	// for a target — used to resolve GetRelated.
	ListDiscoveryEntriesByProducer(producerID string, target happydns.CheckTarget) ([]*happydns.StoredDiscoveryEntry, error)
	// ReplaceDiscoveryEntries atomically replaces the set of entries
	// stored for (producerID, target).
	ReplaceDiscoveryEntries(producerID string, target happydns.CheckTarget, entries []happydns.DiscoveryEntry) error
	DeleteDiscoveryEntriesByProducer(producerID string, target happydns.CheckTarget) error
	ListAllDiscoveryEntries() (happydns.Iterator[happydns.StoredDiscoveryEntry], error)
	RestoreDiscoveryEntry(entry *happydns.StoredDiscoveryEntry) error
	ClearDiscoveryEntries() error
}

// DiscoveryObservationStorage persists the lineage linking consumer-produced
// observations to the DiscoveryEntry records they covered.
type DiscoveryObservationStorage interface {
	PutDiscoveryObservationRef(ref *happydns.DiscoveryObservationRef) error
	ListDiscoveryObservationRefs(producerID string, target happydns.CheckTarget, ref string) ([]*happydns.DiscoveryObservationRef, error)
	DeleteDiscoveryObservationRefsForSnapshot(snapshotID happydns.Identifier) error
	ListAllDiscoveryObservationRefs() (happydns.Iterator[happydns.DiscoveryObservationRef], error)
	RestoreDiscoveryObservationRef(ref *happydns.DiscoveryObservationRef) error
	ClearDiscoveryObservationRefs() error
}
