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
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
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

func instanceOptionsUC(t *testing.T) (*checkerUC.CheckerOptionsUsecase, *optionsStore) {
	t.Helper()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: secrettest.NewSafes()})
	if err != nil {
		t.Fatal(err)
	}
	store := newOptionsStore()
	return checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(m), store
}

func storedString(t *testing.T, store *optionsStore, userId *happydns.Identifier, key string) string {
	t.Helper()
	v, _ := store.data[posKey(secretChecker, userId, nil, nil)][key].(string)
	return v
}

func TestSecretOptionsAreSealedAndOpenedForRuns(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, nil, nil, nil, happydns.CheckerOptions{"api_key": "operator-key"}); err != nil {
		t.Fatalf("SetCheckerOptions(admin): %v", err)
	}
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "user-token", "plain": "visible"}); err != nil {
		t.Fatalf("SetCheckerOptions(user): %v", err)
	}

	for _, v := range []string{storedString(t, store, nil, "api_key"), storedString(t, store, user, "user_token")} {
		if !strings.HasPrefix(v, "hds:1:") {
			t.Errorf("stored secret option = %q, want it sealed", v)
		}
	}
	if storedString(t, store, user, "plain") != "visible" {
		t.Error("a non-secret option must be stored as is")
	}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, nil)
	if err != nil {
		t.Fatalf("BuildMergedCheckerOptionsWithAutoFill: %v", err)
	}
	if merged["api_key"] != "operator-key" || merged["user_token"] != "user-token" || merged["plain"] != "visible" {
		t.Errorf("merged = %v, want the secrets in clear", merged)
	}

	forUse, err := uc.GetCheckerOptionsForUse(context.Background(), secretChecker, user, nil, nil)
	if err != nil || forUse["user_token"] != "user-token" {
		t.Errorf("GetCheckerOptionsForUse = %v, %v; want the secrets in clear", forUse, err)
	}
}

func TestSecretOptionsLegacyPlaintextStillWork(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "legacy"}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, nil)
	if err != nil || merged["user_token"] != "legacy" {
		t.Errorf("merged = %v, %v; want the legacy value", merged, err)
	}
}

func TestRedactCheckerOptions(t *testing.T) {
	uc, _ := instanceOptionsUC(t)

	out := uc.RedactCheckerOptions(secretChecker, happydns.CheckerOptions{
		"user_token": "hds:1:AQ:c2VhbGVk",
		"api_key":    "legacy-clear",
		"plain":      "visible",
	})
	if out["user_token"] != happydns.RedactedSecret || out["api_key"] != happydns.RedactedSecret || out["plain"] != "visible" {
		t.Errorf("redacted = %v", out)
	}

	if out := uc.RedactCheckerOptions(secretChecker, happydns.CheckerOptions{"plain": "x"}); out["user_token"] != nil {
		t.Errorf("an absent secret must stay absent: %v", out)
	}
}

func TestSecretOptionsEchoKeepsStoredValue(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "kept", "plain": "a"}); err != nil {
		t.Fatal(err)
	}
	sealed := storedString(t, store, user, "user_token")

	// Full replace echoing the placeholder.
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed || storedString(t, store, user, "plain") != "b" {
		t.Errorf("after full replace: %v", store.data[posKey(secretChecker, user, nil, nil)])
	}

	// Partial merge echoing the placeholder.
	if _, err := uc.AddCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "c"}); err != nil {
		t.Fatalf("AddCheckerOptions(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed {
		t.Error("the merge replaced the stored secret by the placeholder")
	}

	// Single option echoing the placeholder.
	if err := uc.SetCheckerOption(context.Background(), secretChecker, user, nil, nil, "user_token", happydns.RedactedSecret); err != nil {
		t.Fatalf("SetCheckerOption(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed {
		t.Error("setting the placeholder replaced the stored secret")
	}

	// A new value replaces it.
	if err := uc.SetCheckerOption(context.Background(), secretChecker, user, nil, nil, "user_token", "rotated"); err != nil {
		t.Fatal(err)
	}
	if v := storedString(t, store, user, "user_token"); v == sealed || !strings.HasPrefix(v, "hds:1:") {
		t.Errorf("after rotation = %q, want a new sealed value", v)
	}
}

func TestSecretOptionsRefuseBadInput(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	user := idPtr()

	if err := uc.ValidateOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": 42.0}, false); err == nil {
		t.Error("a non-string secret option passed validation")
	}
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": 42.0}); err == nil {
		t.Error("a non-string secret option was stored")
	}

	sealed := "hds:1:AQ:c2VhbGVk"
	var verr happydns.ValidationError
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": sealed}); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOptions(sealed from client) = %v, want a ValidationError", err)
	}
	if _, err := uc.MergeCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": sealed}); !errors.As(err, &verr) {
		t.Errorf("MergeCheckerOptions(sealed from client) = %v, want a ValidationError", err)
	}
	if err := uc.SetCheckerOption(context.Background(), secretChecker, user, nil, nil, "user_token", sealed); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOption(sealed from client) = %v, want a ValidationError", err)
	}
}

