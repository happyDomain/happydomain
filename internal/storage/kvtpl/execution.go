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

	"git.happydns.org/happyDomain/model"
)

func (s *KVStorage) ListExecutionsByPlan(planID happydns.Identifier) ([]*happydns.Execution, error) {
	return listByIndex(s, fmt.Sprintf("chckexec-plan|%s|", planID.String()), s.GetExecution)
}

func (s *KVStorage) ListExecutionsByChecker(checkerID string, target happydns.CheckTarget, limit int) ([]*happydns.Execution, error) {
	return listByIndexSorted(
		s,
		fmt.Sprintf("chckexec-chkr|%s|%s|", checkerID, target.String()),
		s.GetExecution,
		func(a, b *happydns.Execution) bool { return a.StartedAt.After(b.StartedAt) },
		limit,
	)
}

func (s *KVStorage) ListAllExecutions() (happydns.Iterator[happydns.Execution], error) {
	iter := s.db.Search("chckexec|")
	return NewKVIterator[happydns.Execution](s.db, iter), nil
}

func (s *KVStorage) GetExecution(execID happydns.Identifier) (*happydns.Execution, error) {
	exec := &happydns.Execution{}
	err := s.db.Get(fmt.Sprintf("chckexec|%s", execID.String()), exec)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrExecutionNotFound
	}
	return exec, err
}

func (s *KVStorage) CreateExecution(exec *happydns.Execution) error {
	key, id, err := s.db.FindIdentifierKey("chckexec|")
	if err != nil {
		return err
	}
	exec.Id = id

	if err := s.db.Put(key, exec); err != nil {
		return err
	}

	// Secondary index by plan.
	if exec.PlanID != nil {
		indexKey := fmt.Sprintf("chckexec-plan|%s|%s", exec.PlanID.String(), exec.Id.String())
		if err := s.db.Put(indexKey, true); err != nil {
			return err
		}
	}

	// Secondary index by checker+target.
	checkerIndexKey := fmt.Sprintf("chckexec-chkr|%s|%s|%s", exec.CheckerID, exec.Target.String(), exec.Id.String())
	if err := s.db.Put(checkerIndexKey, true); err != nil {
		return err
	}

	return nil
}

func (s *KVStorage) UpdateExecution(exec *happydns.Execution) error {
	// Load the old record so we can detect changed index keys.
	old, err := s.GetExecution(exec.Id)
	if err != nil {
		return err
	}

	if err := s.db.Put(fmt.Sprintf("chckexec|%s", exec.Id.String()), exec); err != nil {
		return err
	}

	// Clean up stale plan index if PlanID changed.
	if old.PlanID != nil {
		oldPlanKey := fmt.Sprintf("chckexec-plan|%s|%s", old.PlanID.String(), exec.Id.String())
		newPlanKey := ""
		if exec.PlanID != nil {
			newPlanKey = fmt.Sprintf("chckexec-plan|%s|%s", exec.PlanID.String(), exec.Id.String())
		}
		if oldPlanKey != newPlanKey {
			if err := s.db.Delete(oldPlanKey); err != nil {
				log.Printf("UpdateExecution: failed to delete stale plan index %s: %v\n", oldPlanKey, err)
			}
		}
	}

	// Update secondary index by plan if applicable.
	if exec.PlanID != nil {
		indexKey := fmt.Sprintf("chckexec-plan|%s|%s", exec.PlanID.String(), exec.Id.String())
		if err := s.db.Put(indexKey, true); err != nil {
			return err
		}
	}

	// Clean up stale checker+target index if CheckerID or Target changed.
	oldCheckerKey := fmt.Sprintf("chckexec-chkr|%s|%s|%s", old.CheckerID, old.Target.String(), exec.Id.String())
	newCheckerKey := fmt.Sprintf("chckexec-chkr|%s|%s|%s", exec.CheckerID, exec.Target.String(), exec.Id.String())
	if oldCheckerKey != newCheckerKey {
		if err := s.db.Delete(oldCheckerKey); err != nil {
			log.Printf("UpdateExecution: failed to delete stale checker index %s: %v\n", oldCheckerKey, err)
		}
	}

	// Update secondary index by checker+target.
	if err := s.db.Put(newCheckerKey, true); err != nil {
		return err
	}

	return nil
}

func (s *KVStorage) DeleteExecution(execID happydns.Identifier) error {
	exec, err := s.GetExecution(execID)
	if err != nil {
		return err
	}

	if exec.PlanID != nil {
		indexKey := fmt.Sprintf("chckexec-plan|%s|%s", exec.PlanID.String(), execID.String())
		if err := s.db.Delete(indexKey); err != nil {
			log.Printf("DeleteExecution: failed to delete plan index %s: %v\n", indexKey, err)
		}
	}

	checkerIndexKey := fmt.Sprintf("chckexec-chkr|%s|%s|%s", exec.CheckerID, exec.Target.String(), execID.String())
	if err := s.db.Delete(checkerIndexKey); err != nil {
		log.Printf("DeleteExecution: failed to delete checker index %s: %v\n", checkerIndexKey, err)
	}

	return s.db.Delete(fmt.Sprintf("chckexec|%s", execID.String()))
}

func (s *KVStorage) DeleteExecutionsByChecker(checkerID string, target happydns.CheckTarget) error {
	prefix := fmt.Sprintf("chckexec-chkr|%s|%s|", checkerID, target.String())
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

		if exec.PlanID != nil {
			planIndexKey := fmt.Sprintf("chckexec-plan|%s|%s", exec.PlanID.String(), exec.Id.String())
			if err := s.db.Delete(planIndexKey); err != nil {
				log.Printf("DeleteExecutionsByChecker: failed to delete plan index %s: %v\n", planIndexKey, err)
			}
		}

		if err := s.db.Delete(fmt.Sprintf("chckexec|%s", exec.Id.String())); err != nil {
			log.Printf("DeleteExecutionsByChecker: failed to delete primary record %s: %v\n", exec.Id.String(), err)
		}

		if err := s.db.Delete(iter.Key()); err != nil {
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
	for _, prefix := range []string{"chckexec-plan|", "chckexec-user|", "chckexec-domain|"} {
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
	s.tidyTwoPartIndex("chckexec-plan|", "execution plan", func(id happydns.Identifier) bool {
		_, err := s.GetCheckPlan(id)
		return err == nil
	}, s.execExists)

	// Tidy chckexec-chkr|{checkerID}|{target}|{execId} indexes.
	s.tidyLastSegmentIndex("chckexec-chkr|", "execution checker", s.execExists)

	return nil
}

func (s *KVStorage) ClearExecutions() error {
	// Delete secondary indexes (chckexec-plan|..., chckexec-chkr|...).
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
