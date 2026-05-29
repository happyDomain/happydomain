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

package database

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"git.happydns.org/happyDomain/model"
)

const (
	ExecutionPrimaryPrefix        = "chckexec|"
	ExecutionByCheckerIndexPrefix = "chckexec-chkr|"
	ExecutionByDomainIndexPrefix  = "chckexec-domain|"
	ExecutionByPlanIndexPrefix    = "chckexec-plan|"
	ExecutionByUserIndexPrefix    = "chckexec-user|"
)

// The checker, user and domain indexes embed a reverseChronoSegment derived
// from the execution's StartedAt, so a forward prefix scan returns the most
// recent executions first and can stop at the requested limit. The plan index
// keeps the plain {planId}|{execId} layout because ListExecutionsByPlan does
// not need recency ordering.
func executionCheckerIndexKey(checkerID string, target happydns.CheckTarget, startedAt time.Time, execId string) string {
	return fmt.Sprintf("%s%s|%s|%s|%s", ExecutionByCheckerIndexPrefix, checkerID, target.String(), reverseChronoSegment(startedAt), execId)
}

func executionUserIndexKey(userId string, startedAt time.Time, execId string) string {
	return fmt.Sprintf("%s%s|%s|%s", ExecutionByUserIndexPrefix, userId, reverseChronoSegment(startedAt), execId)
}

func executionDomainIndexKey(domainId string, startedAt time.Time, execId string) string {
	return fmt.Sprintf("%s%s|%s|%s", ExecutionByDomainIndexPrefix, domainId, reverseChronoSegment(startedAt), execId)
}

func (s *KVStorage) ListExecutionsByPlan(planID happydns.Identifier) ([]*happydns.Execution, error) {
	return listByIndex(s, fmt.Sprintf("%s%s|", ExecutionByPlanIndexPrefix, planID.String()), s.GetExecution)
}

// listRecentExecutions scans a time sortable index prefix whose entries are
// already ordered newest first, applies an optional filter predicate, and stops
// once limit matches have been collected.
func (s *KVStorage) listRecentExecutions(prefix string, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return listByPresortedIndex(s, prefix, s.GetExecution, limit, filter)
}

func (s *KVStorage) ListExecutionsByChecker(checkerID string, target happydns.CheckTarget, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return s.listRecentExecutions(fmt.Sprintf("%s%s|%s|", ExecutionByCheckerIndexPrefix, checkerID, target.String()), limit, filter)
}

func (s *KVStorage) ListExecutionsByUser(userId happydns.Identifier, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return s.listRecentExecutions(fmt.Sprintf("%s%s|", ExecutionByUserIndexPrefix, userId.String()), limit, filter)
}

func (s *KVStorage) ListExecutionsByDomain(domainId happydns.Identifier, limit int, filter func(*happydns.Execution) bool) ([]*happydns.Execution, error) {
	return s.listRecentExecutions(fmt.Sprintf("%s%s|", ExecutionByDomainIndexPrefix, domainId.String()), limit, filter)
}

func (s *KVStorage) ListAllExecutions() (happydns.Iterator[happydns.Execution], error) {
	iter := s.db.Search(ExecutionPrimaryPrefix)
	return NewKVIterator[happydns.Execution](s.db, iter), nil
}

func (s *KVStorage) GetExecution(execID happydns.Identifier) (*happydns.Execution, error) {
	exec := &happydns.Execution{}
	err := s.db.Get(fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, execID.String()), exec)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrExecutionNotFound
	}
	return exec, err
}

