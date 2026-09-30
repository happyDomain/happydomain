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
	"reflect"
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
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

// optionsManagers returns two managers sharing one instance key and safes,
// one under the instance policy and one under the plaintext policy.
func optionsManagers(t *testing.T, safes secret.SafeStorage, owners secret.OwnerStorage) (instance, plaintext *secret.Manager) {
	t.Helper()
	h, err := secret.GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	key, err := secret.NewInstanceKey(h)
	if err != nil {
		t.Fatal(err)
	}
	instance, err = secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: safes, Owners: owners})
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err = secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: safes, Owners: owners})
	if err != nil {
		t.Fatal(err)
	}
	return instance, plaintext
}

func instanceOptionsManager(t *testing.T) *secret.Manager {
	t.Helper()
	m, _ := optionsManagers(t, secrettest.NewSafes(), nil)
	return m
}

func instanceOptionsUC(t *testing.T) (*checkerUC.CheckerOptionsUsecase, *optionsStore) {
	t.Helper()
	store := newOptionsStore()
	return checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(instanceOptionsManager(t)), store
}

func storedString(t *testing.T, store *optionsStore, userId *happydns.Identifier, key string) string {
	t.Helper()
	v, _ := store.data[posKey(secretChecker, userId, nil, nil)][key].(string)
	return v
}

func TestSecretOptionsAreSealedAndOpenedForRuns(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()

	if err := uc.SetCheckerOptions(secretChecker, nil, nil, nil, happydns.CheckerOptions{"api_key": "operator-key"}); err != nil {
		t.Fatalf("SetCheckerOptions(admin): %v", err)
	}
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "user-token", "plain": "visible"}); err != nil {
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

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, nil)
	if err != nil {
		t.Fatalf("BuildMergedCheckerOptionsWithAutoFill: %v", err)
	}
	if merged["api_key"] != "operator-key" || merged["user_token"] != "user-token" || merged["plain"] != "visible" {
		t.Errorf("merged = %v, want the secrets in clear", merged)
	}

	forUse, err := uc.GetCheckerOptionsForUse(secretChecker, user, nil, nil)
	if err != nil || forUse["user_token"] != "user-token" {
		t.Errorf("GetCheckerOptionsForUse = %v, %v; want the secrets in clear", forUse, err)
	}
}

func TestSecretOptionsLegacyPlaintextStillWork(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "legacy"}

	merged, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, nil)
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

// A checker this build does not load has no documentation telling which of
// its options are secret: every option set is withheld rather than guessed.
func TestRedactOptionsOfUnknownChecker(t *testing.T) {
	out := checkerUC.RedactOptions("checker_not_loaded", happydns.CheckerOptions{
		"api_key": "legacy-clear",
		"count":   3,
		"empty":   "",
	})
	if out["api_key"] != happydns.RedactedSecret || out["count"] != happydns.RedactedSecret {
		t.Errorf("redacted = %v, want every option set withheld", out)
	}
	if out["empty"] != "" {
		t.Errorf("empty = %v, want an unset option left unset", out["empty"])
	}
}

// An option the documentation of a loaded checker no longer describes, renamed
// or dropped since it was stored, may have been a secret stored in clear under
// the plaintext policy: it is withheld like the options of a checker that is
// not loaded.
func TestRedactOptionsWithholdsUndocumentedOptions(t *testing.T) {
	uc, _ := instanceOptionsUC(t)

	out := checkerUC.RedactOptions(secretChecker, happydns.CheckerOptions{
		"old_token": "legacy-clear",
		"old_count": 3,
		"plain":     "visible",
		"old_empty": "",
	})
	if out["old_token"] != happydns.RedactedSecret || out["old_count"] != happydns.RedactedSecret {
		t.Errorf("redacted = %v, want the undocumented options withheld", out)
	}
	if out["plain"] != "visible" || out["old_empty"] != "" {
		t.Errorf("redacted = %v, want documented and unset options as is", out)
	}

	if v := uc.RedactCheckerOptionValue(secretChecker, "old_token", "legacy-clear"); v != happydns.RedactedSecret {
		t.Errorf("RedactCheckerOptionValue(old_token) = %v, want it withheld", v)
	}
	if v := uc.RedactCheckerOptionValue(secretChecker, "plain", "visible"); v != "visible" {
		t.Errorf("RedactCheckerOptionValue(plain) = %v, want it as is", v)
	}
}