func TestSecretOptionsFailClosedWithoutManager(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)
	user := idPtr()

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "v"}); err == nil {
		t.Error("a secret option was stored without a secret manager")
	}
	// Options without secrets need none.
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": "v"}); err != nil {
		t.Errorf("SetCheckerOptions(plain) = %v", err)
	}

	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "hds:1:AQ:c2VhbGVk"}
	if _, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, nil); err == nil {
		t.Error("a run got options with a secret it could not open")
	}
}

func TestSecretOptionsResealAndInspect(t *testing.T) {
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: safes})

	store := newListableOptionsStore()
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "legacy", "plain": "p"}
	store.data[posKey(secretChecker, nil, nil, nil)] = happydns.CheckerOptions{"api_key": "operator"}

	holder := checkerUC.NewCheckerOptionsSecrets(store, instance)
	counts, err := holder.InspectSecrets(context.Background())
	if err != nil || counts.Clear != 2 {
		t.Fatalf("InspectSecrets = %+v, %v; want 2 clear", counts, err)
	}

	report, err := holder.ResealSecrets(context.Background())
	if err != nil || report.Changed != 2 || report.Failed != 0 {
		t.Fatalf("ResealSecrets = %+v, %v", report, err)
	}
	if v := storedString(t, store.optionsStore, user, "user_token"); !strings.HasPrefix(v, "hds:1:") {
		t.Errorf("user option after reseal = %q", v)
	}
	counts, _ = holder.InspectSecrets(context.Background())
	if counts.Sealed[secret.KindInstance] != 2 || counts.Clear != 0 {
		t.Errorf("InspectSecrets after = %+v", counts)
	}

	report, err = checkerUC.NewCheckerOptionsSecrets(store, plaintext).ResealSecrets(context.Background())
	if err != nil || report.Changed != 2 {
		t.Fatalf("ResealSecrets(plaintext) = %+v, %v", report, err)
	}
	if v := storedString(t, store.optionsStore, user, "user_token"); v != "legacy" {
		t.Errorf("user option back in clear = %q", v)
	}
}

// listableOptionsStore also lists every stored configuration.
type listableOptionsStore struct {
	*optionsStore
}

func newListableOptionsStore() *listableOptionsStore {
	return &listableOptionsStore{newOptionsStore()}
}

func (s *listableOptionsStore) ListAllCheckerConfigurations() (happydns.Iterator[happydns.CheckerOptionsPositional], error) {
	var items []*happydns.CheckerOptionsPositional
	for key, opts := range s.data {
		parts := strings.Split(key, "|")
		p := &happydns.CheckerOptionsPositional{CheckName: parts[0], Options: opts}
		for i, dst := range []**happydns.Identifier{&p.UserId, &p.DomainId, &p.ServiceId} {
			if parts[i+1] != "" {
				id, _ := happydns.NewIdentifierFromString(parts[i+1])
				*dst = &id
			}
		}
		items = append(items, p)
	}
	return &positionalIterator{items: items, idx: -1}, nil
}

type positionalIterator struct {
	items []*happydns.CheckerOptionsPositional
	idx   int
}

func (it *positionalIterator) Next() bool                               { it.idx++; return it.idx < len(it.items) }
func (it *positionalIterator) NextWithError() bool                      { return it.Next() }
func (it *positionalIterator) Item() *happydns.CheckerOptionsPositional { return it.items[it.idx] }
func (it *positionalIterator) DropItem() error                          { return nil }
func (it *positionalIterator) Key() string                              { return "" }
func (it *positionalIterator) Raw() any                                 { return nil }
func (it *positionalIterator) Err() error                               { return nil }
func (it *positionalIterator) Close()                                   {}

