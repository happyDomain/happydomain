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

package check_test

import (
	"testing"

	"git.happydns.org/happyDomain/checks"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	uc "git.happydns.org/happyDomain/internal/usecase/check"
	"git.happydns.org/happyDomain/model"
)

// ---------------------------------------------------------------------------
// mockCheckerForOptions – registered once at package init.
// ---------------------------------------------------------------------------

const testCheckerName = "test-mock-checker-options"

type mockCheckerForOptions struct{}

func (m *mockCheckerForOptions) ID() string   { return testCheckerName }
func (m *mockCheckerForOptions) Name() string { return testCheckerName }
func (m *mockCheckerForOptions) Availability() happydns.CheckerAvailability {
	return happydns.CheckerAvailability{ApplyToDomain: true}
}
func (m *mockCheckerForOptions) Options() happydns.CheckerOptionsDocumentation {
	return happydns.CheckerOptionsDocumentation{
		RunOpts: []happydns.CheckerOptionDocumentation{
			{Id: "run-param", Default: "run-default"},
		},
		DomainOpts: []happydns.CheckerOptionDocumentation{
			{Id: "domain-autofill", AutoFill: happydns.AutoFillDomainName},
			{Id: "domain-param", Default: "domain-default"},
		},
		UserOpts: []happydns.CheckerOptionDocumentation{
			{Id: "user-param"},
		},
		ServiceOpts: []happydns.CheckerOptionDocumentation{
			{Id: "service-param"},
		},
	}
}
func (m *mockCheckerForOptions) RunCheck(opts happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
	return nil, nil
}

func init() {
	checks.RegisterChecker(testCheckerName, &mockCheckerForOptions{})
}

// ---------------------------------------------------------------------------
// Helper: create a fresh in-memory database for each test.
// ---------------------------------------------------------------------------

func newOptionsTestDB(t *testing.T) storage.Storage {
	t.Helper()
	mem, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatalf("failed to create in-memory storage: %v", err)
	}
	db, err := kv.NewKVDatabase(mem)
	if err != nil {
		t.Fatalf("failed to create KV database: %v", err)
	}
	return db
}

func newTestCheckerUsecase(db storage.Storage) happydns.CheckerUsecase {
	return uc.NewCheckerUsecase(&happydns.Options{}, db, db)
}

// ---------------------------------------------------------------------------
// GetStoredCheckerOptionsNoDefault tests
// ---------------------------------------------------------------------------

func Test_GetStoredOptions_EmptyStore(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	opts, err := checkerUC.GetStoredCheckerOptionsNoDefault(testCheckerName, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("expected empty options from empty store, got %v", opts)
	}
}