func (s *KVStorage) CreateExecution(exec *happydns.Execution) error {
	key, id, err := s.db.FindIdentifierKey(ExecutionPrimaryPrefix)
	if err != nil {
		return err
	}
	exec.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, exec); err != nil {
		return err
	}

	// Secondary index by plan.
	if exec.PlanID != nil {
		indexKey := fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, exec.PlanID.String(), exec.Id.String())
		if err := batch.Put(indexKey, true); err != nil {
			return err
		}
	}

	// Secondary index by checker+target.
	if err := batch.Put(executionCheckerIndexKey(exec.CheckerID, exec.Target, exec.StartedAt, exec.Id.String()), true); err != nil {
		return err
	}

	// Secondary index by user.
	if exec.Target.UserId != "" {
		if err := batch.Put(executionUserIndexKey(exec.Target.UserId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	// Secondary index by domain.
	if exec.Target.DomainId != "" {
		if err := batch.Put(executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	return batch.Commit()
}

// RestoreExecution writes an execution at its existing Id and rebuilds
// its secondary indexes. Used by the backup restore path.
func (s *KVStorage) RestoreExecution(exec *happydns.Execution) error {
	batch := s.db.NewBatch()
	if err := batch.Put(fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, exec.Id.String()), exec); err != nil {
		return err
	}

	if exec.PlanID != nil {
		indexKey := fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, exec.PlanID.String(), exec.Id.String())
		if err := batch.Put(indexKey, true); err != nil {
			return err
		}
	}

	if err := batch.Put(executionCheckerIndexKey(exec.CheckerID, exec.Target, exec.StartedAt, exec.Id.String()), true); err != nil {
		return err
	}

	if exec.Target.UserId != "" {
		if err := batch.Put(executionUserIndexKey(exec.Target.UserId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	if exec.Target.DomainId != "" {
		if err := batch.Put(executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	return batch.Commit()
}

func (s *KVStorage) UpdateExecution(exec *happydns.Execution) error {
	// Load the old record so we can detect changed index keys.
	old, err := s.GetExecution(exec.Id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	if err := batch.Put(fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, exec.Id.String()), exec); err != nil {
		return err
	}

	// Compute new plan index key (if any) once for reuse.
	newPlanKey := ""
	if exec.PlanID != nil {
		newPlanKey = fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, exec.PlanID.String(), exec.Id.String())
	}

	// Clean up stale plan index if PlanID changed.
	if old.PlanID != nil {
		oldPlanKey := fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, old.PlanID.String(), exec.Id.String())
		if oldPlanKey != newPlanKey {
			batch.Delete(oldPlanKey)
		}
	}

	// Update secondary index by plan if applicable.
	if exec.PlanID != nil {
		if err := batch.Put(newPlanKey, true); err != nil {
			return err
		}
	}

	// Clean up stale checker+target index if CheckerID, Target or StartedAt
	// changed (StartedAt feeds the reverseChronoSegment in the key).
	oldCheckerKey := executionCheckerIndexKey(old.CheckerID, old.Target, old.StartedAt, exec.Id.String())
	newCheckerKey := executionCheckerIndexKey(exec.CheckerID, exec.Target, exec.StartedAt, exec.Id.String())
	if oldCheckerKey != newCheckerKey {
		batch.Delete(oldCheckerKey)
	}

	// Update secondary index by checker+target.
	if err := batch.Put(newCheckerKey, true); err != nil {
		return err
	}

	// Clean up stale user index if UserId or StartedAt changed.
	if old.Target.UserId != "" {
		oldUserKey := executionUserIndexKey(old.Target.UserId, old.StartedAt, exec.Id.String())
		if exec.Target.UserId == "" || oldUserKey != executionUserIndexKey(exec.Target.UserId, exec.StartedAt, exec.Id.String()) {
			batch.Delete(oldUserKey)
		}
	}

	// Update secondary index by user.
	if exec.Target.UserId != "" {
		if err := batch.Put(executionUserIndexKey(exec.Target.UserId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	// Clean up stale domain index if DomainId or StartedAt changed.
	if old.Target.DomainId != "" {
		oldDomainKey := executionDomainIndexKey(old.Target.DomainId, old.StartedAt, exec.Id.String())
		if exec.Target.DomainId == "" || oldDomainKey != executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, exec.Id.String()) {
			batch.Delete(oldDomainKey)
		}
	}

	// Update secondary index by domain.
	if exec.Target.DomainId != "" {
		if err := batch.Put(executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, exec.Id.String()), true); err != nil {
			return err
		}
	}

	return batch.Commit()
}

func (s *KVStorage) DeleteExecution(execID happydns.Identifier) error {
	exec, err := s.GetExecution(execID)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()

	if exec.PlanID != nil {
		batch.Delete(fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, exec.PlanID.String(), execID.String()))
	}

	batch.Delete(executionCheckerIndexKey(exec.CheckerID, exec.Target, exec.StartedAt, execID.String()))

	if exec.Target.UserId != "" {
		batch.Delete(executionUserIndexKey(exec.Target.UserId, exec.StartedAt, execID.String()))
	}

	if exec.Target.DomainId != "" {
		batch.Delete(executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, execID.String()))
	}

	batch.Delete(fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, execID.String()))

	return batch.Commit()
}

func (s *KVStorage) DeleteExecutionsByChecker(checkerID string, target happydns.CheckTarget) error {
	prefix := fmt.Sprintf("%s%s|%s|", ExecutionByCheckerIndexPrefix, checkerID, target.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	for iter.Next() {
		execId, err := lastKeySegment(iter.Key())
		if err != nil {
			continue
		}

		exec, err := s.GetExecution(execId)
		if err != nil {
			// Primary record already gone; just clean up this index entry
			// and attempt to clean up other indexes (best-effort scan).
			if err := s.db.Delete(iter.Key()); err != nil {
				return err
			}
			s.deleteExecSecondaryIndexesByExecID(execId)
			continue
		}

		batch := s.db.NewBatch()

		if exec.PlanID != nil {
			batch.Delete(fmt.Sprintf("%s%s|%s", ExecutionByPlanIndexPrefix, exec.PlanID.String(), exec.Id.String()))
		}

		if exec.Target.UserId != "" {
			batch.Delete(executionUserIndexKey(exec.Target.UserId, exec.StartedAt, exec.Id.String()))
		}

		if exec.Target.DomainId != "" {
			batch.Delete(executionDomainIndexKey(exec.Target.DomainId, exec.StartedAt, exec.Id.String()))
		}

		batch.Delete(fmt.Sprintf("%s%s", ExecutionPrimaryPrefix, exec.Id.String()))
		batch.Delete(iter.Key())

		if err := batch.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// deleteExecSecondaryIndexesByExecID scans plan, user and domain indexes to
// remove any entry for the given execution ID. Used when the primary record is
// already gone and we don't know which plan/user/domain it belonged to.
func (s *KVStorage) deleteExecSecondaryIndexesByExecID(execId happydns.Identifier) {
	suffix := "|" + execId.String()
	for _, prefix := range []string{ExecutionByPlanIndexPrefix, ExecutionByUserIndexPrefix, ExecutionByDomainIndexPrefix} {
		iter := s.db.Search(prefix)
		for iter.Next() {
			if strings.HasSuffix(iter.Key(), suffix) {
				if err := s.db.Delete(iter.Key()); err != nil {
					log.Printf("deleteExecSecondaryIndexesByExecID: failed to delete %s: %v\n", iter.Key(), err)
				}
			}
		}
		iter.Release()
	}
}

func (s *KVStorage) execExists(id happydns.Identifier) bool {
	_, err := s.GetExecution(id)
	return err == nil
}

func (s *KVStorage) TidyExecutionIndexes() error {
	// Tidy chckexec-plan|{planId}|{execId} indexes.
	s.tidyTwoPartIndex(ExecutionByPlanIndexPrefix, "execution plan", func(id happydns.Identifier) bool {
		_, err := s.GetCheckPlan(id)
		return err == nil
	}, s.execExists)

	// Tidy chckexec-chkr|{checkerID}|{target}|{revTime}|{execId} indexes.
	s.tidyLastSegmentIndex(ExecutionByCheckerIndexPrefix, "execution checker", s.execExists)

	// Tidy chckexec-user|{userId}|{revTime}|{execId} indexes.
	s.tidyOwnerTimeIndex(ExecutionByUserIndexPrefix, "execution user", func(id happydns.Identifier) bool {
		_, err := s.GetUser(id)
		return err == nil
	}, s.execExists)

	// Tidy chckexec-domain|{domainId}|{revTime}|{execId} indexes.
	s.tidyOwnerTimeIndex(ExecutionByDomainIndexPrefix, "execution domain", func(id happydns.Identifier) bool {
		_, err := s.GetDomain(id)
		return err == nil
	}, s.execExists)

	return nil
}

func (s *KVStorage) ClearExecutions() error {
	// Delete secondary indexes (chckexec-plan|..., chckexec-chkr|..., chckexec-user|..., chckexec-domain|...).
	if err := s.clearByPrefix("chckexec-"); err != nil {
		return err
	}

	// Delete primary records (chckexec|...).
	iter, err := s.ListAllExecutions()
	if err != nil {
		return err
	}
	defer iter.Close()
	for iter.Next() {
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}
	return nil
}
