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
	"errors"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	checkerUC "git.happydns.org/happyDomain/internal/usecase/checker"
	"git.happydns.org/happyDomain/model"
)

// goneChecker is never registered: it stands for a checker removed since its
// options were saved, or whose option no longer documents Secret.
const goneChecker = "secret_opts_gone_checker"

// A value stored sealed is a secret, whatever the documentation of its
// checker says now: it is opened for runs, and never sent out.
func TestSealedOptionsAreFoundByTheirPrefix(t *testing.T) {
	instance, _ := optionsManagers(t, secrettest.NewSafes(), nil)

	store := newListableOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(instance)
	user := idPtr()

	token, err := instance.SealValue(context.Background(), checkerUC.OptionsSecretContext(goneChecker, user, nil, nil, "old_key"), "the-key")
	if err != nil || !secret.IsSealed(token) {
		t.Fatalf("SealValue = %q, %v", token, err)
	}
	store.data[posKey(goneChecker, user, nil, nil)] = happydns.CheckerOptions{"old_key": token, "plain": "p"}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(goneChecker, user, nil, nil, nil)
	if err != nil || merged["old_key"] != "the-key" {
		t.Errorf("merged = %v, %v; want the sealed value opened", merged, err)
	}
	forUse, err := uc.GetCheckerOptionsForUse(goneChecker, user, nil, nil)
	if err != nil || forUse["old_key"] != "the-key" {
		t.Errorf("GetCheckerOptionsForUse = %v, %v; want the sealed value opened", forUse, err)
	}

	// Without the documentation of its checker, nothing tells that "plain"
	// is not a secret either: it is withheld too.
	out := checkerUC.RedactOptions(goneChecker, happydns.CheckerOptions{"old_key": token, "plain": "p"})
	if out["old_key"] != happydns.RedactedSecret || out["plain"] != happydns.RedactedSecret {
		t.Errorf("redacted = %v, want every value withheld", out)
	}
	if v := uc.RedactCheckerOptionValue(goneChecker, "old_key", token); v != happydns.RedactedSecret {
		t.Errorf("RedactCheckerOptionValue = %v, want the sealed value withheld", v)
	}
	if v := uc.RedactCheckerOptionValue(goneChecker, "plain", "p"); v != happydns.RedactedSecret {
		t.Errorf("RedactCheckerOptionValue(plain) = %v, want it withheld", v)
	}
	if v := uc.RedactCheckerOptionValue(goneChecker, "plain", ""); v != "" {
		t.Errorf("RedactCheckerOptionValue(empty) = %v, want it left unset", v)
	}
}

// Echoing back the placeholder of a value sent out redacted keeps what is
// stored, whether or not the option is documented as secret.
func TestEchoOfASealedUndocumentedOptionKeepsIt(t *testing.T) {
	m := instanceOptionsManager(t)
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(m)
	user := idPtr()

	token, err := m.SealValue(context.Background(), checkerUC.OptionsSecretContext(secretChecker, user, nil, nil, "plain"), "was-secret")
	if err != nil {
		t.Fatal(err)
	}
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"plain": token}

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": happydns.RedactedSecret}); err != nil {
		t.Fatalf("SetCheckerOptions: %v", err)
	}
	if v := storedString(t, store, user, "plain"); v != token {
		t.Errorf("stored = %q, want the sealed value kept", v)
	}
	if _, err := uc.AddCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": happydns.RedactedSecret}); err != nil {
		t.Fatalf("AddCheckerOptions: %v", err)
	}
	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "plain", happydns.RedactedSecret); err != nil {
		t.Fatalf("SetCheckerOption: %v", err)
	}
	if v := storedString(t, store, user, "plain"); v != token {
		t.Errorf("stored = %q, want the sealed value kept", v)
	}
}

// A client never has a reason to send a sealed value, for any option.
func TestSealedValuesFromClientsAreRefusedForEveryOption(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	user := idPtr()
	sealed := "hds:1:AQ:c2VhbGVk"

	var verr happydns.ValidationError
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": sealed}); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOptions = %v, want a ValidationError", err)
	}
	if _, err := uc.AddCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": sealed}); !errors.As(err, &verr) {
		t.Errorf("AddCheckerOptions = %v, want a ValidationError", err)
	}
	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "plain", sealed); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOption = %v, want a ValidationError", err)
	}
}

// A placeholder stored by mistake is never handed to a checker as the
// credential it stands for.
func TestStoredPlaceholderFailsRuns(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": happydns.RedactedSecret}

	if _, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, nil); !errors.Is(err, secret.ErrRedactedSecret) {
		t.Errorf("BuildMergedCheckerOptionsWithAutoFill = %v, want ErrRedactedSecret", err)
	}
}

