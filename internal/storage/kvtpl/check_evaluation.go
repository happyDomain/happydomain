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

func (s *KVStorage) ListEvaluationsByPlan(planID happydns.Identifier) ([]*happydns.CheckEvaluation, error) {
	return listByIndex(s, fmt.Sprintf("chckeval-plan|%s|", planID.String()), s.GetEvaluation)
}

func (s *KVStorage) ListAllEvaluations() (happydns.Iterator[happydns.CheckEvaluation], error) {
	iter := s.db.Search("chckeval|")
	return NewKVIterator[happydns.CheckEvaluation](s.db, iter), nil
}

func (s *KVStorage) GetEvaluation(evalID happydns.Identifier) (*happydns.CheckEvaluation, error) {
	eval := &happydns.CheckEvaluation{}
	err := s.db.Get(fmt.Sprintf("chckeval|%s", evalID.String()), eval)
	if errors.Is(err, happydns.ErrNotFound) {
		return nil, happydns.ErrCheckEvaluationNotFound
	}
	return eval, err
}

func (s *KVStorage) GetLatestEvaluation(planID happydns.Identifier) (*happydns.CheckEvaluation, error) {
	evals, err := s.ListEvaluationsByPlan(planID)
	if err != nil {
		return nil, err
	}
	if len(evals) == 0 {
		return nil, happydns.ErrCheckEvaluationNotFound
	}

	latest := evals[0]
	for _, e := range evals[1:] {
		if e.EvaluatedAt.After(latest.EvaluatedAt) {
			latest = e
		}
	}
	return latest, nil
}

func (s *KVStorage) ListEvaluationsByChecker(checkerID string, target happydns.CheckTarget, limit int) ([]*happydns.CheckEvaluation, error) {
	return listByIndexSorted(
		s,
		fmt.Sprintf("chckeval-chkr|%s|%s|", checkerID, target.String()),
		s.GetEvaluation,
		func(a, b *happydns.CheckEvaluation) bool { return a.EvaluatedAt.After(b.EvaluatedAt) },
		limit,
	)
}

func (s *KVStorage) CreateEvaluation(eval *happydns.CheckEvaluation) error {
	key, id, err := s.db.FindIdentifierKey("chckeval|")
	if err != nil {
		return err
	}
	eval.Id = id

	// Store the primary record.
	if err := s.db.Put(key, eval); err != nil {
		return err
	}

	// Store secondary index by plan if applicable.
	if eval.PlanID != nil {
		indexKey := fmt.Sprintf("chckeval-plan|%s|%s", eval.PlanID.String(), eval.Id.String())
		if err := s.db.Put(indexKey, true); err != nil {
			return err
		}
	}

	// Store secondary index by checker+target.
	checkerIndexKey := fmt.Sprintf("chckeval-chkr|%s|%s|%s", eval.CheckerID, eval.Target.String(), eval.Id.String())
	if err := s.db.Put(checkerIndexKey, true); err != nil {
		return err
	}

	return nil
}

func (s *KVStorage) DeleteEvaluation(evalID happydns.Identifier) error {
	// Load first to find plan ID for index cleanup.
	eval, err := s.GetEvaluation(evalID)
	if err != nil {
		return err
	}

	if eval.PlanID != nil {
		indexKey := fmt.Sprintf("chckeval-plan|%s|%s", eval.PlanID.String(), eval.Id.String())
		if err := s.db.Delete(indexKey); err != nil {
			log.Printf("DeleteEvaluation: failed to delete plan index %s: %v\n", indexKey, err)
		}
	}

	// Clean up checker+target index.
	checkerIndexKey := fmt.Sprintf("chckeval-chkr|%s|%s|%s", eval.CheckerID, eval.Target.String(), eval.Id.String())
	if err := s.db.Delete(checkerIndexKey); err != nil {
		log.Printf("DeleteEvaluation: failed to delete checker index %s: %v\n", checkerIndexKey, err)
	}

	return s.db.Delete(fmt.Sprintf("chckeval|%s", evalID.String()))
}

func (s *KVStorage) DeleteEvaluationsByChecker(checkerID string, target happydns.CheckTarget) error {
	prefix := fmt.Sprintf("chckeval-chkr|%s|%s|", checkerID, target.String())
	iter := s.db.Search(prefix)
	defer iter.Release()

	for iter.Next() {
		evalId, err := lastKeySegment(iter.Key())
		if err != nil {
			continue
		}

		eval, err := s.GetEvaluation(evalId)
		if err != nil {
			// Primary record already gone; just clean up this index entry
			// and attempt to clean up the plan index (best-effort scan).
			if err := s.db.Delete(iter.Key()); err != nil {
				return err
			}
			s.deleteEvalPlanIndexByEvalID(evalId)
			continue
		}

		// Delete plan index if applicable.
		if eval.PlanID != nil {
			planIndexKey := fmt.Sprintf("chckeval-plan|%s|%s", eval.PlanID.String(), eval.Id.String())
			if err := s.db.Delete(planIndexKey); err != nil {
				log.Printf("DeleteEvaluationsByChecker: failed to delete plan index %s: %v\n", planIndexKey, err)
			}
		}

		// Delete primary record.
		if err := s.db.Delete(fmt.Sprintf("chckeval|%s", eval.Id.String())); err != nil {
			log.Printf("DeleteEvaluationsByChecker: failed to delete primary record %s: %v\n", eval.Id.String(), err)
		}

		// Delete this checker index entry.
		if err := s.db.Delete(iter.Key()); err != nil {
			return err
		}
	}
	return nil
}

// deleteEvalPlanIndexByEvalID scans plan indexes to remove any entry for the
// given evaluation ID. Used when the primary record is already gone and we
// don't know which plan it belonged to.
func (s *KVStorage) deleteEvalPlanIndexByEvalID(evalId happydns.Identifier) {
	suffix := "|" + evalId.String()
	iter := s.db.Search("chckeval-plan|")
	defer iter.Release()
	for iter.Next() {
		if strings.HasSuffix(iter.Key(), suffix) {
			if err := s.db.Delete(iter.Key()); err != nil {
				log.Printf("deleteEvalPlanIndexByEvalID: failed to delete %s: %v\n", iter.Key(), err)
			}
		}
	}
}

func (s *KVStorage) evalExists(id happydns.Identifier) bool {
	_, err := s.GetEvaluation(id)
	return err == nil
}

func (s *KVStorage) TidyEvaluationIndexes() error {
	// Tidy chckeval-plan|{planId}|{evalId} indexes.
	s.tidyTwoPartIndex("chckeval-plan|", "evaluation plan", func(id happydns.Identifier) bool {
		_, err := s.GetCheckPlan(id)
		return err == nil
	}, s.evalExists)

	// Tidy chckeval-chkr|{checkerID}|{target}|{evalId} indexes.
	s.tidyLastSegmentIndex("chckeval-chkr|", "evaluation checker", s.evalExists)

	return nil
}

func (s *KVStorage) ClearEvaluations() error {
	// Delete secondary indexes (chckeval-plan|..., chckeval-chkr|...).
	if err := s.clearByPrefix("chckeval-"); err != nil {
		return err
	}

	// Delete primary records (chckeval|...).
	iter, err := s.ListAllEvaluations()
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
