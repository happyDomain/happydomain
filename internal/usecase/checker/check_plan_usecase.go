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
	"fmt"

	checkerPkg "git.happydns.org/happyDomain/internal/checker"
	"git.happydns.org/happyDomain/model"
)

// targetMatchesResource verifies that every non-empty field in scope
// matches the corresponding field in resource. Returns false if any
// scope-specified field does not match, indicating the resource belongs
// to a different user/domain/service than the caller's scope.
func targetMatchesResource(scope, resource happydns.CheckTarget) bool {
	if scope.UserId != "" && scope.UserId != resource.UserId {
		return false
	}
	if scope.DomainId != "" && scope.DomainId != resource.DomainId {
		return false
	}
	if scope.ServiceId != "" && scope.ServiceId != resource.ServiceId {
		return false
	}
	return true
}

// CheckPlanUsecase handles business logic for check plans.
type CheckPlanUsecase struct {
	store CheckPlanStorage
}

// NewCheckPlanUsecase creates a new CheckPlanUsecase.
func NewCheckPlanUsecase(store CheckPlanStorage) *CheckPlanUsecase {
	return &CheckPlanUsecase{store: store}
}

// ListCheckPlansByTarget returns all check plans matching the given target.
func (u *CheckPlanUsecase) ListCheckPlansByTarget(target happydns.CheckTarget) ([]*happydns.CheckPlan, error) {
	return u.store.ListCheckPlansByTarget(target)
}

// CreateCheckPlan validates that the checker exists and persists the plan.
func (u *CheckPlanUsecase) CreateCheckPlan(plan *happydns.CheckPlan) error {
	if checkerPkg.FindChecker(plan.CheckerID) == nil {
		return fmt.Errorf("checker %q not found", plan.CheckerID)
	}
	return u.store.CreateCheckPlan(plan)
}

// GetCheckPlan retrieves a check plan by ID and verifies it belongs to the given scope.
func (u *CheckPlanUsecase) GetCheckPlan(scope happydns.CheckTarget, planID happydns.Identifier) (*happydns.CheckPlan, error) {
	plan, err := u.store.GetCheckPlan(planID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, plan.Target) {
		return nil, happydns.ErrCheckPlanNotFound
	}
	return plan, nil
}

// UpdateCheckPlan fetches the existing plan, verifies scope ownership,
// validates the checker exists, preserves Id and Target (immutable),
// and persists the merged result.
func (u *CheckPlanUsecase) UpdateCheckPlan(scope happydns.CheckTarget, planID happydns.Identifier, updated *happydns.CheckPlan) (*happydns.CheckPlan, error) {
	existing, err := u.store.GetCheckPlan(planID)
	if err != nil {
		return nil, err
	}
	if !targetMatchesResource(scope, existing.Target) {
		return nil, happydns.ErrCheckPlanNotFound
	}

	if checkerPkg.FindChecker(updated.CheckerID) == nil {
		return nil, fmt.Errorf("checker %q not found", updated.CheckerID)
	}

	updated.Id = existing.Id
	updated.Target = existing.Target

	if err := u.store.UpdateCheckPlan(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteCheckPlan deletes a check plan by ID after verifying scope ownership.
func (u *CheckPlanUsecase) DeleteCheckPlan(scope happydns.CheckTarget, planID happydns.Identifier) error {
	plan, err := u.store.GetCheckPlan(planID)
	if err != nil {
		return err
	}
	if !targetMatchesResource(scope, plan.Target) {
		return happydns.ErrCheckPlanNotFound
	}
	return u.store.DeleteCheckPlan(planID)
}