// A sealed value is a secret whatever the documentation says: its checker no
// longer loaded, or the option renamed since. It is counted and resealed, so
// that dropping the safes cannot lose it, and opened for a run.
func TestSealedOptionsAreSecretsWhateverTheDocs(t *testing.T) {
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: safes})

	user := idPtr()
	seal := func(checker, k, v string) string {
		out, err := instance.SealValue(context.Background(), checkerUC.OptionsSecretContext(checker, user, nil, nil, k), v)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	store := newListableOptionsStore()
	store.data[posKey("gone_checker", user, nil, nil)] = happydns.CheckerOptions{"token": seal("gone_checker", "token", "gone")}
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"renamed": seal(secretChecker, "renamed", "kept")}

	counts, err := checkerUC.NewCheckerOptionsSecrets(store, instance).InspectSecrets(context.Background())
	if err != nil || counts.Sealed[secret.KindInstance] != 2 {
		t.Errorf("InspectSecrets = %+v, %v; want both sealed values counted", counts, err)
	}

	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(instance)
	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, nil)
	if err != nil || merged["renamed"] != "kept" {
		t.Errorf("merged = %v, %v; want the undocumented secret opened", merged, err)
	}

	report, err := checkerUC.NewCheckerOptionsSecrets(store, plaintext).ResealSecrets(context.Background())
	if err != nil || report.Changed != 2 || report.Failed != 0 {
		t.Fatalf("ResealSecrets(plaintext) = %+v, %v", report, err)
	}
	if v := store.data[posKey("gone_checker", user, nil, nil)]["token"]; v != "gone" {
		t.Errorf("option of an unloaded checker = %v, want it back in clear", v)
	}
}

// What leaves happyDomain withholds every value that may be a secret: those
// documented so, any sealed one, any option the documentation does not
// describe, and every option of a checker this version does not load.
func TestRedactOptionsWithholdsWhatMayBeSecret(t *testing.T) {
	out := checkerUC.RedactOptions(secretChecker, happydns.CheckerOptions{
		"user_token": "legacy-clear",
		"plain":      "hds:1:AQ:c2VhbGVk",
		"renamed":    "clear-but-undocumented",
		"api_key":    "",
	})
	want := happydns.CheckerOptions{
		"user_token": happydns.RedactedSecret,
		"plain":      happydns.RedactedSecret,
		"renamed":    happydns.RedactedSecret,
		"api_key":    "",
	}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("redacted[%q] = %v, want %v", k, out[k], v)
		}
	}
	if out := checkerUC.RedactOptions(secretChecker, happydns.CheckerOptions{"plain": "visible"}); out["plain"] != "visible" {
		t.Errorf("a documented plain option = %v, want it as is", out["plain"])
	}
	if out := checkerUC.RedactOptions("gone_checker", happydns.CheckerOptions{"anything": "x"}); out["anything"] != happydns.RedactedSecret {
		t.Errorf("an option of an unknown checker = %v, want it withheld", out["anything"])
	}
}

// A client never sends a sealed value, whatever the option.
func TestSealedValueFromClientRefusedOnAnyOption(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	var verr happydns.ValidationError
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, idPtr(), nil, nil, happydns.CheckerOptions{"plain": "hds:1:AQ:c2VhbGVk"}); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOptions = %v, want a ValidationError", err)
	}
}

// Every value withheld comes back as the placeholder: echoed back, it keeps
// the stored value, documented secret or not.
func TestPlaceholderKeepsAnyStoredOption(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"renamed": "undocumented", "plain": "a"}

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"renamed": happydns.RedactedSecret, "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions: %v", err)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["renamed"]; v != "undocumented" {
		t.Errorf("renamed = %v, want the stored value kept", v)
	}

	if _, err := uc.AddCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"renamed": happydns.RedactedSecret}); err != nil {
		t.Fatalf("AddCheckerOptions: %v", err)
	}
	if v := store.data[posKey(secretChecker, user, nil, nil)]["renamed"]; v != "undocumented" {
		t.Errorf("renamed = %v, want the stored value kept", v)
	}
}

