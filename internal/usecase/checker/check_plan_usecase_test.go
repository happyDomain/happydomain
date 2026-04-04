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

package checker_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/checker"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

func setupPlanUC(t *testing.T) (*checkerUC.CheckPlanUsecase, *planStore) {
	t.Helper()
	// Register a checker so CreateCheckPlan validation passes.
	checker.RegisterChecker(&happydns.CheckerDefinition{
		ID:   "plan_test_checker",
		Name: "Plan Test Checker",
		Availability: happydns.CheckerAvailability{
			ApplyToDomain: true,
		},
		Rules: []happydns.CheckRule{
			&testCheckRule{name: "rule_a", status: happydns.StatusOK},
		},
	})

	store := newPlanStore()
	uc := checkerUC.NewCheckPlanUsecase(store)
	return uc, store
}

func TestCheckPlanUsecase_CreateAndGet(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}

	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	if plan.Id.IsEmpty() {
		t.Fatal("expected plan to get an ID assigned")
	}

	got, err := uc.GetCheckPlan(target, plan.Id)
	if err != nil {
		t.Fatalf("GetCheckPlan() error: %v", err)
	}
	if got.CheckerID != "plan_test_checker" {
		t.Errorf("expected CheckerID plan_test_checker, got %s", got.CheckerID)
	}
}

func TestCheckPlanUsecase_CreateUnknownChecker(t *testing.T) {
	uc, _ := setupPlanUC(t)

	plan := &happydns.CheckPlan{
		CheckerID: "nonexistent_checker",
	}

	if err := uc.CreateCheckPlan(plan); err == nil {
		t.Fatal("expected error for unknown checker")
	}
}

