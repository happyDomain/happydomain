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

const (
	evaluationPrimaryPrefix        = "chckeval|"
	evaluationByPlanIndexPrefix    = "chckeval-plan|"
	evaluationByCheckerIndexPrefix = "chckeval-chkr|"
)

func (s *KVStorage) ListEvaluationsByPlan(planID happydns.Identifier) ([]*happydns.CheckEvaluation, error) {
	return listByIndex(s, fmt.Sprintf("%s%s|", evaluationByPlanIndexPrefix, planID.String()), s.GetEvaluation)
}

func (s *KVStorage) ListAllEvaluations() (happydns.Iterator[happydns.CheckEvaluation], error) {
	iter := s.db.Search(evaluationPrimaryPrefix)
	return NewKVIterator[happydns.CheckEvaluation](s.db, iter), nil
}

func (s *KVStorage) GetEvaluation(evalID happydns.Identifier) (*happydns.CheckEvaluation, error) {
	eval := &happydns.CheckEvaluation{}
	err := s.db.Get(fmt.Sprintf("%s%s", evaluationPrimaryPrefix, evalID.String()), eval)
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
		fmt.Sprintf("%s%s|%s|", evaluationByCheckerIndexPrefix, checkerID, target.String()),
		s.GetEvaluation,
		func(a, b *happydns.CheckEvaluation) bool { return a.EvaluatedAt.After(b.EvaluatedAt) },
		limit,
		nil,
	)
}

func (s *KVStorage) CreateEvaluation(eval *happydns.CheckEvaluation) error {
	key, id, err := s.db.FindIdentifierKey(evaluationPrimaryPrefix)
	if err != nil {
		return err
	}
	eval.Id = id

	batch := s.db.NewBatch()
	if err := batch.Put(key, eval); err != nil {
		return err
	}

	if eval.PlanID != nil {
		indexKey := fmt.Sprintf("%s%s|%s", evaluationByPlanIndexPrefix, eval.PlanID.String(), eval.Id.String())
		if err := batch.Put(indexKey, true); err != nil {
			return err
		}
	}

	checkerIndexKey := fmt.Sprintf("%s%s|%s|%s", evaluationByCheckerIndexPrefix, eval.CheckerID, eval.Target.String(), eval.Id.String())
	if err := batch.Put(checkerIndexKey, true); err != nil {
		return err
	}

	return batch.Commit()
}

// RestoreEvaluation writes an evaluation at its existing Id and rebuilds
// its secondary indexes. Used by the backup restore path.
func (s *KVStorage) RestoreEvaluation(eval *happydns.CheckEvaluation) error {
	batch := s.db.NewBatch()
	if err := batch.Put(fmt.Sprintf("%s%s", evaluationPrimaryPrefix, eval.Id.String()), eval); err != nil {
		return err
	}

	if eval.PlanID != nil {
		indexKey := fmt.Sprintf("%s%s|%s", evaluationByPlanIndexPrefix, eval.PlanID.String(), eval.Id.String())
		if err := batch.Put(indexKey, true); err != nil {
			return err
		}
	}

	checkerIndexKey := fmt.Sprintf("%s%s|%s|%s", evaluationByCheckerIndexPrefix, eval.CheckerID, eval.Target.String(), eval.Id.String())
	if err := batch.Put(checkerIndexKey, true); err != nil {
		return err
	}

	return batch.Commit()
}

func (s *KVStorage) DeleteEvaluation(evalID happydns.Identifier) error {
	// Load first to find plan ID for index cleanup.
	eval, err := s.GetEvaluation(evalID)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()

	if eval.PlanID != nil {
		batch.Delete(fmt.Sprintf("%s%s|%s", evaluationByPlanIndexPrefix, eval.PlanID.String(), eval.Id.String()))
	}

	batch.Delete(fmt.Sprintf("%s%s|%s|%s", evaluationByCheckerIndexPrefix, eval.CheckerID, eval.Target.String(), eval.Id.String()))
	batch.Delete(fmt.Sprintf("%s%s", evaluationPrimaryPrefix, evalID.String()))

	return batch.Commit()
}

func (s *KVStorage) DeleteEvaluationsByChecker(checkerID string, target happydns.CheckTarget) error {
	prefix := fmt.Sprintf("%s%s|%s|", evaluationByCheckerIndexPrefix, checkerID, target.String())
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

		batch := s.db.NewBatch()
		if eval.PlanID != nil {
			batch.Delete(fmt.Sprintf("%s%s|%s", evaluationByPlanIndexPrefix, eval.PlanID.String(), eval.Id.String()))
		}
		batch.Delete(fmt.Sprintf("%s%s", evaluationPrimaryPrefix, eval.Id.String()))
		batch.Delete(iter.Key())

		if err := batch.Commit(); err != nil {
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
	iter := s.db.Search(evaluationByPlanIndexPrefix)
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
	s.tidyTwoPartIndex(evaluationByPlanIndexPrefix, "evaluation plan", func(id happydns.Identifier) bool {
		_, err := s.GetCheckPlan(id)
		return err == nil
	}, s.evalExists)

	// Tidy chckeval-chkr|{checkerID}|{target}|{evalId} indexes.
	s.tidyLastSegmentIndex(evaluationByCheckerIndexPrefix, "evaluation checker", s.evalExists)

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