// The placeholder in the options of a run stands for what is stored.
func TestPlaceholderInRunOptionsIsIgnored(t *testing.T) {
	uc, _ := instanceOptionsUC(t)
	user := idPtr()
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "stored"}); err != nil {
		t.Fatal(err)
	}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret})
	if err != nil || merged["user_token"] != "stored" {
		t.Errorf("merged = %v, %v; want the stored value", merged, err)
	}
}

// A record that does not decode may hold sealed options: it is counted, and
// keeps neither the others from being resealed nor the status from being
// reported.
func TestOptionsSecretsGoPastUndecodableRecords(t *testing.T) {
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}
	if err := kvdb.Put("cfg|corrupt", "not options"); err != nil {
		t.Fatal(err)
	}

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	user := idPtr()
	if err := db.UpdateCheckerConfiguration(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "legacy"}); err != nil {
		t.Fatal(err)
	}

	holder := checkerUC.NewCheckerOptionsSecrets(db, instance)
	counts, err := holder.InspectSecrets(context.Background())
	if err != nil || counts.Clear != 1 || counts.Undecodable != 1 {
		t.Errorf("InspectSecrets = %+v, %v; want 1 clear, 1 undecodable", counts, err)
	}

	report, err := holder.ResealSecrets(context.Background())
	if err != nil || report.Changed != 1 || report.Failed != 1 {
		t.Errorf("ResealSecrets = %+v, %v; want the readable scope resealed, 1 failure", report, err)
	}
}

// A sealed option is bound to its scope: copied into another user's options,
// or another checker's, it does not open for a run, and the reseal leaves it
// as stored, reports it, and goes on with the others.
func TestSealedOptionsAreBoundToTheirScope(t *testing.T) {
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: safes})

	store := newListableOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(instance)
	victim, attacker := idPtr(), idPtr()
	if err := uc.SetCheckerOptions(context.Background(), secretChecker, victim, nil, nil, happydns.CheckerOptions{"user_token": "victim-token"}); err != nil {
		t.Fatal(err)
	}
	sealed := storedString(t, store.optionsStore, victim, "user_token")
	store.data[posKey(secretChecker, attacker, nil, nil)] = happydns.CheckerOptions{"user_token": sealed}
	store.data[posKey("gone_checker", victim, nil, nil)] = happydns.CheckerOptions{"user_token": sealed}

	if merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), secretChecker, attacker, nil, nil, nil); err == nil {
		t.Errorf("opened in another user's scope: %v", merged)
	}
	if opts, err := uc.GetCheckerOptionsForUse(context.Background(), secretChecker, attacker, nil, nil); err == nil {
		t.Errorf("opened for a precheck in another user's scope: %v", opts)
	}
	if merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(context.Background(), "gone_checker", victim, nil, nil, nil); err == nil {
		t.Errorf("opened under another checker: %v", merged)
	}

	report, err := checkerUC.NewCheckerOptionsSecrets(store, plaintext).ResealSecrets(context.Background())
	if err != nil || report.Changed != 1 || report.Failed != 2 {
		t.Errorf("ResealSecrets = %+v, %v; want the victim resealed, 2 failures", report, err)
	}
	if v := storedString(t, store.optionsStore, attacker, "user_token"); v != sealed {
		t.Errorf("a failed reseal changed the copy: %q", v)
	}
	if v := storedString(t, store.optionsStore, victim, "user_token"); v != "victim-token" {
		t.Errorf("victim option after reseal = %q, want it in clear", v)
	}
}