// RedactOptions gives a copy whatever the checker, and nil for nil.
func TestRedactOptionsCopies(t *testing.T) {
	for _, name := range []string{secretChecker, "checker_not_loaded"} {
		if out := checkerUC.RedactOptions(name, nil); out != nil {
			t.Errorf("RedactOptions(%s, nil) = %#v, want nil", name, out)
		}
	}

	in := happydns.CheckerOptions{"plain": "visible"}
	out := checkerUC.RedactOptions(secretChecker, in)
	out["plain"] = "changed"
	if in["plain"] != "visible" {
		t.Error("changing what RedactOptions returned changed its input")
	}
}

func TestSecretOptionsEchoKeepsStoredValue(t *testing.T) {
	uc, store := instanceOptionsUC(t)
	user := idPtr()

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "kept", "plain": "a"}); err != nil {
		t.Fatal(err)
	}
	sealed := storedString(t, store, user, "user_token")

	// Full replace echoing the placeholder.
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed || storedString(t, store, user, "plain") != "b" {
		t.Errorf("after full replace: %v", store.data[posKey(secretChecker, user, nil, nil)])
	}

	// Partial merge echoing the placeholder.
	if _, err := uc.AddCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "c"}); err != nil {
		t.Fatalf("AddCheckerOptions(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed {
		t.Error("the merge replaced the stored secret by the placeholder")
	}

	// Single option echoing the placeholder.
	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "user_token", happydns.RedactedSecret); err != nil {
		t.Fatalf("SetCheckerOption(echo): %v", err)
	}
	if storedString(t, store, user, "user_token") != sealed {
		t.Error("setting the placeholder replaced the stored secret")
	}

	// A new value replaces it.
	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "user_token", "rotated"); err != nil {
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
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": 42.0}); err == nil {
		t.Error("a non-string secret option was stored")
	}

	sealed := "hds:1:AQ:c2VhbGVk"
	var verr happydns.ValidationError
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": sealed}); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOptions(sealed from client) = %v, want a ValidationError", err)
	}
	if _, err := uc.MergeCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": sealed}); !errors.As(err, &verr) {
		t.Errorf("MergeCheckerOptions(sealed from client) = %v, want a ValidationError", err)
	}
	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "user_token", sealed); !errors.As(err, &verr) {
		t.Errorf("SetCheckerOption(sealed from client) = %v, want a ValidationError", err)
	}
}

func TestSecretOptionsFailClosedWithoutManager(t *testing.T) {
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil)
	user := idPtr()

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "v"}); err == nil {
		t.Error("a secret option was stored without a secret manager")
	}
	// Options without secrets need none.
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": "v"}); err != nil {
		t.Errorf("SetCheckerOptions(plain) = %v", err)
	}

	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "hds:1:AQ:c2VhbGVk"}
	if _, _, err := uc.BuildMergedCheckerOptionsWithAutoFill(secretChecker, user, nil, nil, nil); err == nil {
		t.Error("a run got options with a secret it could not open")
	}
}

