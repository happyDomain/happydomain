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

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/model"
)

const (
	checkPlanPrimaryPrefix        = "chckpln|"
	checkPlanByCheckerIndexPrefix = "chckpln-chkr|"
	checkPlanByTargetIndexPrefix  = "chckpln-tgt|"
	checkPlanByUserIndexPrefix    = "chckpln-user|"
)

func planTargetIndexKey(target happydns.CheckTarget, planId string) string {
	return fmt.Sprintf("%s%s|%s", checkPlanByTargetIndexPrefix, target.String(), planId)
}

func planCheckerIndexKey(checkerID string, planId string) string {
	return fmt.Sprintf("%s%s|%s", checkPlanByCheckerIndexPrefix, checkerID, planId)
}

func planUserIndexKey(userId string, planId string) string {
	return fmt.Sprintf("%s%s|%s", checkPlanByUserIndexPrefix, userId, planId)
}

func (s *KVStorage) ListAllCheckPlans() (happydns.Iterator[happydns.CheckPlan], error) {
	iter := s.db.Search(checkPlanPrimaryPrefix)
	return NewKVIterator[happydns.CheckPlan](s.db, iter), nil
}

func (s *KVStorage) ListCheckPlansByTarget(target happydns.CheckTarget) ([]*happydns.CheckPlan, error) {
	return listByIndex(s, fmt.Sprintf("%s%s|", checkPlanByTargetIndexPrefix, target.String()), s.GetCheckPlan)
}

func (s *KVStorage) ListCheckPlansByChecker(checkerID string) ([]*happydns.CheckPlan, error) {
	return listByIndex(s, fmt.Sprintf("%s%s|", checkPlanByCheckerIndexPrefix, checkerID), s.GetCheckPlan)
}

func (s *KVStorage) ListCheckPlansByUser(userId happydns.Identifier) ([]*happydns.CheckPlan, error) {
	return listByIndex(s, fmt.Sprintf("%s%s|", checkPlanByUserIndexPrefix, userId.String()), s.GetCheckPlan)
}

func (s *KVStorage) GetCheckPlan(planID happydns.Identifier) (*happydns.CheckPlan, error) {
	plan := &happydns.CheckPlan{}
	err := s.db.Get(fmt.Sprintf("%s%s", checkPlanPrimaryPrefix, planID.String()), plan)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrCheckPlanNotFound
	}
	return plan, err
}

func (s *KVStorage) CreateCheckPlan(plan *happydns.CheckPlan) error {
	key, id, err := s.db.FindIdentifierKey(checkPlanPrimaryPrefix)
	if err != nil {
		return err
	}
	plan.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, plan); err != nil {
		return err
	}
	if err := stageCheckPlanIndexes(batch, plan); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) UpdateCheckPlan(plan *happydns.CheckPlan) error {
	old, err := s.GetCheckPlan(plan.Id)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	if err := batch.Put(fmt.Sprintf("%s%s", checkPlanPrimaryPrefix, plan.Id.String()), plan); err != nil {
		return err
	}

	// Clean up stale target index if target changed.
	oldTargetKey := planTargetIndexKey(old.Target, old.Id.String())
	newTargetKey := planTargetIndexKey(plan.Target, plan.Id.String())
	if oldTargetKey != newTargetKey {
		batch.Delete(oldTargetKey)
	}

	// Clean up stale checker index if checker changed.
	oldCheckerKey := planCheckerIndexKey(old.CheckerID, old.Id.String())
	newCheckerKey := planCheckerIndexKey(plan.CheckerID, plan.Id.String())
	if oldCheckerKey != newCheckerKey {
		batch.Delete(oldCheckerKey)
	}

	// Clean up stale user index if user changed.
	if old.Target.UserId != "" && old.Target.UserId != plan.Target.UserId {
		batch.Delete(planUserIndexKey(old.Target.UserId, old.Id.String()))
	}

	if err := stageCheckPlanIndexes(batch, plan); err != nil {
		return err
	}
	return batch.Commit()
}