// Options given for one run follow the rules of saved ones: the placeholder
// keeps what is stored, a sealed value is refused.
func TestRunOptionsFollowTheSecretRules(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	user := idPtr()
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "kept"}); err != nil {
		t.Fatal(err)
	}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret})
	if err != nil || merged["user_token"] != "kept" {
		t.Errorf("merged = %v, %v; want the placeholder to keep the stored value", merged, err)
	}

	var verr happydns.ValidationError
	for _, key := range []string{"user_token", "plain"} {
		if _, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, happydns.CheckerOptions{key: "hds:1:AQ:c2VhbGVk"}); !errors.As(err, &verr) {
			t.Errorf("run option %s sealed = %v, want a ValidationError", key, err)
		}
	}
}

const validatedSecretChecker = "secret_opts_validated_checker"

// tokenFormatRule checks the format of the secret option user_token, as a
// checker validating an API key would.
type tokenFormatRule struct{}

func (tokenFormatRule) Name() string        { return "token_format" }
func (tokenFormatRule) Description() string { return "checks the token format" }
func (tokenFormatRule) Evaluate(_ context.Context, _ happydns.ObservationGetter, _ happydns.CheckerOptions) []happydns.CheckState {
	return []happydns.CheckState{{Status: happydns.StatusOK}}
}
func (tokenFormatRule) ValidateOptions(opts happydns.CheckerOptions) error {
	if v, ok := opts["user_token"]; ok {
		if s, _ := v.(string); !strings.HasPrefix(s, "tok-") {
			return errors.New("user_token must start with tok-")
		}
	}
	return nil
}

func init() {
	registerTestChecker(validatedSecretChecker, &happydns.CheckerDefinition{
		Options: happydns.CheckerOptionsDocumentation{
			UserOpts: []happydns.CheckerOptionDocumentation{
				{Id: "user_token", Type: "string", Secret: true},
				{Id: "plain", Type: "string"},
			},
		},
		Rules: []happydns.CheckRule{tokenFormatRule{}},
	})
}

// Validation sees the value that will be stored or used, never a sealed
// value nor the placeholder standing for it.
func TestValidationSeesSecretsInClear(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	user := idPtr()
	if err := uc.SetCheckerOptions(validatedSecretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "tok-kept"}); err != nil {
		t.Fatal(err)
	}

	merged, err := uc.MergeCheckerOptions(validatedSecretChecker, user, nil, nil, happydns.CheckerOptions{"plain": "b"})
	if err != nil || merged["user_token"] != "tok-kept" {
		t.Fatalf("MergeCheckerOptions = %v, %v; want the stored secret opened", merged, err)
	}
	if err := uc.ValidateOptions(validatedSecretChecker, user, nil, nil, merged, false); err != nil {
		t.Errorf("ValidateOptions(merged) = %v", err)
	}

	echo := happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "b"}
	if err := uc.ValidateOptions(validatedSecretChecker, user, nil, nil, echo, false); err != nil {
		t.Errorf("ValidateOptions(echo) = %v, want the placeholder to stand for the stored value", err)
	}
	if err := uc.ValidateOptions(validatedSecretChecker, user, nil, nil, echo, true); err != nil {
		t.Errorf("ValidateOptions(echo, run) = %v, want the placeholder to stand for the stored value", err)
	}

	var verr happydns.ValidationError
	if err := uc.ValidateOptions(validatedSecretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "hds:1:AQ:c2VhbGVk"}, false); !errors.As(err, &verr) {
		t.Errorf("ValidateOptions(sealed) = %v, want a ValidationError", err)
	}
	if err := uc.ValidateOptions(validatedSecretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": 42}, false); !errors.As(err, &verr) {
		t.Errorf("ValidateOptions(number) = %v, want a ValidationError", err)
	}
}

// countingSafes counts the safes read by identifier.
type countingSafes struct {
	*secrettest.Safes
	gets int
}

func (s *countingSafes) GetSafe(id happydns.Identifier) (*happydns.Safe, error) {
	s.gets++
	return s.Safes.GetSafe(id)
}

// Opening the options of a checker, as each status listing does, reads each
// safe once however many secrets it holds.
func TestOpeningOptionsReadsEachSafeOnce(t *testing.T) {
	safes := &countingSafes{Safes: secrettest.NewSafes()}
	m, _ := optionsManagers(t, safes, nil)
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(m)
	user := idPtr()

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "u"}); err != nil {
		t.Fatal(err)
	}
	extra, err := m.SealValue(context.Background(), checkerUC.OptionsSecretContext(secretChecker, user, nil, nil, "extra"), "x")
	if err != nil {
		t.Fatal(err)
	}
	store.data[posKey(secretChecker, user, nil, nil)]["extra"] = extra

	safes.gets = 0
	opts, err := uc.GetCheckerOptionsForUse(secretChecker, user, nil, nil)
	if err != nil || opts["user_token"] != "u" || opts["extra"] != "x" {
		t.Fatalf("GetCheckerOptionsForUse = %v, %v", opts, err)
	}
	if safes.gets != 1 {
		t.Errorf("the user's safe was read %d times, want once", safes.gets)
	}
}
