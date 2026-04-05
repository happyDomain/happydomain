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

package checker_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/checker"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// --- helpers ---

func idPtr() *happydns.Identifier {
	id, _ := happydns.NewRandomIdentifier()
	return &id
}

// optionsStore is a minimal in-memory CheckerOptionsStorage that supports
// multi-scope positional lookup and update.
type optionsStore struct {
	// key: "checker|userId|domainId|serviceId"
	data map[string]happydns.CheckerOptions
}

func newOptionsStore() *optionsStore {
	return &optionsStore{data: make(map[string]happydns.CheckerOptions)}
}

func posKey(checkerName string, userId, domainId, serviceId *happydns.Identifier) string {
	f := func(id *happydns.Identifier) string {
		if id == nil {
			return ""
		}
		return id.String()
	}
	return checkerName + "|" + f(userId) + "|" + f(domainId) + "|" + f(serviceId)
}

func (s *optionsStore) ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptionsPositional], error) {
	return nil, nil
}
func (s *optionsStore) ListCheckerConfiguration(checkerName string) ([]*happydns.CheckerOptionsPositional, error) {
	return nil, nil
}

// GetCheckerConfiguration returns positionals from least to most specific.
// It constructs the hierarchy: admin -> user -> domain -> service.
func (s *optionsStore) GetCheckerConfiguration(checkerName string, userId, domainId, serviceId *happydns.Identifier) ([]*happydns.CheckerOptionsPositional, error) {
	var result []*happydns.CheckerOptionsPositional

	// admin level
	if opts, ok := s.data[posKey(checkerName, nil, nil, nil)]; ok {
		result = append(result, &happydns.CheckerOptionsPositional{
			CheckName: checkerName, Options: opts,
		})
	}
	// user level
	if userId != nil {
		if opts, ok := s.data[posKey(checkerName, userId, nil, nil)]; ok {
			result = append(result, &happydns.CheckerOptionsPositional{
				CheckName: checkerName, UserId: userId, Options: opts,
			})
		}
	}
	// domain level
	if domainId != nil {
		if opts, ok := s.data[posKey(checkerName, userId, domainId, nil)]; ok {
			result = append(result, &happydns.CheckerOptionsPositional{
				CheckName: checkerName, UserId: userId, DomainId: domainId, Options: opts,
			})
		}
	}
	// service level
	if serviceId != nil {
		if opts, ok := s.data[posKey(checkerName, userId, domainId, serviceId)]; ok {
			result = append(result, &happydns.CheckerOptionsPositional{
				CheckName: checkerName, UserId: userId, DomainId: domainId, ServiceId: serviceId, Options: opts,
			})
		}
	}

	return result, nil
}

func (s *optionsStore) UpdateCheckerConfiguration(checkerName string, userId, domainId, serviceId *happydns.Identifier, opts happydns.CheckerOptions) error {
	s.data[posKey(checkerName, userId, domainId, serviceId)] = opts
	return nil
}

func (s *optionsStore) DeleteCheckerConfiguration(checkerName string, userId, domainId, serviceId *happydns.Identifier) error {
	delete(s.data, posKey(checkerName, userId, domainId, serviceId))
	return nil
}

func (s *optionsStore) ClearCheckerConfigurations() error {
	s.data = make(map[string]happydns.CheckerOptions)
	return nil
}

// --- test rule/checker types ---

// validatingRule is a CheckRule that also implements OptionsValidator.
type validatingRule struct {
	name        string
	validateErr error
}

func (r *validatingRule) Name() string        { return r.name }
func (r *validatingRule) Description() string { return "validating rule" }
func (r *validatingRule) Evaluate(_ context.Context, _ happydns.ObservationGetter, _ happydns.CheckerOptions) happydns.CheckState {
	return happydns.CheckState{Status: happydns.StatusOK}
}
func (r *validatingRule) ValidateOptions(_ happydns.CheckerOptions) error {
	return r.validateErr
}

// ruleWithOptions is a CheckRule that implements CheckRuleWithOptions.
type ruleWithOptions struct {
	name string
	opts happydns.CheckerOptionsDocumentation
}

func (r *ruleWithOptions) Name() string        { return r.name }
func (r *ruleWithOptions) Description() string { return "rule with options" }
func (r *ruleWithOptions) Evaluate(_ context.Context, _ happydns.ObservationGetter, _ happydns.CheckerOptions) happydns.CheckState {
	return happydns.CheckState{Status: happydns.StatusOK}
}
func (r *ruleWithOptions) Options() happydns.CheckerOptionsDocumentation {
	return r.opts
}

// --- CRUD tests ---