func Test_GetStoredOptions_MergesStored(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	userId, _ := happydns.NewRandomIdentifier()
	// Store user-level option.
	if err := db.UpdateCheckerConfiguration(testCheckerName, &userId, nil, nil, happydns.CheckerOptions{"user-param": "val"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	opts, err := checkerUC.GetStoredCheckerOptionsNoDefault(testCheckerName, &userId, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts["user-param"] != "val" {
		t.Errorf("expected user-param='val', got %v", opts["user-param"])
	}
}

func Test_GetStoredOptions_AutoFillInjects(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Create a domain in the db.
	domain := &happydns.Domain{
		DomainName: "example.com.",
	}
	if err := db.CreateDomain(domain); err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	opts, err := checkerUC.GetStoredCheckerOptionsNoDefault(testCheckerName, nil, &domain.Id, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts["domain-autofill"] != "example.com." {
		t.Errorf("expected domain-autofill='example.com.', got %v", opts["domain-autofill"])
	}
}

func Test_GetStoredOptions_UnknownCheckerReturnsStored(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Store options for an unknown checker.
	if err := db.UpdateCheckerConfiguration("unknown-checker", nil, nil, nil, happydns.CheckerOptions{"some-param": "some-value"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	opts, err := checkerUC.GetStoredCheckerOptionsNoDefault("unknown-checker", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts["some-param"] != "some-value" {
		t.Errorf("expected some-param='some-value', got %v", opts["some-param"])
	}
}

// ---------------------------------------------------------------------------
// BuildMergedCheckerOptions tests
// ---------------------------------------------------------------------------

func Test_BuildMerged_DefaultsFirst(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	merged, err := checkerUC.BuildMergedCheckerOptions(testCheckerName, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["run-param"] != "run-default" {
		t.Errorf("expected run-param='run-default', got %v", merged["run-param"])
	}
	if merged["domain-param"] != "domain-default" {
		t.Errorf("expected domain-param='domain-default', got %v", merged["domain-param"])
	}
}

func Test_BuildMerged_StoredOverridesDefault(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	domainId, _ := happydns.NewRandomIdentifier()

	// Store domain-level option that overrides the default.
	if err := db.UpdateCheckerConfiguration(testCheckerName, nil, &domainId, nil, happydns.CheckerOptions{"domain-param": "custom"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	merged, err := checkerUC.BuildMergedCheckerOptions(testCheckerName, nil, &domainId, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["domain-param"] != "custom" {
		t.Errorf("expected domain-param='custom' (stored overrides default), got %v", merged["domain-param"])
	}
}

func Test_BuildMerged_RunOptsOverrideStored(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Store an admin-level value for run-param.
	if err := db.UpdateCheckerConfiguration(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"run-param": "stored-value"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	runOpts := happydns.CheckerOptions{"run-param": "runtime"}
	merged, err := checkerUC.BuildMergedCheckerOptions(testCheckerName, nil, nil, nil, runOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["run-param"] != "runtime" {
		t.Errorf("expected run-param='runtime' (runOpts wins), got %v", merged["run-param"])
	}
}

func Test_BuildMerged_AutoFillWinsOverAll(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Create domain in db.
	domain := &happydns.Domain{
		DomainName: "example.com.",
	}
	if err := db.CreateDomain(domain); err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Both stored and runOpts attempt to set domain-autofill.
	if err := db.UpdateCheckerConfiguration(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"domain-autofill": "manual-value"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}
	runOpts := happydns.CheckerOptions{"domain-autofill": "runtime-value"}

	merged, err := checkerUC.BuildMergedCheckerOptions(testCheckerName, nil, &domain.Id, nil, runOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Auto-fill always wins.
	if merged["domain-autofill"] != "example.com." {
		t.Errorf("expected domain-autofill='example.com.' (auto-fill wins), got %v", merged["domain-autofill"])
	}
}

func Test_BuildMerged_NilAutoFillStoreSkips(t *testing.T) {
	db := newOptionsTestDB(t)
	// Pass nil as the CheckAutoFillStorage interface (not a typed nil).
	checkerUC := uc.NewCheckerUsecase(&happydns.Options{}, db, nil)

	domainId, _ := happydns.NewRandomIdentifier()

	// Should not panic even when autoFillStore is nil.
	merged, err := checkerUC.BuildMergedCheckerOptions(testCheckerName, nil, &domainId, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// domain-autofill should NOT be set (no auto-fill storage available).
	if _, ok := merged["domain-autofill"]; ok {
		t.Errorf("expected domain-autofill to be absent when autoFillStore is nil, got %v", merged["domain-autofill"])
	}
}

// ---------------------------------------------------------------------------
// SetCheckerOptions tests
// ---------------------------------------------------------------------------

func Test_SetOptions_ServiceLevel(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	userId, _ := happydns.NewRandomIdentifier()
	domainId, _ := happydns.NewRandomIdentifier()
	serviceId, _ := happydns.NewRandomIdentifier()

	opts := happydns.CheckerOptions{"service-param": "val"}
	if err := checkerUC.SetCheckerOptions(testCheckerName, &userId, &domainId, &serviceId, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the configuration was stored at service scope.
	configs, err := db.GetCheckerConfiguration(testCheckerName, &userId, &domainId, &serviceId)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	// Find the service-level entry (UserId, DomainId, ServiceId all set).
	found := false
	for _, c := range configs {
		if c.UserId != nil && c.DomainId != nil && c.ServiceId != nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a service-level configuration entry to be stored")
	}
}

func Test_SetOptions_DomainLevel(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	userId, _ := happydns.NewRandomIdentifier()
	domainId, _ := happydns.NewRandomIdentifier()

	opts := happydns.CheckerOptions{"domain-param": "val"}
	if err := checkerUC.SetCheckerOptions(testCheckerName, &userId, &domainId, nil, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configs, err := db.GetCheckerConfiguration(testCheckerName, &userId, &domainId, nil)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	found := false
	for _, c := range configs {
		if c.UserId != nil && c.DomainId != nil && c.ServiceId == nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a domain-level configuration entry to be stored")
	}
}

func Test_SetOptions_UserLevel(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	userId, _ := happydns.NewRandomIdentifier()

	opts := happydns.CheckerOptions{"user-param": "val"}
	if err := checkerUC.SetCheckerOptions(testCheckerName, &userId, nil, nil, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configs, err := db.GetCheckerConfiguration(testCheckerName, &userId, nil, nil)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	found := false
	for _, c := range configs {
		if c.UserId != nil && c.DomainId == nil && c.ServiceId == nil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a user-level configuration entry to be stored")
	}
}

func Test_SetOptions_AdminLevel(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	opts := happydns.CheckerOptions{"run-param": "admin-val"}
	if err := checkerUC.SetCheckerOptions(testCheckerName, nil, nil, nil, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configs, err := db.GetCheckerConfiguration(testCheckerName, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	if len(configs) == 0 {
		t.Error("expected at least one admin-level configuration entry to be stored")
	}
}

func Test_SetOptions_UnknownCheckerErrors(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	opts := happydns.CheckerOptions{"param": "val"}
	if err := checkerUC.SetCheckerOptions("unknown-checker-xyz", nil, nil, nil, opts); err == nil {
		t.Fatal("expected error for unknown checker")
	}
}

// ---------------------------------------------------------------------------
// OverwriteSomeCheckerOptions tests
// ---------------------------------------------------------------------------

func Test_Overwrite_MergesWithExisting(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Pre-seed existing options at admin scope.
	if err := db.UpdateCheckerConfiguration(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"a": "1"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	if err := checkerUC.OverwriteSomeCheckerOptions(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"b": "2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve the stored options and verify both keys are present.
	configs, err := db.GetCheckerConfiguration(testCheckerName, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	if len(configs) == 0 {
		t.Fatal("expected at least one config entry")
	}
	merged := configs[0].Options
	if merged["a"] != "1" {
		t.Errorf("expected a='1' to be preserved, got %v", merged["a"])
	}
	if merged["b"] != "2" {
		t.Errorf("expected b='2' to be added, got %v", merged["b"])
	}
}

func Test_Overwrite_OverridesExistingKey(t *testing.T) {
	db := newOptionsTestDB(t)
	checkerUC := newTestCheckerUsecase(db)

	// Pre-seed existing options.
	if err := db.UpdateCheckerConfiguration(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"a": "1"}); err != nil {
		t.Fatalf("failed to seed option: %v", err)
	}

	if err := checkerUC.OverwriteSomeCheckerOptions(testCheckerName, nil, nil, nil, happydns.CheckerOptions{"a": "99"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configs, err := db.GetCheckerConfiguration(testCheckerName, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to retrieve config: %v", err)
	}
	if len(configs) == 0 {
		t.Fatal("expected at least one config entry")
	}
	if configs[0].Options["a"] != "99" {
		t.Errorf("expected a='99' after overwrite, got %v", configs[0].Options["a"])
	}
}