func TestSecretOptionsResealAndInspect(t *testing.T) {
	instance, plaintext := optionsManagers(t, secrettest.NewSafes(), nil)

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

// knownUsers answers for the users that exist.
type knownUsers map[string]bool

func (k knownUsers) GetUser(id happydns.Identifier) (*happydns.User, error) {
	if k[id.String()] {
		return &happydns.User{Id: id}, nil
	}
	return nil, happydns.ErrUserNotFound
}

// A scope that fails, or whose user was deleted, keeps neither the others
// from being resealed nor the status from being told.
func TestSecretOptionsResealGoesPastWhatItCannotReseal(t *testing.T) {
	user, gone := idPtr(), idPtr()
	instance, plaintext := optionsManagers(t, secrettest.NewSafes(), knownUsers{user.String(): true})

	store := newListableOptionsStore()
	store.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "legacy"}
	store.data[posKey(secretChecker, gone, nil, nil)] = happydns.CheckerOptions{"user_token": "orphan"}

	report, err := checkerUC.NewCheckerOptionsSecrets(store, instance).ResealSecrets(context.Background())
	if err != nil || report.Changed != 1 || report.Skipped != 1 || report.Failed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want 1 resealed, the orphan skipped", report, err)
	}
	if v := storedString(t, store.optionsStore, gone, "user_token"); v != "orphan" {
		t.Errorf("orphan option = %q, want it left as it was", v)
	}

	// A value that no safe opens is lost already: it is left as it is, and
	// the others go back to clear.
	delete(store.data, posKey(secretChecker, gone, nil, nil))
	store.data[posKey(secretChecker, nil, nil, nil)] = happydns.CheckerOptions{"api_key": "hds:1:AQ:c2VhbGVk"}

	holder := checkerUC.NewCheckerOptionsSecrets(store, plaintext)
	counts, err := holder.InspectSecrets(context.Background())
	if err != nil || counts.Unreadable != 1 || counts.Sealed[secret.KindInstance] != 1 {
		t.Errorf("InspectSecrets = %+v, %v; want 1 sealed, 1 unreadable", counts, err)
	}
	report, err = holder.ResealSecrets(context.Background())
	if err != nil || report.Changed != 1 || report.Failed != 0 {
		t.Errorf("ResealSecrets(plaintext) = %+v, %v; want 1 back in clear, no failure", report, err)
	}
	if v := storedString(t, store.optionsStore, user, "user_token"); v != "legacy" {
		t.Errorf("user option = %q, want it back in clear", v)
	}
	if v := storedString(t, store.optionsStore, nil, "api_key"); v != "hds:1:AQ:c2VhbGVk" {
		t.Errorf("lost option = %q, want it left as it was", v)
	}
}

// racingOptions runs during once, right after the next read of the options
// of a scope and before whatever is written from that read: another writer
// landing in between, however the reader reads.
type racingOptions struct {
	*listableOptionsStore
	during func()
}

func (s *racingOptions) run() {
	if s.during != nil {
		during := s.during
		s.during = nil
		during()
	}
}

func (s *racingOptions) GetCheckerConfiguration(checkerName string, userId, domainId, serviceId *happydns.Identifier) ([]*happydns.CheckerOptionsPositional, error) {
	positionals, err := s.optionsStore.GetCheckerConfiguration(checkerName, userId, domainId, serviceId)
	s.run()
	return positionals, err
}

func (s *racingOptions) ReplaceCheckerConfiguration(checkerName string, userId, domainId, serviceId *happydns.Identifier, update func(happydns.CheckerOptions) (happydns.CheckerOptions, error)) error {
	return s.optionsStore.ReplaceCheckerConfiguration(checkerName, userId, domainId, serviceId, func(opts happydns.CheckerOptions) (happydns.CheckerOptions, error) {
		s.run()
		return update(opts)
	})
}

// A user save carries stored secrets forward. Written over what a reseal
// stored in between, it would bring back a token of a safe the reseal just
// emptied, and that may be dropped next: the save is refused instead, by
// each way of writing options.
func TestSecretOptionsSaveDoesNotOverwriteAConcurrentWrite(t *testing.T) {
	writes := map[string]func(uc *checkerUC.CheckerOptionsUsecase, user *happydns.Identifier) error{
		"SetCheckerOptions": func(uc *checkerUC.CheckerOptionsUsecase, user *happydns.Identifier) error {
			return uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "b"})
		},
		"AddCheckerOptions": func(uc *checkerUC.CheckerOptionsUsecase, user *happydns.Identifier) error {
			_, err := uc.AddCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"plain": "b"})
			return err
		},
		"SetCheckerOption": func(uc *checkerUC.CheckerOptionsUsecase, user *happydns.Identifier) error {
			return uc.SetCheckerOption(secretChecker, user, nil, nil, "plain", "b")
		},
	}

	for name, write := range writes {
		t.Run(name, func(t *testing.T) {
			store := newListableOptionsStore()
			racing := &racingOptions{listableOptionsStore: store}
			uc := checkerUC.NewCheckerOptionsUsecase(racing, nil).WithSecrets(instanceOptionsManager(t))
			user := idPtr()
			if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "kept", "plain": "a"}); err != nil {
				t.Fatal(err)
			}

			meanwhile := happydns.CheckerOptions{"user_token": "written-meanwhile", "plain": "a"}
			racing.during = func() {
				store.data[posKey(secretChecker, user, nil, nil)] = meanwhile
			}

			err := write(uc, user)
			var conflict happydns.ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("%s = %v, want a ConflictError", name, err)
			}
			if got := store.data[posKey(secretChecker, user, nil, nil)]; !reflect.DeepEqual(got, meanwhile) {
				t.Errorf("stored = %v, want what was written meanwhile", got)
			}

			// Retrying reads again.
			if err := write(uc, user); err != nil {
				t.Fatalf("retried %s: %v", name, err)
			}
			if v := storedString(t, store.optionsStore, user, "plain"); v != "b" {
				t.Errorf("plain after retry = %q", v)
			}
		})
	}
}