func TestCheckPlanUsecase_ListByTarget(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	plans, err := uc.ListCheckPlansByTarget(target)
	if err != nil {
		t.Fatalf("ListCheckPlansByTarget() error: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}

	// Different target should return empty.
	uid2, _ := happydns.NewRandomIdentifier()
	other := happydns.CheckTarget{UserId: uid2.String()}
	plans2, err := uc.ListCheckPlansByTarget(other)
	if err != nil {
		t.Fatalf("ListCheckPlansByTarget() error: %v", err)
	}
	if len(plans2) != 0 {
		t.Errorf("expected 0 plans for different target, got %d", len(plans2))
	}
}

func TestCheckPlanUsecase_ListByTargetAndChecker(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	// Create a plan for plan_test_checker.
	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	// Query for the matching checker - should return the plan.
	plans, err := uc.ListCheckPlansByTargetAndChecker(target, "plan_test_checker")
	if err != nil {
		t.Fatalf("ListCheckPlansByTargetAndChecker() error: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}

	// Query for a different checker on the same target - should return nothing.
	plans2, err := uc.ListCheckPlansByTargetAndChecker(target, "other_checker")
	if err != nil {
		t.Fatalf("ListCheckPlansByTargetAndChecker() error: %v", err)
	}
	if len(plans2) != 0 {
		t.Errorf("expected 0 plans for different checker, got %d", len(plans2))
	}
}

func TestCheckPlanUsecase_UpdatePreservesIdAndTarget(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	did, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: did.String()}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	origID := plan.Id

	// Update with different target and ID; they should be preserved.
	uid2, _ := happydns.NewRandomIdentifier()
	fakeID, _ := happydns.NewRandomIdentifier()
	updated := &happydns.CheckPlan{
		Id:        fakeID,
		CheckerID: "plan_test_checker",
		Target:    happydns.CheckTarget{UserId: uid2.String()},
		Enabled:   map[string]bool{"rule_a": false},
	}

	result, err := uc.UpdateCheckPlan(target, origID, updated)
	if err != nil {
		t.Fatalf("UpdateCheckPlan() error: %v", err)
	}

	if !result.Id.Equals(origID) {
		t.Errorf("expected Id to be preserved as %s, got %s", origID, result.Id)
	}
	if result.Target.String() != target.String() {
		t.Errorf("expected Target to be preserved")
	}
	if result.Enabled["rule_a"] != false {
		t.Errorf("expected Enabled to be updated")
	}
}

func TestCheckPlanUsecase_UpdateScopeMismatch(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	// Update with a different user scope should fail.
	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	_, err := uc.UpdateCheckPlan(wrongScope, plan.Id, &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Enabled:   map[string]bool{"rule_a": false},
	})
	if err == nil {
		t.Fatal("expected error when scope doesn't match plan target")
	}

	// Verify the original plan is unchanged.
	got, err := uc.GetCheckPlan(target, plan.Id)
	if err != nil {
		t.Fatalf("GetCheckPlan() error: %v", err)
	}
	if got.Enabled != nil {
		t.Errorf("expected original plan to be unchanged, got Enabled=%v", got.Enabled)
	}
}

func TestCheckPlanUsecase_GetScopeMismatch(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	// Get with a different user scope should fail.
	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	_, err := uc.GetCheckPlan(wrongScope, plan.Id)
	if err == nil {
		t.Fatal("expected error when scope doesn't match plan target")
	}
}

func TestCheckPlanUsecase_DeleteScopeMismatch(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	uid2, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String(), DomainId: "d1"}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	// Delete with a different user scope should fail.
	wrongScope := happydns.CheckTarget{UserId: uid2.String()}
	if err := uc.DeleteCheckPlan(wrongScope, plan.Id); err == nil {
		t.Fatal("expected error when scope doesn't match plan target")
	}

	// Verify the plan still exists.
	_, err := uc.GetCheckPlan(target, plan.Id)
	if err != nil {
		t.Fatalf("plan should still exist after failed delete: %v", err)
	}
}

func TestCheckPlanUsecase_UpdateNotFound(t *testing.T) {
	uc, _ := setupPlanUC(t)

	fakeID, _ := happydns.NewRandomIdentifier()
	_, err := uc.UpdateCheckPlan(happydns.CheckTarget{}, fakeID, &happydns.CheckPlan{})
	if err == nil {
		t.Fatal("expected error for nonexistent plan")
	}
}

func TestCheckPlanUsecase_Delete(t *testing.T) {
	uc, _ := setupPlanUC(t)

	uid, _ := happydns.NewRandomIdentifier()
	target := happydns.CheckTarget{UserId: uid.String()}

	plan := &happydns.CheckPlan{
		CheckerID: "plan_test_checker",
		Target:    target,
	}
	if err := uc.CreateCheckPlan(plan); err != nil {
		t.Fatalf("CreateCheckPlan() error: %v", err)
	}

	if err := uc.DeleteCheckPlan(target, plan.Id); err != nil {
		t.Fatalf("DeleteCheckPlan() error: %v", err)
	}

	_, err := uc.GetCheckPlan(target, plan.Id)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

// --- planStore: minimal in-memory CheckPlanStorage ---

type planStore struct {
	plans map[string]*happydns.CheckPlan
}

func newPlanStore() *planStore {
	return &planStore{plans: make(map[string]*happydns.CheckPlan)}
}

func (s *planStore) ListAllCheckPlans() (happydns.Iterator[happydns.CheckPlan], error) {
	return nil, nil
}

func (s *planStore) ListCheckPlansByTarget(target happydns.CheckTarget) ([]*happydns.CheckPlan, error) {
	var result []*happydns.CheckPlan
	for _, p := range s.plans {
		if p.Target.String() == target.String() {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *planStore) ListCheckPlansByChecker(checkerID string) ([]*happydns.CheckPlan, error) {
	return nil, nil
}

func (s *planStore) ListCheckPlansByUser(userId happydns.Identifier) ([]*happydns.CheckPlan, error) {
	return nil, nil
}

func (s *planStore) GetCheckPlan(planID happydns.Identifier) (*happydns.CheckPlan, error) {
	p, ok := s.plans[planID.String()]
	if !ok {
		return nil, happydns.ErrCheckPlanNotFound
	}
	return p, nil
}

func (s *planStore) CreateCheckPlan(plan *happydns.CheckPlan) error {
	id, _ := happydns.NewRandomIdentifier()
	plan.Id = id
	s.plans[plan.Id.String()] = plan
	return nil
}

func (s *planStore) UpdateCheckPlan(plan *happydns.CheckPlan) error {
	s.plans[plan.Id.String()] = plan
	return nil
}

func (s *planStore) DeleteCheckPlan(planID happydns.Identifier) error {
	delete(s.plans, planID.String())
	return nil
}

func (s *planStore) TidyCheckPlanIndexes() error { return nil }

func (s *planStore) ClearCheckPlans() error {
	s.plans = make(map[string]*happydns.CheckPlan)
	return nil
}
