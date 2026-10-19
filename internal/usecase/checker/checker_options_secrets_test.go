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
	"context"
	"testing"

	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

const secretChecker = "secret_opts_checker"

func init() {
	registerTestChecker(secretChecker, &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			AdminOpts: []happydns.CheckerOptionDocumentation{
				{Id: "api_key", Type: "string", Secret: true},
			},
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "user_token", Type: "string", Secret: true},
				{Id: "plain", Type: "string"},
			},
		},
	})
}

func TestRedactCheckerOptions(t *testing.T) {
	uc := checkerUC.NewCheckerOptionsUsecase(newOptionsStore(), nil)

	out := uc.RedactCheckerOptions(secretChecker, happydns.CheckerOptions{
		"user_token": "clear-token",
		"api_key":    "legacy-clear",
		"plain":      "visible",
	})
	if out["user_token"] != happydns.RedactedSecret || out["api_key"] != happydns.RedactedSecret || out["plain"] != "visible" {
		t.Errorf("redacted = %v", out)
	}

	if out := uc.RedactCheckerOptions(secretChecker, happydns.CheckerOptions{"plain": "x"}); out["user_token"] != nil {
		t.Errorf("an absent secret must stay absent: %v", out)
	}

	if v := uc.RedactCheckerOptionValue(secretChecker, "user_token", "clear-token"); v != happydns.RedactedSecret {
		t.Errorf("RedactCheckerOptionValue(secret) = %v, want it withheld", v)
	}
	if v := uc.RedactCheckerOptionValue(secretChecker, "plain", "visible"); v != "visible" {
		t.Errorf("RedactCheckerOptionValue(plain) = %v, want it as is", v)
	}
}

// What leaves happyDomain withholds every value that may be a secret: those
// documented so, any option the documentation does not describe, and every
// option of a checker this version does not load.
func TestRedactOptionsWithholdsWhatMayBeSecret(t *testing.T) {
	opts := happydns.CheckerOptions{
		"user_token": "legacy-clear",
		"plain":      "visible",
		"renamed":    "clear-but-undocumented",
		"api_key":    "",
	}
	out := checkerUC.RedactOptions(secretChecker, opts)
	want := happydns.CheckerOptions{
		"user_token": happydns.RedactedSecret,
		"plain":      "visible",
		"renamed":    happydns.RedactedSecret,
		"api_key":    "",
	}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("redacted[%q] = %v, want %v", k, out[k], v)
		}
	}
	if opts["user_token"] != "legacy-clear" {
		t.Errorf("the options given were changed: %v", opts)
	}
	if out := checkerUC.RedactOptions("gone_checker", happydns.CheckerOptions{"anything": "x"}); out["anything"] != happydns.RedactedSecret {
		t.Errorf("an option of an unknown checker = %v, want it withheld", out["anything"])
	}
}

// Every value withheld comes back as the placeholder: echoed back, it keeps
// the stored value, documented secret or not.
func TestPlaceholderKeepsAnyStoredOption(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"renamed": "undocumented", "plain": "a"}

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"renamed": happydns.RedactedSecret, "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions: %v", err)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["renamed"]; v != "undocumented" {
		t.Errorf("renamed = %v, want the stored value kept", v)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["plain"]; v != "b" {
		t.Errorf("plain = %v, want it replaced", v)
	}

	if _, err := uc.AddCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"renamed": happydns.RedactedSecret}); err != nil {
		t.Fatalf("AddCheckerOptions: %v", err)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["renamed"]; v != "undocumented" {
		t.Errorf("renamed = %v, want the stored value kept", v)
	}

	if err := uc.SetCheckerOption(context.Background(), secretChecker, user, nil, nil, "renamed", happydns.RedactedSecret); err != nil {
		t.Fatalf("SetCheckerOption: %v", err)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["renamed"]; v != "undocumented" {
		t.Errorf("renamed = %v, want the stored value kept", v)
	}
}

// The placeholder in the options of a run stands for what is stored.
func TestPlaceholderInRunOptionsIsIgnored(t *testing.T) {
	uc := checkerUC.NewCheckerOptionsUsecase(newOptionsStore(), nil)
	user := idPtr()
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": "stored"}); err != nil {
		t.Fatal(err)
	}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": happydns.RedactedSecret})
	if err != nil || merged["plain"] != "stored" {
		t.Errorf("merged = %v, %v; want the stored value", merged, err)
	}
}

// The placeholder with nothing stored behind it stands for no value: it is
// dropped, never stored.
func TestPlaceholderWithNothingStoredIsDropped(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)
	user := idPtr()

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": happydns.RedactedSecret, "user_token": happydns.RedactedSecret}); err != nil {
		t.Fatalf("SetCheckerOptions: %v", err)
	}
	if _, err := uc.AddCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": happydns.RedactedSecret}); err != nil {
		t.Fatalf("AddCheckerOptions: %v", err)
	}
	for _, k := range []string{"plain", "user_token"} {
		if v, ok := store.data[posKey(secretChecker, user, nil, nil)][k]; ok {
			t.Errorf("stored %s = %v, want nothing", k, v)
		}
	}
}