// A user saving their options while they are resealed keeps what they saved.
func TestSecretOptionsResealLosesToConcurrentWrites(t *testing.T) {
	instance := instanceOptionsManager(t)

	user := idPtr()
	listable := newListableOptionsStore()
	listable.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "legacy"}
	store := &racingOptions{listableOptionsStore: listable, during: func() {
		listable.data[posKey(secretChecker, user, nil, nil)] = happydns.CheckerOptions{"user_token": "hds:1:AQ:c2F2ZWQ"}
	}}

	report, err := checkerUC.NewCheckerOptionsSecrets(store, instance).ResealSecrets(context.Background())
	if err != nil || report.Skipped != 1 || report.Changed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want the scope skipped", report, err)
	}
	if v := storedString(t, listable.optionsStore, user, "user_token"); v != "hds:1:AQ:c2F2ZWQ" {
		t.Errorf("stored = %q, want the user's save kept", v)
	}
}

// A stored secret option that no longer opens, its safe gone, does not make
// every later save an error: stored single values are kept as they are
// without being opened, and entering the secret again repairs the options.
func TestSecretOptionsSaveWithAStoredValueThatNoLongerOpens(t *testing.T) {
	safes := secrettest.NewSafes()
	m, _ := optionsManagers(t, safes, nil)
	store := newOptionsStore()
	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(m)
	user := idPtr()

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "lost", "plain": "a"}); err != nil {
		t.Fatal(err)
	}
	secrettest.DeleteSafeOf(t, safes, *user, secret.KindInstance)

	if err := uc.SetCheckerOption(secretChecker, user, nil, nil, "plain", "b"); err != nil {
		t.Fatalf("SetCheckerOption keeping it = %v", err)
	}

	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "entered-again", "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions entering it again = %v", err)
	}
	forUse, err := uc.GetCheckerOptionsForUse(secretChecker, user, nil, nil)
	if err != nil || forUse["user_token"] != "entered-again" {
		t.Errorf("GetCheckerOptionsForUse = %v, %v; want the value entered again", forUse, err)
	}
}

// Under the plaintext policy, a secret option still sealed is stored in clear
// on the next save of its scope, as a reseal would do.
func TestSecretOptionsPlaintextPolicyStoresSealedInClearOnNextSave(t *testing.T) {
	instance, plaintext := optionsManagers(t, secrettest.NewSafes(), nil)
	store := newOptionsStore()
	user := idPtr()

	if err := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(instance).SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": "kept", "plain": "a"}); err != nil {
		t.Fatal(err)
	}
	if v := storedString(t, store, user, "user_token"); !strings.HasPrefix(v, "hds:1:") {
		t.Fatalf("stored = %q, want it sealed", v)
	}

	uc := checkerUC.NewCheckerOptionsUsecase(store, nil).WithSecrets(plaintext)
	if err := uc.SetCheckerOptions(secretChecker, user, nil, nil, happydns.CheckerOptions{"user_token": happydns.RedactedSecret, "plain": "b"}); err != nil {
		t.Fatalf("SetCheckerOptions(plaintext): %v", err)
	}
	if v := storedString(t, store, user, "user_token"); v != "kept" {
		t.Errorf("stored = %q, want the option in clear", v)
	}
}