func TestSetAndGetCheckerOptions(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	opts := happydns.CheckerOptions{"key1": "value1", "key2": float64(42)}

	if err := uc.SetCheckerOptions("c1", uid, nil, nil, opts); err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetCheckerOptions("c1", uid, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["key1"] != "value1" || got["key2"] != float64(42) {
		t.Errorf("unexpected options: %v", got)
	}
}

func TestSetCheckerOptions_FiltersEmptyValues(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	opts := happydns.CheckerOptions{"keep": "yes", "drop_nil": nil, "drop_empty": ""}
	if err := uc.SetCheckerOptions("c1", nil, nil, nil, opts); err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetCheckerOptions("c1", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["drop_nil"]; ok {
		t.Error("nil value should have been filtered")
	}
	if _, ok := got["drop_empty"]; ok {
		t.Error("empty string value should have been filtered")
	}
	if got["keep"] != "yes" {
		t.Error("non-empty value should be kept")
	}
}

func TestAddCheckerOptions_MergesIntoExisting(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"a": "1", "b": "2"})

	merged, err := uc.AddCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"b": "updated", "c": "3"})
	if err != nil {
		t.Fatal(err)
	}
	if merged["a"] != "1" {
		t.Errorf("existing key 'a' should be preserved, got %v", merged["a"])
	}
	if merged["b"] != "updated" {
		t.Errorf("key 'b' should be updated, got %v", merged["b"])
	}
	if merged["c"] != "3" {
		t.Errorf("key 'c' should be added, got %v", merged["c"])
	}
}

func TestAddCheckerOptions_DeletesEmptyValues(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"a": "1", "b": "2"})

	merged, err := uc.AddCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"a": nil, "b": ""})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := merged["a"]; ok {
		t.Error("nil value should delete the key")
	}
	if _, ok := merged["b"]; ok {
		t.Error("empty string value should delete the key")
	}
}

func TestGetCheckerOption_SingleKey(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"x": "hello"})

	val, err := uc.GetCheckerOption("c1", uid, nil, nil, "x")
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}

	val, err = uc.GetCheckerOption("c1", uid, nil, nil, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if val != nil {
		t.Errorf("expected nil for missing key, got %v", val)
	}
}

func TestSetCheckerOption_SingleKey(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"a": "1"})

	if err := uc.SetCheckerOption("c1", uid, nil, nil, "b", "2"); err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetCheckerOptions("c1", uid, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != "1" || got["b"] != "2" {
		t.Errorf("unexpected options: %v", got)
	}
}

func TestSetCheckerOption_DeletesEmpty(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"a": "1", "b": "2"})

	if err := uc.SetCheckerOption("c1", uid, nil, nil, "a", nil); err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetCheckerOptions("c1", uid, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["a"]; ok {
		t.Error("key 'a' should have been deleted")
	}
	if got["b"] != "2" {
		t.Error("key 'b' should be preserved")
	}
}

// --- Scope merging tests ---

func TestGetCheckerOptions_MergesScopes(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Admin sets defaults.
	_ = uc.SetCheckerOptions("c1", nil, nil, nil, happydns.CheckerOptions{"a": "admin", "shared": "admin"})
	// User overrides shared.
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"b": "user", "shared": "user"})
	// Domain overrides shared again.
	_ = uc.SetCheckerOptions("c1", uid, did, nil, happydns.CheckerOptions{"c": "domain", "shared": "domain"})

	got, err := uc.GetCheckerOptions("c1", uid, did, nil)
	if err != nil {
		t.Fatal(err)
	}

	if got["a"] != "admin" {
		t.Errorf("admin key 'a' should be visible, got %v", got["a"])
	}
	if got["b"] != "user" {
		t.Errorf("user key 'b' should be visible, got %v", got["b"])
	}
	if got["c"] != "domain" {
		t.Errorf("domain key 'c' should be visible, got %v", got["c"])
	}
	if got["shared"] != "domain" {
		t.Errorf("'shared' should be overridden to 'domain', got %v", got["shared"])
	}
}

func TestGetCheckerOptions_ServiceScopeOverridesAll(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	sid := idPtr()

	_ = uc.SetCheckerOptions("c1", nil, nil, nil, happydns.CheckerOptions{"key": "admin"})
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"key": "user"})
	_ = uc.SetCheckerOptions("c1", uid, did, nil, happydns.CheckerOptions{"key": "domain"})
	_ = uc.SetCheckerOptions("c1", uid, did, sid, happydns.CheckerOptions{"key": "service"})

	got, err := uc.GetCheckerOptions("c1", uid, did, sid)
	if err != nil {
		t.Fatal(err)
	}
	if got["key"] != "service" {
		t.Errorf("service scope should win, got %v", got["key"])
	}
}

func TestGetCheckerOptionsPositional_ReturnsAllLevels(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	_ = uc.SetCheckerOptions("c1", nil, nil, nil, happydns.CheckerOptions{"a": "1"})
	_ = uc.SetCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"b": "2"})
	_ = uc.SetCheckerOptions("c1", uid, did, nil, happydns.CheckerOptions{"c": "3"})

	positionals, err := uc.GetCheckerOptionsPositional("c1", uid, did, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(positionals) != 3 {
		t.Fatalf("expected 3 positional levels, got %d", len(positionals))
	}
	// Least specific first.
	if positionals[0].UserId != nil {
		t.Error("first positional should be admin (no userId)")
	}
	if positionals[1].DomainId != nil {
		t.Error("second positional should be user (no domainId)")
	}
	if positionals[2].DomainId == nil {
		t.Error("third positional should be domain level")
	}
}