// The placeholder with nothing stored behind it stands for no value: it is
// dropped, never stored.
func TestPlaceholderWithNothingStoredIsDropped(t *testing.T) {
	uc, store := instanceOptionsUC(t)
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

// The context of the request reaches the secrets: a cancelled one neither
// seals nor opens an option.
func TestSecretOptionsHonourContext(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.SetCheckerOptions(cancelled, secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"}); !errors.Is(err, context.Canceled) {
		t.Errorf("SetCheckerOptions: got %v, want context.Canceled", err)
	}
	if _, err := uc.AddCheckerOptions(cancelled, secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"}); !errors.Is(err, context.Canceled) {
		t.Errorf("AddCheckerOptions: got %v, want context.Canceled", err)
	}
	if err := uc.SetCheckerOption(cancelled, secretChecker, user, nil, nil, "user_token", "t"); !errors.Is(err, context.Canceled) {
		t.Errorf("SetCheckerOption: got %v, want context.Canceled", err)
	}
	if storedString(t, store, user, "user_token") != "" {
		t.Error("an option was stored despite the cancelled context")
	}
	plaintext, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := checkerUC.RestoreOptions(cancelled, plaintext, secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"}); !errors.Is(err, context.Canceled) {
		t.Errorf("RestoreOptions: got %v, want context.Canceled", err)
	}

	if err := uc.SetCheckerOptions(context.Background(), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"}); err != nil {
		t.Fatalf("SetCheckerOptions: %v", err)
	}
	if _, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(cancelled, secretChecker, user, nil, nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("BuildMergedCheckerOptionsWithAutoFill: got %v, want context.Canceled", err)
	}
	if _, err := uc.GetCheckerOptionsForUse(cancelled, secretChecker, user, nil, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("GetCheckerOptionsForUse: got %v, want context.Canceled", err)
	}
}

// Options are used by scheduled runs: they never go to a keystore that needs
// their owner, chosen by the request or default.
func TestSecretOptionsKeystoreIsUnattended(t *testing.T) {
	ctx := context.Background()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	safes := secrettest.NewSafes()
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes, Defaults: secrettest.NewDefaults()}))
	if err != nil {
		t.Fatal(err)
	}
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(m)
	user := idPtr()

	personal, err := m.CreateKeystore(ctx, *user, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetDefaultKeystore(ctx, *user, personal.Ref); err != nil {
		t.Fatal(err)
	}

	err = uc.SetCheckerOptions(secret.WithKeystore(ctx, personal.Ref), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"})
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("SetCheckerOptions(personal keystore) = %v, want a ValidationError", err)
	}

	if err := uc.SetCheckerOptions(ctx, secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "t"}); err != nil {
		t.Fatal(err)
	}
	id, ok := secret.SafeOfValue(storedString(t, store, user, "user_token"))
	if !ok || id.Equals(personal.SafeId) {
		t.Errorf("an option went to the default keystore, which needs its owner")
	}
}

// A reference its keystore refuses is the owner's to fix: they are told why.
func TestSecretOptionsUnresolvedReference(t *testing.T) {
	ctx := context.Background()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithReferenceStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: secrettest.NewSafes(), Defaults: secrettest.NewDefaults()}))
	if err != nil {
		t.Fatal(err)
	}
	uc := checkerUC.NewCheckerOptionsUsecase(newOptionsStore(), nil).WithSecrets(m)
	user := idPtr()

	refs, err := m.CreateKeystore(ctx, *user, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(map[string]string{"token": "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.SetCheckerOptions(secret.WithKeystore(ctx, refs.Ref), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "token"}); err != nil {
		t.Fatalf("SetCheckerOptions(known reference) = %v", err)
	}
	err = uc.SetCheckerOptions(secret.WithKeystore(ctx, refs.Ref), secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "nope"})
	var verr happydns.ValidationError
	if !errors.As(err, &verr) || !strings.Contains(err.Error(), "unknown reference") || !strings.Contains(err.Error(), `"user_token"`) {
		t.Errorf("SetCheckerOptions(unknown reference) = %v, want a ValidationError naming the option, with the reason of the keystore", err)
	}
}

// Options in clear from a backup are sealed under the policy, not into the
// default keystore of their owner, which may only keep references.
func TestRestoreOptionsIgnoresTheDefault(t *testing.T) {
	ctx := context.Background()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithReferenceStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: secrettest.NewSafes(), Defaults: secrettest.NewDefaults()}))
	if err != nil {
		t.Fatal(err)
	}
	user := idPtr()

	refs, err := m.CreateKeystore(ctx, *user, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(map[string]string{"token": "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetDefaultKeystore(ctx, *user, refs.Ref); err != nil {
		t.Fatal(err)
	}

	opts, err := checkerUC.RestoreOptions(ctx, m, secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "s3cr3t"})
	if err != nil {
		t.Fatalf("RestoreOptions: %v", err)
	}
	v, _ := opts["user_token"].(string)
	id, ok := secret.SafeOfValue(v)
	if !ok || id.Equals(refs.SafeId) {
		t.Errorf("RestoreOptions = %q, want it sealed under the policy", v)
	}
}