// stageCheckPlanIndexes stages the target/checker/user index puts for a plan
// onto the given batch. Caller is responsible for committing.
func stageCheckPlanIndexes(batch storage.Batch, plan *happydns.CheckPlan) error {
	if err := batch.Put(planTargetIndexKey(plan.Target, plan.Id.String()), true); err != nil {
		return err
	}

	if err := batch.Put(planCheckerIndexKey(plan.CheckerID, plan.Id.String()), true); err != nil {
		return err
	}

	if plan.Target.UserId != "" {
		if err := batch.Put(planUserIndexKey(plan.Target.UserId, plan.Id.String()), true); err != nil {
			return err
		}
	}

	return nil
}

// RestoreCheckPlan writes a plan at its existing Id and (re)builds its
// secondary indexes. Used by the backup restore path, which must preserve
// the original identifier instead of generating a new one.
func (s *KVStorage) RestoreCheckPlan(plan *happydns.CheckPlan) error {
	batch := s.db.NewBatch()
	if err := batch.Put(fmt.Sprintf("%s%s", checkPlanPrimaryPrefix, plan.Id.String()), plan); err != nil {
		return err
	}
	if err := stageCheckPlanIndexes(batch, plan); err != nil {
		return err
	}
	return batch.Commit()
}

func (s *KVStorage) DeleteCheckPlan(planID happydns.Identifier) error {
	plan, err := s.GetCheckPlan(planID)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	batch.Delete(planTargetIndexKey(plan.Target, planID.String()))
	batch.Delete(planCheckerIndexKey(plan.CheckerID, planID.String()))

	if plan.Target.UserId != "" {
		batch.Delete(planUserIndexKey(plan.Target.UserId, planID.String()))
	}

	batch.Delete(fmt.Sprintf("%s%s", checkPlanPrimaryPrefix, planID.String()))

	return batch.Commit()
}

// deleteCheckPlanSecondaryIndexesByPlanID scans all plan index prefixes to
// remove any entry for the given plan ID. Used when the primary record is
// already gone and we don't know which target/checker/user it belonged to.
func (s *KVStorage) deleteCheckPlanSecondaryIndexesByPlanID(planId happydns.Identifier) {
	suffix := "|" + planId.String()
	for _, prefix := range []string{checkPlanByTargetIndexPrefix, checkPlanByCheckerIndexPrefix, checkPlanByUserIndexPrefix} {
		iter := s.db.Search(prefix)
		for iter.Next() {
			if strings.HasSuffix(iter.Key(), suffix) {
				if err := s.db.Delete(iter.Key()); err != nil {
					log.Printf("deleteCheckPlanSecondaryIndexesByPlanID: failed to delete %s: %v\n", iter.Key(), err)
				}
			}
		}
		iter.Release()
	}
}

func (s *KVStorage) checkPlanExists(id happydns.Identifier) bool {
	_, err := s.GetCheckPlan(id)
	return err == nil
}

func (s *KVStorage) TidyCheckPlanIndexes() error {
	// Tidy chckpln-tgt|{target}|{planId} indexes.
	s.tidyLastSegmentIndex(checkPlanByTargetIndexPrefix, "plan target", s.checkPlanExists)

	// Tidy chckpln-chkr|{checkerID}|{planId} indexes.
	s.tidyLastSegmentIndex(checkPlanByCheckerIndexPrefix, "plan checker", s.checkPlanExists)

	// Tidy chckpln-user|{userId}|{planId} indexes.
	s.tidyTwoPartIndex(checkPlanByUserIndexPrefix, "plan user", func(id happydns.Identifier) bool {
		_, err := s.GetUser(id)
		return err == nil
	}, s.checkPlanExists)

	return nil
}

func (s *KVStorage) ClearCheckPlans() error {
	// Delete secondary indexes.
	if err := s.clearByPrefix("chckpln-"); err != nil {
		return err
	}

	// Delete primary records.
	iter, err := s.ListAllCheckPlans()
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