func TestGetCheckerOptions_EmptyStore(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	got, err := uc.GetCheckerOptions("nonexistent", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty options, got %v", got)
	}
}

func TestAddCheckerOptions_CreatesNewScope(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	merged, err := uc.AddCheckerOptions("c1", uid, nil, nil, happydns.CheckerOptions{"new": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if merged["new"] != "value" {
		t.Errorf("expected 'value', got %v", merged["new"])
	}
}

// --- BuildMergedCheckerOptions tests ---

func TestBuildMergedCheckerOptions(t *testing.T) {
	stored := happydns.CheckerOptions{"a": "stored", "shared": "stored"}
	run := happydns.CheckerOptions{"b": "run", "shared": "run"}

	result := checkerUC.BuildMergedCheckerOptions(stored, run)
	if result["a"] != "stored" {
		t.Errorf("stored key should be preserved")
	}
	if result["b"] != "run" {
		t.Errorf("run key should be added")
	}
	if result["shared"] != "run" {
		t.Errorf("run should override stored, got %v", result["shared"])
	}
}

func TestBuildMergedCheckerOptions_NilInputs(t *testing.T) {
	result := checkerUC.BuildMergedCheckerOptions(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}

	result = checkerUC.BuildMergedCheckerOptions(happydns.CheckerOptions{"a": "1"}, nil)
	if result["a"] != "1" {
		t.Errorf("stored key should be preserved with nil runOpts")
	}

	result = checkerUC.BuildMergedCheckerOptions(nil, happydns.CheckerOptions{"b": "2"})
	if result["b"] != "2" {
		t.Errorf("run key should be present with nil storedOpts")
	}
}

// --- Validation tests ---

// registerTestChecker is a helper that registers a checker in the global
// registry and returns its ID. Each call should use a unique ID.
func registerTestChecker(id string, def *happydns.CheckerDefinition) {
	def.ID = id
	def.Name = id
	checker.RegisterChecker(def)
}

func TestValidateOptions_UnknownChecker(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("no_such_checker", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err == nil {
		t.Fatal("expected error for unknown checker")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOptions_AdminScope_AcceptsAdminOpts(t *testing.T) {
	registerTestChecker("val_admin_ok", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "admin_key", Type: "string"},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// Admin scope (all nil): admin_key is valid.
	err := uc.ValidateOptions("val_admin_ok", nil, nil, nil, happydns.CheckerOptions{"admin_key": "hello"}, false)
	if err != nil {
		t.Fatalf("expected no error for valid admin opt, got: %v", err)
	}
}

func TestValidateOptions_AdminScope_RejectsDomainOpt(t *testing.T) {
	registerTestChecker("val_admin_reject_domain", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "admin_key", Type: "string"},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// Admin scope: domain_key should be rejected as unknown.
	err := uc.ValidateOptions("val_admin_reject_domain", nil, nil, nil, happydns.CheckerOptions{"domain_key": "x"}, false)
	if err == nil {
		t.Fatal("expected error for domain opt at admin scope")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected 'unknown' error, got: %v", err)
	}
}

func TestValidateOptions_DomainScope_AcceptsDomainOpts(t *testing.T) {
	registerTestChecker("val_domain_ok", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "admin_key", Type: "string"},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	err := uc.ValidateOptions("val_domain_ok", uid, did, nil, happydns.CheckerOptions{"domain_key": "hello"}, false)
	if err != nil {
		t.Fatalf("expected no error for valid domain opt, got: %v", err)
	}
}

func TestValidateOptions_DomainScope_RejectsAdminOpt(t *testing.T) {
	registerTestChecker("val_domain_reject_admin", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "admin_key", Type: "string"},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	err := uc.ValidateOptions("val_domain_reject_admin", uid, did, nil, happydns.CheckerOptions{"admin_key": "x"}, false)
	if err == nil {
		t.Fatal("expected error for admin opt at domain scope")
	}
}

func TestValidateOptions_UserScope_AcceptsUserOpts(t *testing.T) {
	registerTestChecker("val_user_ok", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "user_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	err := uc.ValidateOptions("val_user_ok", uid, nil, nil, happydns.CheckerOptions{"user_key": "val"}, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateOptions_ServiceScope_AcceptsServiceOpts(t *testing.T) {
	registerTestChecker("val_service_ok", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			ServiceOpts: []happydns.CheckerOptionDocumentation{
				{Id: "svc_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	sid := idPtr()
	err := uc.ValidateOptions("val_service_ok", uid, did, sid, happydns.CheckerOptions{"svc_key": "val"}, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateOptions_ServiceScope_RejectsRunOpts(t *testing.T) {
	registerTestChecker("val_service_reject_run", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			ServiceOpts: []happydns.CheckerOptionDocumentation{
				{Id: "svc_key", Type: "string"},
			},
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "run_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	sid := idPtr()
	err := uc.ValidateOptions("val_service_reject_run", uid, did, sid, happydns.CheckerOptions{"run_key": "val"}, false)
	if err == nil {
		t.Fatal("expected error for run opt at service scope")
	}
}

func TestValidateOptions_DomainScope_RejectsRunOpts(t *testing.T) {
	registerTestChecker("val_domain_reject_run", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "run_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	err := uc.ValidateOptions("val_domain_reject_run", uid, did, nil, happydns.CheckerOptions{"run_key": "val"}, false)
	if err == nil {
		t.Fatal("expected error for run opt at domain scope")
	}
}

func TestValidateOptions_RequiredField(t *testing.T) {
	registerTestChecker("val_required", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "must_have", Type: "string", Required: true, Label: "Must Have"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Missing required field.
	err := uc.ValidateOptions("val_required", uid, did, nil, happydns.CheckerOptions{}, false)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}

	// Present but empty.
	err = uc.ValidateOptions("val_required", uid, did, nil, happydns.CheckerOptions{"must_have": ""}, false)
	if err == nil {
		t.Fatal("expected error for empty required field")
	}

	// Valid.
	err = uc.ValidateOptions("val_required", uid, did, nil, happydns.CheckerOptions{"must_have": "ok"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptions_ChoicesField(t *testing.T) {
	registerTestChecker("val_choices", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "mode", Type: "string", Choices: []string{"fast", "slow", "auto"}},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	err := uc.ValidateOptions("val_choices", uid, nil, nil, happydns.CheckerOptions{"mode": "fast"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = uc.ValidateOptions("val_choices", uid, nil, nil, happydns.CheckerOptions{"mode": "invalid"}, false)
	if err == nil {
		t.Fatal("expected error for invalid choice")
	}
}

func TestValidateOptions_TypeCheckNumber(t *testing.T) {
	registerTestChecker("val_type_num", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "count", Type: "int"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// float64 is fine (JSON numbers are float64).
	err := uc.ValidateOptions("val_type_num", uid, did, nil, happydns.CheckerOptions{"count": float64(10)}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// String is not a number.
	err = uc.ValidateOptions("val_type_num", uid, did, nil, happydns.CheckerOptions{"count": "ten"}, false)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestValidateOptions_TypeCheckBool(t *testing.T) {
	registerTestChecker("val_type_bool", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "enabled", Type: "bool"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("val_type_bool", nil, nil, nil, happydns.CheckerOptions{"enabled": true}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = uc.ValidateOptions("val_type_bool", nil, nil, nil, happydns.CheckerOptions{"enabled": "true"}, false)
	if err == nil {
		t.Fatal("expected error for string instead of bool")
	}
}

func TestValidateOptions_EmptyOptionsValid(t *testing.T) {
	registerTestChecker("val_empty_ok", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "optional_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	err := uc.ValidateOptions("val_empty_ok", uid, did, nil, happydns.CheckerOptions{}, false)
	if err != nil {
		t.Fatalf("empty options should be valid when no required fields, got: %v", err)
	}
}

func TestValidateOptions_NoFieldsAtScope_AcceptsEmpty(t *testing.T) {
	registerTestChecker("val_no_fields_scope", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// At admin scope there are no fields defined; empty opts should pass.
	err := uc.ValidateOptions("val_no_fields_scope", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err != nil {
		t.Fatalf("empty options at scope with no fields should be valid, got: %v", err)
	}
}

func TestValidateOptions_NoFieldsAtScope_AcceptsAnything(t *testing.T) {
	// When no fields are defined at the target scope, validation is skipped
	// (the OptionsValidator may still reject), so unknown keys at a scope
	// without field definitions pass through.
	registerTestChecker("val_no_fields_pass", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// At user scope, no fields are declared, so any key is accepted.
	uid := idPtr()
	err := uc.ValidateOptions("val_no_fields_pass", uid, nil, nil, happydns.CheckerOptions{"anything": "value"}, false)
	if err != nil {
		t.Fatalf("scope with no fields should skip validation, got: %v", err)
	}
}

// --- OptionsValidator interface tests ---

func TestValidateOptions_OptionsValidatorCalled(t *testing.T) {
	registerTestChecker("val_validator_err", &happydns.CheckerDefinition{
		Rules: []happydns.CheckRule{
			&validatingRule{name: "r1", validateErr: fmt.Errorf("custom validation failed")},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("val_validator_err", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err == nil {
		t.Fatal("expected error from OptionsValidator")
	}
	if !strings.Contains(err.Error(), "custom validation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOptions_OptionsValidatorPasses(t *testing.T) {
	registerTestChecker("val_validator_ok", &happydns.CheckerDefinition{
		Rules: []happydns.CheckRule{
			&validatingRule{name: "r1", validateErr: nil},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("val_validator_ok", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptions_MultipleValidators_StopsAtFirst(t *testing.T) {
	registerTestChecker("val_multi_validators", &happydns.CheckerDefinition{
		Rules: []happydns.CheckRule{
			&validatingRule{name: "r1", validateErr: nil},
			&validatingRule{name: "r2", validateErr: fmt.Errorf("r2 failed")},
			&validatingRule{name: "r3", validateErr: fmt.Errorf("r3 failed")},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("val_multi_validators", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err == nil {
		t.Fatal("expected error from second validator")
	}
	if !strings.Contains(err.Error(), "r2 failed") {
		t.Errorf("expected r2 error, got: %v", err)
	}
}

// --- Rule-level options tests ---

func TestValidateOptions_RuleOptionsAtCorrectScope(t *testing.T) {
	registerTestChecker("val_rule_opts", &happydns.CheckerDefinition{
		Rules: []happydns.CheckRule{
			&ruleWithOptions{
				name: "rule_with_domain_opt",
				opts: happydns.CheckerOptionsDocumentation{
					DomainOpts: []happydns.CheckerOptionDocumentation{
						{Id: "rule_domain_opt", Type: "string"},
					},
					RunOpts: []happydns.CheckerOptionDocumentation{
						{Id: "rule_run_opt", Type: "string"},
					},
				},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Domain scope should accept rule's domain opt.
	err := uc.ValidateOptions("val_rule_opts", uid, did, nil, happydns.CheckerOptions{"rule_domain_opt": "val"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Domain scope should reject rule's run opt.
	err = uc.ValidateOptions("val_rule_opts", uid, did, nil, happydns.CheckerOptions{"rule_run_opt": "val"}, false)
	if err == nil {
		t.Fatal("expected error for run opt from rule at domain scope")
	}
}

func TestValidateOptions_CombinesDefAndRuleFields(t *testing.T) {
	registerTestChecker("val_combined", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "def_opt", Type: "string"},
			},
		},
		Rules: []happydns.CheckRule{
			&ruleWithOptions{
				name: "rule_extra",
				opts: happydns.CheckerOptionsDocumentation{
					DomainOpts: []happydns.CheckerOptionDocumentation{
						{Id: "rule_opt", Type: "string"},
					},
				},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Both def and rule opts should be accepted.
	err := uc.ValidateOptions("val_combined", uid, did, nil, happydns.CheckerOptions{
		"def_opt":  "a",
		"rule_opt": "b",
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Unknown key should be rejected.
	err = uc.ValidateOptions("val_combined", uid, did, nil, happydns.CheckerOptions{"unknown": "x"}, false)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

// --- Validation + OptionsValidator combined ---

func TestValidateOptions_FieldValidationRunsBeforeOptionsValidator(t *testing.T) {
	registerTestChecker("val_order", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "a", Type: "string", Required: true, Label: "A"},
			},
		},
		Rules: []happydns.CheckRule{
			&validatingRule{name: "r1", validateErr: fmt.Errorf("should not reach here")},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// Field validation should fail before reaching OptionsValidator.
	err := uc.ValidateOptions("val_order", nil, nil, nil, happydns.CheckerOptions{}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "should not reach here") {
		t.Error("OptionsValidator should not have been called; field validation should fail first")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}
}

// --- Scope isolation tests ---

func TestValidateOptions_DomainScope_DoesNotEnforceUserRequired(t *testing.T) {
	registerTestChecker("val_scope_isolation", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "user_required", Type: "string", Required: true},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_opt", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Domain scope should not enforce user-level required field.
	err := uc.ValidateOptions("val_scope_isolation", uid, did, nil, happydns.CheckerOptions{"domain_opt": "val"}, false)
	if err != nil {
		t.Fatalf("domain scope should not enforce user required field, got: %v", err)
	}
}

func TestValidateOptions_AdminScope_DoesNotEnforceServiceRequired(t *testing.T) {
	registerTestChecker("val_admin_no_svc", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "admin_opt", Type: "string"},
			},
			ServiceOpts: []happydns.CheckerOptionDocumentation{
				{Id: "svc_required", Type: "string", Required: true},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	err := uc.ValidateOptions("val_admin_no_svc", nil, nil, nil, happydns.CheckerOptions{"admin_opt": "val"}, false)
	if err != nil {
		t.Fatalf("admin scope should not enforce service required field, got: %v", err)
	}
}

func TestValidateOptions_UserScope_RejectsDomainOpt(t *testing.T) {
	registerTestChecker("val_user_reject_domain", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "user_opt", Type: "string"},
			},
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_opt", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	err := uc.ValidateOptions("val_user_reject_domain", uid, nil, nil, happydns.CheckerOptions{"domain_opt": "x"}, false)
	if err == nil {
		t.Fatal("expected error for domain opt at user scope")
	}
}

func TestValidateOptions_ServiceScope_RejectsDomainOpt(t *testing.T) {
	registerTestChecker("val_svc_reject_domain", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_opt", Type: "string"},
			},
			ServiceOpts: []happydns.CheckerOptionDocumentation{
				{Id: "svc_opt", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()
	sid := idPtr()
	err := uc.ValidateOptions("val_svc_reject_domain", uid, did, sid, happydns.CheckerOptions{"domain_opt": "x"}, false)
	if err == nil {
		t.Fatal("expected error for domain opt at service scope")
	}
}

// --- withRunOpts=true tests ---

func TestValidateOptions_WithRunOpts_AcceptsRunOptKeys(t *testing.T) {
	registerTestChecker("trig_run_accept", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "run_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// With withRunOpts=true, run_key should be accepted alongside domain_key.
	err := uc.ValidateOptions("trig_run_accept", uid, did, nil, happydns.CheckerOptions{
		"domain_key": "foo",
		"run_key":    "bar",
	}, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateOptions_WithRunOpts_EnforcesRequiredRunOpt(t *testing.T) {
	registerTestChecker("trig_run_required", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "must_run", Type: "string", Required: true, Label: "Must Run"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// Missing required run opt.
	err := uc.ValidateOptions("trig_run_required", uid, did, nil, happydns.CheckerOptions{}, true)
	if err == nil {
		t.Fatal("expected error for missing required run opt")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}

	// Present and non-empty.
	err = uc.ValidateOptions("trig_run_required", uid, did, nil, happydns.CheckerOptions{"must_run": "ok"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateOptions_WithRunOpts_StillRejectsUnknownKeys(t *testing.T) {
	registerTestChecker("trig_run_unknown", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "run_key", Type: "string"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	err := uc.ValidateOptions("trig_run_unknown", uid, did, nil, happydns.CheckerOptions{"totally_unknown": "x"}, true)
	if err == nil {
		t.Fatal("expected error for unknown key even with withRunOpts=true")
	}
}

func TestValidateOptions_WithRunOpts_RequiredRunOptNotEnforcedWhenFalse(t *testing.T) {
	registerTestChecker("trig_run_not_enforced", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_key", Type: "string"},
			},
			RunOpts: []happydns.CheckerOptionDocumentation{
				{Id: "must_run", Type: "string", Required: true},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// withRunOpts=false: required run opt is not enforced, run_key is not known.
	err := uc.ValidateOptions("trig_run_not_enforced", uid, did, nil, happydns.CheckerOptions{"domain_key": "val"}, false)
	if err != nil {
		t.Fatalf("persisted scope should not enforce run opt required field, got: %v", err)
	}
}

func TestValidateOptions_WithRunOpts_RuleRunOptsAccepted(t *testing.T) {
	registerTestChecker("trig_rule_run_accept", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "def_domain_opt", Type: "string"},
			},
		},
		Rules: []happydns.CheckRule{
			&ruleWithOptions{
				name: "rule_with_run",
				opts: happydns.CheckerOptionsDocumentation{
					RunOpts: []happydns.CheckerOptionDocumentation{
						{Id: "rule_run_opt", Type: "string", Required: true, Label: "Rule Run Opt"},
					},
				},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()
	did := idPtr()

	// withRunOpts=true: rule run opt is accepted and required.
	err := uc.ValidateOptions("trig_rule_run_accept", uid, did, nil, happydns.CheckerOptions{
		"def_domain_opt": "x",
	}, true)
	if err == nil {
		t.Fatal("expected error: rule's required run opt is missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}

	err = uc.ValidateOptions("trig_rule_run_accept", uid, did, nil, happydns.CheckerOptions{
		"def_domain_opt": "x",
		"rule_run_opt":   "y",
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// withRunOpts=false: rule run opt is unknown (rejected).
	err = uc.ValidateOptions("trig_rule_run_accept", uid, did, nil, happydns.CheckerOptions{
		"rule_run_opt": "y",
	}, false)
	if err == nil {
		t.Fatal("expected error: rule run opt should be unknown at domain scope without withRunOpts")
	}
}

// --- Auto-fill tests ---

// autoFillStore is a minimal in-memory store satisfying CheckAutoFillStorage.
type autoFillStore struct {
	domains map[string]*happydns.Domain
	zones   map[string]*happydns.ZoneMessage
	users   map[string]*happydns.User
}

func newAutoFillStore() *autoFillStore {
	return &autoFillStore{
		domains: make(map[string]*happydns.Domain),
		zones:   make(map[string]*happydns.ZoneMessage),
		users:   make(map[string]*happydns.User),
	}
}

func (s *autoFillStore) GetDomain(id happydns.Identifier) (*happydns.Domain, error) {
	if d, ok := s.domains[id.String()]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("domain %s not found", id)
}

func (s *autoFillStore) GetZone(id happydns.Identifier) (*happydns.ZoneMessage, error) {
	if z, ok := s.zones[id.String()]; ok {
		return z, nil
	}
	return nil, fmt.Errorf("zone %s not found", id)
}

func (s *autoFillStore) ListDomains(u *happydns.User) ([]*happydns.Domain, error) {
	return nil, nil
}

func (s *autoFillStore) GetUser(id happydns.Identifier) (*happydns.User, error) {
	if u, ok := s.users[id.String()]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user %s not found", id)
}

func TestBuildMergedCheckerOptionsWithAutoFill_InjectsValues(t *testing.T) {
	registerTestChecker("af_inject", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "domain_name_field", Type: "string", AutoFill: happydns.AutoFillDomainName},
				{Id: "user_opt", Type: "string"},
			},
		},
	})

	optStore := newOptionsStore()
	afStore := newAutoFillStore()

	uid := idPtr()
	did := idPtr()

	// Set up domain in auto-fill store.
	zoneId, _ := happydns.NewRandomIdentifier()
	afStore.domains[did.String()] = &happydns.Domain{
		Id:          *did,
		Owner:       *uid,
		DomainName:  "example.com.",
		ZoneHistory: []happydns.Identifier{zoneId},
	}
	afStore.zones[zoneId.String()] = &happydns.ZoneMessage{}

	uc := checkerUC.NewCheckerOptionsUsecase(optStore, afStore)
	_ = uc.SetCheckerOptions("af_inject", uid, nil, nil, happydns.CheckerOptions{"user_opt": "hello"})

	merged, err := uc.BuildMergedCheckerOptionsWithAutoFill("af_inject", uid, did, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if merged["domain_name_field"] != "example.com." {
		t.Errorf("expected auto-filled domain name, got %v", merged["domain_name_field"])
	}
	if merged["user_opt"] != "hello" {
		t.Errorf("expected stored opt to be preserved, got %v", merged["user_opt"])
	}
}

func TestBuildMergedCheckerOptionsWithAutoFill_OverridesRunOpts(t *testing.T) {
	registerTestChecker("af_override", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "dn", Type: "string", AutoFill: happydns.AutoFillDomainName},
			},
		},
	})

	optStore := newOptionsStore()
	afStore := newAutoFillStore()

	uid := idPtr()
	did := idPtr()
	zoneId, _ := happydns.NewRandomIdentifier()
	afStore.domains[did.String()] = &happydns.Domain{
		Id:          *did,
		Owner:       *uid,
		DomainName:  "real.example.com.",
		ZoneHistory: []happydns.Identifier{zoneId},
	}
	afStore.zones[zoneId.String()] = &happydns.ZoneMessage{}

	uc := checkerUC.NewCheckerOptionsUsecase(optStore, afStore)

	// Even if runOpts tries to set the auto-fill field, auto-fill wins.
	merged, err := uc.BuildMergedCheckerOptionsWithAutoFill("af_override", uid, did, nil,
		happydns.CheckerOptions{"dn": "user-provided.com."})
	if err != nil {
		t.Fatal(err)
	}

	if merged["dn"] != "real.example.com." {
		t.Errorf("auto-fill should override run opts, got %v", merged["dn"])
	}
}

func TestSetCheckerOptions_StripsAutoFillKeys(t *testing.T) {
	registerTestChecker("af_strip", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "dn", Type: "string", AutoFill: happydns.AutoFillDomainName},
				{Id: "normal", Type: "string"},
			},
		},
	})

	optStore := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(optStore, nil)

	uid := idPtr()
	err := uc.SetCheckerOptions("af_strip", uid, nil, nil, happydns.CheckerOptions{
		"dn":     "should-be-stripped",
		"normal": "kept",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := uc.GetCheckerOptions("af_strip", uid, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["dn"]; ok {
		t.Error("auto-fill key should have been stripped from persisted options")
	}
	if got["normal"] != "kept" {
		t.Errorf("normal key should be preserved, got %v", got["normal"])
	}
}

func TestValidateOptions_SkipsAutoFillFields(t *testing.T) {
	registerTestChecker("af_validate_skip", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			DomainOpts: []happydns.CheckerOptionDocumentation{
				{Id: "dn", Type: "string", AutoFill: happydns.AutoFillDomainName, Required: true},
				{Id: "normal", Type: "string"},
			},
		},
	})

	optStore := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(optStore, nil)

	uid := idPtr()
	did := idPtr()

	// The auto-fill field "dn" is required, but since it's auto-filled,
	// validation should not enforce it as a user-provided requirement.
	err := uc.ValidateOptions("af_validate_skip", uid, did, nil, happydns.CheckerOptions{
		"normal": "val",
	}, false)
	if err != nil {
		t.Fatalf("auto-fill required field should be skipped during validation, got: %v", err)
	}
}

// --- NoOverride tests ---

func TestGetCheckerOptions_NoOverridePreservesAdminValue(t *testing.T) {
	registerTestChecker("no_override_merge", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "threshold", Type: "number"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	// Set at admin scope.
	store.UpdateCheckerConfiguration("no_override_merge", nil, nil, nil, happydns.CheckerOptions{
		"locked": true,
	})
	// Attempt to override at user scope (should be ignored during merge).
	store.UpdateCheckerConfiguration("no_override_merge", uid, nil, nil, happydns.CheckerOptions{
		"locked":    false,
		"threshold": float64(42),
	})

	merged, err := uc.GetCheckerOptions("no_override_merge", uid, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["locked"] != true {
		t.Errorf("expected locked=true (admin value preserved), got %v", merged["locked"])
	}
	if merged["threshold"] != float64(42) {
		t.Errorf("expected threshold=42 (user value applied), got %v", merged["threshold"])
	}
}

func TestGetCheckerOptions_NoOverrideAllowsSameScope(t *testing.T) {
	registerTestChecker("no_override_same_scope", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	// Only admin scope sets the value, no conflict.
	store.UpdateCheckerConfiguration("no_override_same_scope", nil, nil, nil, happydns.CheckerOptions{
		"locked": true,
	})

	merged, err := uc.GetCheckerOptions("no_override_same_scope", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["locked"] != true {
		t.Errorf("expected locked=true, got %v", merged["locked"])
	}
}

func TestBuildMergedCheckerOptionsWithAutoFill_NoOverrideBlocksRunOpts(t *testing.T) {
	registerTestChecker("no_override_runopt", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "threshold", Type: "number"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	// Admin sets locked=true.
	store.UpdateCheckerConfiguration("no_override_runopt", nil, nil, nil, happydns.CheckerOptions{
		"locked": true,
	})
	// User sets threshold.
	store.UpdateCheckerConfiguration("no_override_runopt", uid, nil, nil, happydns.CheckerOptions{
		"threshold": float64(10),
	})

	// RunOpts tries to override locked.
	merged, err := uc.BuildMergedCheckerOptionsWithAutoFill("no_override_runopt", uid, nil, nil, happydns.CheckerOptions{
		"locked": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["locked"] != true {
		t.Errorf("expected locked=true (NoOverride should block runOpts), got %v", merged["locked"])
	}
	if merged["threshold"] != float64(10) {
		t.Errorf("expected threshold=10, got %v", merged["threshold"])
	}
}

func TestSetCheckerOptions_StripsNoOverrideAtLowerScope(t *testing.T) {
	registerTestChecker("no_override_set", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "threshold", Type: "number"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	// Try to set locked at user scope; should be silently stripped.
	err := uc.SetCheckerOptions("no_override_set", uid, nil, nil, happydns.CheckerOptions{
		"locked":    true,
		"threshold": float64(99),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check what was actually stored.
	stored := store.data[posKey("no_override_set", uid, nil, nil)]
	if _, ok := stored["locked"]; ok {
		t.Error("expected locked to be stripped from user-scope storage")
	}
	if stored["threshold"] != float64(99) {
		t.Errorf("expected threshold=99 to be stored, got %v", stored["threshold"])
	}
}

func TestAddCheckerOptions_StripsNoOverrideAtLowerScope(t *testing.T) {
	registerTestChecker("no_override_add", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "threshold", Type: "number"},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	// Pre-populate user scope with threshold.
	store.UpdateCheckerConfiguration("no_override_add", uid, nil, nil, happydns.CheckerOptions{
		"threshold": float64(50),
	})

	// Try to add locked at user scope; should be silently skipped.
	result, err := uc.AddCheckerOptions("no_override_add", uid, nil, nil, happydns.CheckerOptions{
		"locked":    true,
		"threshold": float64(75),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["locked"]; ok {
		t.Error("expected locked to be skipped in AddCheckerOptions result")
	}
	if result["threshold"] != float64(75) {
		t.Errorf("expected threshold=75, got %v", result["threshold"])
	}
}

func TestSetCheckerOption_RejectsNoOverrideAtLowerScope(t *testing.T) {
	registerTestChecker("no_override_set_single", &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "locked", Type: "boolean", NoOverride: true},
			},
		},
	})

	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)

	uid := idPtr()

	// Setting at admin scope should work.
	err := uc.SetCheckerOption("no_override_set_single", nil, nil, nil, "locked", true)
	if err != nil {
		t.Fatalf("expected SetCheckerOption at admin scope to succeed, got: %v", err)
	}

	// Setting at user scope should fail.
	err = uc.SetCheckerOption("no_override_set_single", uid, nil, nil, "locked", false)
	if err == nil {
		t.Fatal("expected error when setting NoOverride field at lower scope")
	}
	if !strings.Contains(err.Error(), "cannot be overridden") {
		t.Errorf("unexpected error message: %v", err)
	}
}
