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

package provider_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/secret/secrettest"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	kv "git.happydns.org/happyDomain/internal/storage/kvtpl"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

// SecretTestProvider holds a happydns.Secret, as every provider will. It
// records the credential it was instantiated with.
type SecretTestProvider struct {
	Host   string          `json:"host"`
	ApiKey happydns.Secret `json:"apikey" happydomain:"label=API Key,required"`
}

// instantiatedWith is what the last instantiation received, or nil when none
// happened.
var instantiatedWith *string

func (p *SecretTestProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
	v := p.ApiKey.Reveal()
	instantiatedWith = &v
	return &fakeActuator{}, nil
}

// OtherSecretTestProvider is a second type, to change a provider's type.
type OtherSecretTestProvider struct {
	ApiKey happydns.Secret `json:"apikey"`
}

func (p *OtherSecretTestProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
	return &fakeActuator{}, nil
}

type fakeActuator struct{}

func (fakeActuator) CanCreateDomain() bool     { return false }
func (fakeActuator) CanListZones() bool        { return true }
func (fakeActuator) CreateDomain(string) error { return nil }
func (fakeActuator) GetZoneRecords(string) ([]happydns.Record, error) {
	return nil, nil
}
func (fakeActuator) GetZoneCorrections(string, []happydns.Record) ([]*happydns.Correction, int, error) {
	return nil, 0, nil
}
func (fakeActuator) ListZones() ([]string, error) { return nil, nil }

func init() {
	providerReg.RegisterProvider(func() happydns.ProviderBody { return &SecretTestProvider{} }, happydns.ProviderInfos{Name: "secret test"})
	providerReg.RegisterProvider(func() happydns.ProviderBody { return &OtherSecretTestProvider{} }, happydns.ProviderInfos{Name: "other secret test"})
}

func plaintextSecrets(t *testing.T) *secret.Manager {
	t.Helper()
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newSecretTestService(t *testing.T) (*provider.Service, storage.Storage) {
	t.Helper()
	db, _ := inmemory.Instantiate()
	return provider.NewService(db, &mockValidator{}, nil, plaintextSecrets(t)), db
}

func secretMessage(t *testing.T, providerType, body string) *happydns.ProviderMessage {
	t.Helper()
	return &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{Type: providerType, Comment: "secret test"},
		Provider:     json.RawMessage(body),
	}
}

func storedBody(t *testing.T, db storage.Storage, id happydns.Identifier) string {
	t.Helper()
	msg, err := db.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	return string(msg.Provider)
}

func Test_Secret_CreateStoresTodaysFormat(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	if got := storedBody(t, db, p.Id); got != `{"host":"h","apikey":"my-key"}` {
		t.Errorf("stored body = %s, want the format of today", got)
	}
}

func Test_Secret_UpdateWithNewValue(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"old"}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h2","apikey":"new"}`)); err != nil {
		t.Fatalf("UpdateProviderFromMessage: %v", err)
	}

	if got := storedBody(t, db, p.Id); got != `{"host":"h2","apikey":"new"}` {
		t.Errorf("stored body = %s", got)
	}
}

func Test_Secret_UpdateEchoingRedacted(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"kept"}`))
	if err != nil {
		t.Fatal(err)
	}

	body := `{"host":"h2","apikey":"` + happydns.RedactedSecret + `"}`
	if _, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", body)); err != nil {
		t.Fatalf("UpdateProviderFromMessage: %v", err)
	}

	if got := storedBody(t, db, p.Id); got != `{"host":"h2","apikey":"kept"}` {
		t.Errorf("stored body = %s, want the stored key carried forward", got)
	}
}

func Test_Secret_UpdateChangingType(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"kept"}`))
	if err != nil {
		t.Fatal(err)
	}

	body := `{"apikey":"` + happydns.RedactedSecret + `"}`
	if _, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "OtherSecretTestProvider", body)); err != nil {
		t.Fatalf("UpdateProviderFromMessage: %v", err)
	}

	// Another type: nothing is carried forward.
	if got := storedBody(t, db, p.Id); got != `{"apikey":""}` {
		t.Errorf("stored body = %s, want the placeholder cleared", got)
	}
}

// Secrets are bound to their owner: an update moving a provider to someone
// else is refused, and leaves the stored record untouched.
func Test_Secret_UpdateCannotChangeOwner(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"kept"}`))
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateProvider(ctx, p.Id, user, func(p *happydns.Provider) {
		p.Owner = happydns.Identifier{0xee}
		p.Comment = "stolen"
	})
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("UpdateProvider changing the owner = %v, want a ValidationError", err)
	}

	stored, err := db.GetProvider(p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Owner.Equals(user.Id) || stored.Comment == "stolen" {
		t.Errorf("stored provider = owner %s, comment %q; want it unchanged", stored.Owner.String(), stored.Comment)
	}
	if got := storedBody(t, db, p.Id); got != `{"host":"h","apikey":"kept"}` {
		t.Errorf("stored body = %s, want it unchanged", got)
	}
}

func Test_Secret_SealedValueFromClientRefused(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")

	sealed := `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`

	_, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", sealed))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("CreateProvider(sealed) = %v, want a ValidationError", err)
	}

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"v"}`))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", sealed))
	if !errors.As(err, &verr) {
		t.Errorf("UpdateProviderFromMessage(sealed) = %v, want a ValidationError", err)
	}
	if got := storedBody(t, db, p.Id); got != `{"host":"h","apikey":"v"}` {
		t.Errorf("stored body = %s, want it unchanged", got)
	}
}

// storeRaw writes a provider record the way an older happyDomain, or another
// process, left it.
func storeRaw(t *testing.T, db storage.Storage, owner happydns.Identifier, body string) *happydns.Provider {
	t.Helper()

	id, _ := happydns.NewRandomIdentifier()
	p, err := provider.ParseProvider(&happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{Type: "SecretTestProvider", Id: id, Owner: owner},
		Provider:     json.RawMessage(body),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Seal as the use cases do, so the storage accepts it.
	if err := plaintextSecrets(t).SealObject(ctx, secret.SecretContext{Owner: owner, ObjectType: "provider", ObjectId: id.String()}, p.Provider); err != nil {
		// A sealed value is left as is, so this only fails for a bad body.
		t.Fatal(err)
	}
	if err := db.UpdateProvider(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func Test_Secret_InstantiateOpensLegacyPlaintext(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")
	stored := storeRaw(t, db, user.Id, `{"host":"h","apikey":"legacy"}`)

	p, err := svc.GetUserProvider(ctx, user, stored.Id)
	if err != nil {
		t.Fatal(err)
	}

	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, p, "example.com"); err != nil {
		t.Fatalf("RetrieveZone: %v", err)
	}
	if instantiatedWith == nil || *instantiatedWith != "legacy" {
		t.Errorf("instantiated with %v, want the legacy value", instantiatedWith)
	}
}

func Test_Secret_InstantiateFailsClosedOnUnopenable(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")
	stored := storeRaw(t, db, user.Id, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`)

	p, err := svc.GetUserProvider(ctx, user, stored.Id)
	if err != nil {
		t.Fatal(err)
	}

	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, p, "example.com"); !errors.Is(err, secret.ErrUnknownSafe) {
		t.Errorf("RetrieveZone = %v, want ErrUnknownSafe", err)
	}
	if instantiatedWith != nil {
		t.Error("the provider was instantiated with a secret that could not be opened")
	}

	// The provider handed around keeps its sealed value.
	if body := p.Provider.(*SecretTestProvider); !body.ApiKey.IsSealed() {
		t.Error("instantiating opened the provider in place")
	}
}

// A provider whose secret no longer opens only fails when connecting: the
// other providers of its owner go on working, and it can still be listed,
// read and deleted.
func Test_Secret_UnopenableLeavesTheOthersUsable(t *testing.T) {
	svc, db := newSecretTestService(t)
	user := createTestUser(t, db, "secret@example.com")
	broken := storeRaw(t, db, user.Id, `{"host":"h","apikey":"hds:1:AQ:c2VhbGVk"}`)
	working := storeRaw(t, db, user.Id, `{"host":"h","apikey":"legacy"}`)

	metas, err := svc.ListUserProviders(ctx, user)
	if err != nil {
		t.Fatalf("ListUserProviders: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("ListUserProviders returned %d providers, want 2", len(metas))
	}

	if _, err := svc.GetUserProviderMeta(ctx, user, broken.Id); err != nil {
		t.Errorf("GetUserProviderMeta of the broken provider: %v", err)
	}
	p, err := svc.GetUserProvider(ctx, user, broken.Id)
	if err != nil {
		t.Fatalf("GetUserProvider of the broken provider: %v", err)
	}
	if _, err := svc.RetrieveZone(ctx, p, "example.com"); !errors.Is(err, secret.ErrUnknownSafe) {
		t.Errorf("RetrieveZone on the broken provider = %v, want ErrUnknownSafe", err)
	}

	p, err = svc.GetUserProvider(ctx, user, working.Id)
	if err != nil {
		t.Fatalf("GetUserProvider of the working provider: %v", err)
	}
	if _, err := svc.RetrieveZone(ctx, p, "example.com"); err != nil {
		t.Errorf("RetrieveZone on the working provider: %v", err)
	}

	if err := svc.DeleteProvider(ctx, user, broken.Id); err != nil {
		t.Errorf("DeleteProvider of the broken provider: %v", err)
	}
}

func Test_Secret_ValidatorOpensBeforeInstantiating(t *testing.T) {
	owner := happydns.Identifier{0x01}
	id := happydns.Identifier{0x02}

	var sealed happydns.Secret
	if err := json.Unmarshal([]byte(`"hds:1:AQ:c2VhbGVk"`), &sealed); err != nil {
		t.Fatal(err)
	}

	p := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "SecretTestProvider", Id: id, Owner: owner},
		Provider:     &SecretTestProvider{Host: "h", ApiKey: sealed},
	}

	instantiatedWith = nil
	err := provider.NewValidator(nil, plaintextSecrets(t)).Validate(ctx, p)
	if !errors.Is(err, secret.ErrUnknownSafe) {
		t.Errorf("Validate = %v, want ErrUnknownSafe", err)
	}
	if instantiatedWith != nil {
		t.Error("the validator instantiated a provider it could not open")
	}

	p.Provider = &SecretTestProvider{Host: "h", ApiKey: happydns.NewSecret("clear")}
	if err := provider.NewValidator(nil, plaintextSecrets(t)).Validate(ctx, p); err != nil {
		t.Fatalf("Validate(clear) = %v", err)
	}
	if instantiatedWith == nil || *instantiatedWith != "clear" {
		t.Errorf("instantiated with %v, want the clear value", instantiatedWith)
	}
}

func Test_Secret_StorageFailsClosed(t *testing.T) {
	db, _ := inmemory.Instantiate()
	owner := happydns.Identifier{0x01}

	clear := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{Type: "SecretTestProvider", Owner: owner},
		Provider:     &SecretTestProvider{Host: "h", ApiKey: happydns.NewSecret("clear")},
	}

	if err := db.CreateProvider(clear); !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Errorf("CreateProvider(clear) = %v, want ErrUnsealedSecret", err)
	}

	clear.Id = happydns.Identifier{0x02}
	if err := db.UpdateProvider(clear); !errors.Is(err, happydns.ErrUnsealedSecret) {
		t.Errorf("UpdateProvider(clear) = %v, want ErrUnsealedSecret", err)
	}

	if n, _ := db.CountProviders(); n != 0 {
		t.Errorf("%d providers stored, want none", n)
	}
	if list, _ := db.ListProviders(&happydns.User{Id: owner}); len(list) != 0 {
		t.Errorf("%d providers indexed, want none", len(list))
	}
}

func instanceSecrets(t *testing.T, db storage.Storage) *secret.Manager {
	t.Helper()
	h, err := secret.GenerateInstanceKeyset()
	if err != nil {
		t.Fatal(err)
	}
	key, err := secret.NewInstanceKey(h)
	if err != nil {
		t.Fatal(err)
	}
	m, err := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func Test_Secret_InstancePolicyNeverStoresClear(t *testing.T) {
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(db, &mockValidator{}, nil, instanceSecrets(t, db))
	user := createTestUser(t, db, "instance@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}

	body := storedBody(t, db, p.Id)
	if strings.Contains(body, "my-key") || !strings.Contains(body, `"apikey":"hds:1:`) {
		t.Fatalf("stored body = %s, want the key sealed", body)
	}

	stored, err := svc.GetUserProvider(ctx, user, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, stored, "example.com"); err != nil {
		t.Fatalf("RetrieveZone: %v", err)
	}
	if instantiatedWith == nil || *instantiatedWith != "my-key" {
		t.Errorf("instantiated with %v, want the clear key", instantiatedWith)
	}
}

// A value stored in clear before the instance policy was turned on is sealed
// the next time its provider is written.
func Test_Secret_LegacyPlaintextSealedOnNextWrite(t *testing.T) {
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(db, &mockValidator{}, nil, instanceSecrets(t, db))
	user := createTestUser(t, db, "lazy@example.com")
	legacy := storeRaw(t, db, user.Id, `{"host":"h","apikey":"legacy"}`)

	body := `{"host":"renamed","apikey":"` + happydns.RedactedSecret + `"}`
	if _, err := svc.UpdateProviderFromMessage(ctx, legacy.Id, user, secretMessage(t, "SecretTestProvider", body)); err != nil {
		t.Fatalf("UpdateProviderFromMessage: %v", err)
	}

	got := storedBody(t, db, legacy.Id)
	if strings.Contains(got, "legacy") || !strings.Contains(got, `"apikey":"hds:1:`) {
		t.Fatalf("stored body = %s, want the legacy key sealed", got)
	}

	stored, _ := svc.GetUserProvider(ctx, user, legacy.Id)
	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, stored, "example.com"); err != nil {
		t.Fatal(err)
	}
	if instantiatedWith == nil || *instantiatedWith != "legacy" {
		t.Errorf("instantiated with %v, want the legacy key", instantiatedWith)
	}
}

// An update gives back the provider as written: its callers answer with it
// rather than read it again. Encoded, it holds what is stored, never the
// credential in clear.
func Test_Secret_UpdateReturnsTheStoredProvider(t *testing.T) {
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(db, &mockValidator{}, nil, instanceSecrets(t, db))
	user := createTestUser(t, db, "returned@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"old"}`))
	if err != nil {
		t.Fatal(err)
	}

	msg := secretMessage(t, "SecretTestProvider", `{"host":"h2","apikey":"new-key"}`)
	msg.Comment = "renamed"
	updated, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, msg)
	if err != nil {
		t.Fatalf("UpdateProviderFromMessage: %v", err)
	}

	if !updated.Id.Equals(p.Id) || !updated.Owner.Equals(user.Id) || updated.Comment != "renamed" {
		t.Errorf("returned meta = %+v, want the updated provider", updated.ProviderMeta)
	}

	body, err := json.Marshal(updated.Provider)
	if err != nil {
		t.Fatalf("encoding the returned provider: %v", err)
	}
	if strings.Contains(string(body), "new-key") {
		t.Errorf("returned provider encodes the key in clear: %s", body)
	}
	if stored := storedBody(t, db, p.Id); string(body) != stored {
		t.Errorf("returned body = %s, want what is stored: %s", body, stored)
	}
}

// failingUpdates makes the nth UpdateProvider fail, as an interrupted reseal.
type failingUpdates struct {
	storage.Storage
	failAt int
	calls  int
}

func (s *failingUpdates) UpdateProvider(p *happydns.Provider) error {
	s.calls++
	if s.calls == s.failAt {
		return errors.New("storage unavailable")
	}
	return s.Storage.UpdateProvider(p)
}

func seedLegacyProviders(t *testing.T, db storage.Storage, n int) (*happydns.User, []happydns.Identifier) {
	t.Helper()
	user := createTestUser(t, db, "reseal@example.com")
	var ids []happydns.Identifier
	for i := range n {
		p := storeRaw(t, db, user.Id, `{"host":"h","apikey":"legacy-`+string(rune('a'+i))+`"}`)
		ids = append(ids, p.Id)
	}
	return user, ids
}

func Test_Secret_ResealProviders(t *testing.T) {
	db, _ := inmemory.Instantiate()
	_, ids := seedLegacyProviders(t, db, 3)

	instance := instanceSecrets(t, db)
	svc := provider.NewService(db, &mockValidator{}, nil, instance)

	counts, err := svc.InspectSecrets(ctx)
	if err != nil || counts.Clear != 3 {
		t.Fatalf("InspectProviders before = %+v, %v; want 3 clear", counts, err)
	}

	report, err := svc.ResealSecrets(ctx)
	if err != nil || report.Processed != 3 || report.Changed != 3 || report.Failed != 0 {
		t.Fatalf("ResealProviders = %+v, %v", report, err)
	}
	for _, id := range ids {
		if body := storedBody(t, db, id); strings.Contains(body, "legacy") {
			t.Errorf("provider still in clear: %s", body)
		}
	}

	counts, _ = svc.InspectSecrets(ctx)
	if counts.Clear != 0 || counts.Sealed[secret.KindInstance] != 3 || counts.Unreadable != 0 {
		t.Errorf("InspectProviders after = %+v, want 3 sealed", counts)
	}

	// Idempotent.
	if report, err := svc.ResealSecrets(ctx); err != nil || report.Changed != 0 {
		t.Errorf("second ResealProviders = %+v, %v; want nothing changed", report, err)
	}
}

func Test_Secret_ResealBackToPlaintext(t *testing.T) {
	db, _ := inmemory.Instantiate()
	_, ids := seedLegacyProviders(t, db, 2)

	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	instance, _ := secret.NewManager(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db})
	plaintext, _ := secret.NewManager(secret.Config{Policy: secret.PolicyPlaintext, InstanceKey: key, Safes: db})

	if _, err := provider.NewService(db, &mockValidator{}, nil, instance).ResealSecrets(ctx); err != nil {
		t.Fatal(err)
	}

	report, err := provider.NewService(db, &mockValidator{}, nil, plaintext).ResealSecrets(ctx)
	if err != nil || report.Changed != 2 {
		t.Fatalf("ResealProviders(plaintext) = %+v, %v", report, err)
	}
	for i, id := range ids {
		want := `{"host":"h","apikey":"legacy-` + string(rune('a'+i)) + `"}`
		if body := storedBody(t, db, id); body != want {
			t.Errorf("stored = %s, want %s", body, want)
		}
	}
}

func Test_Secret_ResealResumesAfterInterruption(t *testing.T) {
	db, _ := inmemory.Instantiate()
	_, ids := seedLegacyProviders(t, db, 3)
	instance := instanceSecrets(t, db)

	broken := &failingUpdates{Storage: db, failAt: 2}
	report, err := provider.NewService(broken, &mockValidator{}, nil, instance).ResealSecrets(ctx)
	if err != nil {
		t.Fatalf("ResealProviders: %v", err)
	}
	if report.Failed != 1 || report.Changed != 2 || len(report.Errors) != 1 {
		t.Errorf("report = %+v, want one failure reported and the others done", report)
	}

	report, err = provider.NewService(db, &mockValidator{}, nil, instance).ResealSecrets(ctx)
	if err != nil || report.Changed != 1 || report.Failed != 0 {
		t.Errorf("rerun = %+v, %v; want the remaining one done", report, err)
	}
	for _, id := range ids {
		if body := storedBody(t, db, id); strings.Contains(body, "legacy") {
			t.Errorf("provider still in clear after rerun: %s", body)
		}
	}
}

type slowReads struct {
	storage.Storage
}

func (s *slowReads) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	msg, err := s.Storage.GetProvider(id)
	time.Sleep(time.Millisecond)
	return msg, err
}

// A user saving a new key while an administrator reseals must not see it
// replaced by the value the reseal read before.
func Test_Secret_ResealDoesNotLoseConcurrentUpdates(t *testing.T) {
	db, _ := inmemory.Instantiate()
	user, ids := seedLegacyProviders(t, db, 1)
	id := ids[0]

	// The user side stores in clear, so that every reseal has work to do.
	userSvc := provider.NewService(db, &mockValidator{}, nil, plaintextSecrets(t))
	// Slow reads widen the window between the reseal reading a provider and
	// writing it back.
	adminSvc := provider.NewService(&slowReads{Storage: db}, &mockValidator{}, nil, instanceSecrets(t, db))

	const rounds = 50
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range rounds {
			body := fmt.Sprintf(`{"host":"h","apikey":"user-%d"}`, i)
			if _, err := userSvc.UpdateProviderFromMessage(ctx, id, user, secretMessage(t, "SecretTestProvider", body)); err != nil {
				t.Errorf("UpdateProviderFromMessage: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for range rounds {
			if _, err := adminSvc.ResealSecrets(ctx); err != nil {
				t.Errorf("ResealProviders: %v", err)
				return
			}
		}
	}()
	wg.Wait()

	p, err := adminSvc.GetUserProvider(ctx, user, id)
	if err != nil {
		t.Fatal(err)
	}
	instantiatedWith = nil
	if _, err := adminSvc.RetrieveZone(ctx, p, "example.com"); err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("user-%d", rounds-1); instantiatedWith == nil || *instantiatedWith != want {
		t.Errorf("stored key = %v, want the user's last value %q", instantiatedWith, want)
	}
}

// storeCorrupt writes, under a provider key, a record that does not decode.
func storeCorrupt(t *testing.T) storage.Storage {
	t.Helper()
	kvdb, err := inmemory.NewInMemoryStorage()
	if err != nil {
		t.Fatal(err)
	}
	db, err := kv.NewKVDatabase(kvdb)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := happydns.NewRandomIdentifier()
	if err := kvdb.Put("provider-"+id.String(), "not a provider"); err != nil {
		t.Fatal(err)
	}
	return db
}

// removedProvider is stored under a type name no registered provider has.
type removedProvider struct {
	ApiKey happydns.Secret `json:"apikey"`
}

func (*removedProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
	return nil, errors.New("removed")
}

// One record that cannot be looked at keeps neither the others from being
// resealed, nor the status from being reported. It is counted: it may hold
// sealed secrets.
func Test_Secret_OneUndecodableRecordDoesNotStopTheOthers(t *testing.T) {
	db := storeCorrupt(t)
	_, ids := seedLegacyProviders(t, db, 2)

	// A provider of a type this build no longer knows.
	gone, _ := happydns.NewRandomIdentifier()
	if err := db.UpdateProvider(&happydns.Provider{ProviderMeta: happydns.ProviderMeta{Type: "RemovedTestProvider", Id: gone, Owner: happydns.Identifier{0x02}}, Provider: &removedProvider{}}); err != nil {
		t.Fatal(err)
	}

	svc := provider.NewService(db, &mockValidator{}, nil, instanceSecrets(t, db))

	counts, err := svc.InspectSecrets(ctx)
	if err != nil {
		t.Fatalf("InspectSecrets: %v", err)
	}
	if counts.Clear != 2 || counts.Undecodable != 2 {
		t.Errorf("counts = %+v, want 2 clear and 2 undecodable", counts)
	}

	report, err := svc.ResealSecrets(ctx)
	if err != nil {
		t.Fatalf("ResealSecrets: %v", err)
	}
	if report.Processed != 4 || report.Changed != 2 || report.Failed != 2 {
		t.Errorf("report = %+v, want both readable providers resealed, 2 failures", report)
	}
	for _, id := range ids {
		if body := storedBody(t, db, id); strings.Contains(body, "legacy") {
			t.Errorf("provider still in clear: %s", body)
		}
	}
}

// A user deleting a provider while an administrator reseals must not see it
// written back with the body the reseal read before.
func Test_Secret_ResealDoesNotResurrectDeletedProviders(t *testing.T) {
	db, _ := inmemory.Instantiate()
	user, ids := seedLegacyProviders(t, db, 20)

	userSvc := provider.NewService(db, &mockValidator{}, nil, plaintextSecrets(t))
	adminSvc := provider.NewService(&slowReads{Storage: db}, &mockValidator{}, nil, instanceSecrets(t, db))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if report, err := adminSvc.ResealSecrets(ctx); err != nil || report.Failed != 0 {
			t.Errorf("ResealSecrets = %+v, %v; want no failure", report, err)
		}
	}()
	go func() {
		defer wg.Done()
		for _, id := range ids {
			if err := userSvc.DeleteProvider(ctx, user, id); err != nil {
				t.Errorf("DeleteProvider: %v", err)
			}
		}
	}()
	wg.Wait()

	for _, id := range ids {
		if _, err := db.GetProvider(id); !errors.Is(err, happydns.ErrProviderNotFound) {
			t.Errorf("provider %s came back after its deletion: %v", id, err)
		}
	}
}

// vanishing deletes each provider just before it is read, as a user deleting
// it between the listing and its reseal.
type vanishing struct{ storage.Storage }

func (s *vanishing) GetProvider(id happydns.Identifier) (*happydns.ProviderMessage, error) {
	if err := s.Storage.DeleteProvider(id); err != nil {
		return nil, err
	}
	return s.Storage.GetProvider(id)
}

func Test_Secret_ResealSkipsProvidersDeletedSinceListed(t *testing.T) {
	db, _ := inmemory.Instantiate()
	_, ids := seedLegacyProviders(t, db, 1)

	report, err := provider.NewService(&vanishing{Storage: db}, &mockValidator{}, nil, instanceSecrets(t, db)).ResealSecrets(ctx)
	if err != nil || report.Processed != 1 || report.Changed != 0 || report.Failed != 0 {
		t.Errorf("ResealSecrets = %+v, %v; want the deleted provider skipped, not failed", report, err)
	}
	if _, err := db.GetProvider(ids[0]); !errors.Is(err, happydns.ErrProviderNotFound) {
		t.Errorf("the deleted provider came back: %v", err)
	}
}

// The keystore a request chooses holds the credentials of the provider it
// writes, those already stored included; an unknown one is refused.
func Test_Secret_KeystoreChosenByRequest(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	svc := provider.NewService(db, &mockValidator{}, nil, m)
	user := createTestUser(t, db, "keystore@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatal(err)
	}
	instanceSafe, err := db.GetSafeByOwner(user.Id, secret.KindInstance)
	if err != nil {
		t.Fatal(err)
	}

	personal, err := m.CreateKeystore(ctx, user.Id, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.UpdateProviderFromMessage(secret.WithKeystore(ctx, "nope"), p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"`+happydns.RedactedSecret+`"}`))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("UpdateProviderFromMessage(unknown keystore) = %v, want a ValidationError", err)
	}

	if _, err := svc.UpdateProviderFromMessage(secret.WithKeystore(ctx, personal.Ref), p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"`+happydns.RedactedSecret+`"}`)); err != nil {
		t.Fatal(err)
	}
	body := storedBody(t, db, p.Id)
	if !strings.Contains(body, `"apikey":"hds:1:`+personal.Ref+`:`) || strings.Contains(body, instanceSafe.Id.String()) {
		t.Errorf("stored body = %s, want the key moved to %s", body, personal.Ref)
	}

	stored, err := svc.GetUserProvider(ctx, user, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, stored, "example.com"); err != nil || instantiatedWith == nil || *instantiatedWith != "my-key" {
		t.Errorf("RetrieveZone = %v, instantiated with %v; want the clear key", err, instantiatedWith)
	}
}

// Moving the secrets of an owner out of a keystore rewrites each of their
// providers there, and leaves those of other owners alone.
func Test_Secret_MoveOwnerSecrets(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	svc := provider.NewService(db, &mockValidator{}, nil, m)
	user := createTestUser(t, db, "mover@example.com")
	other := createTestUser(t, db, "stayer@example.com")

	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatal(err)
	}
	o, err := svc.CreateProvider(ctx, other, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"their-key"}`))
	if err != nil {
		t.Fatal(err)
	}
	otherBody := storedBody(t, db, o.Id)

	objects, err := svc.OwnerObjects(ctx, user.Id)
	if err != nil || len(objects) != 1 || objects[0].ObjectId != p.Id.String() || len(objects[0].Safes) != 1 {
		t.Fatalf("OwnerObjects = %+v, %v; want the provider and its safe", objects, err)
	}
	instanceSafe := objects[0].Safes[0]

	personal, err := m.CreateKeystore(ctx, user.Id, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	report, err := svc.MoveOwnerSecrets(ctx, user.Id, instanceSafe, personal.Ref)
	if err != nil || report.Processed != 1 || report.Changed != 1 || report.Failed != 0 {
		t.Fatalf("MoveOwnerSecrets = %+v, %v", report, err)
	}
	if body := storedBody(t, db, p.Id); !strings.Contains(body, `"apikey":"hds:1:`+personal.Ref+`:`) {
		t.Errorf("stored body = %s, want the key moved", body)
	}
	if storedBody(t, db, o.Id) != otherBody {
		t.Error("the provider of another owner changed")
	}

	stored, err := svc.GetUserProvider(ctx, user, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	instantiatedWith = nil
	if _, err := svc.RetrieveZone(ctx, stored, "example.com"); err != nil || instantiatedWith == nil || *instantiatedWith != "my-key" {
		t.Errorf("RetrieveZone = %v, instantiated with %v; want the clear key", err, instantiatedWith)
	}
}

// A provider in a keystore its owner has not unlocked during the session
// tells which one to unlock, whatever touches it.
func Test_Secret_LockedKeystore(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithInteractiveStores(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	svc := provider.NewService(db, &mockValidator{}, nil, m)
	user := createTestUser(t, db, "locked@example.com")

	ks, err := m.CreateKeystore(ctx, user.Id, secrettest.PassphraseKind, "Mine", secrettest.Material("passphrase", "p"))
	if err != nil {
		t.Fatal(err)
	}
	session := secret.WithUnlockScope(ctx, "s1")
	if err := m.UnlockKeystore(session, user.Id, ks.Ref, secrettest.Material("passphrase", "p")); err != nil {
		t.Fatal(err)
	}
	p, err := svc.CreateProvider(secret.WithKeystore(session, ks.Ref), user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatalf("CreateProvider(unlocked) = %v", err)
	}

	isLocked := func(err error) bool {
		var le happydns.KeystoreLockedError
		return errors.As(err, &le) && le.Ref == ks.Ref
	}

	other := secret.WithUnlockScope(ctx, "s2")
	stored, err := svc.GetUserProvider(other, user, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RetrieveZone(other, stored, "example.com"); !isLocked(err) {
		t.Errorf("RetrieveZone(other session) = %v, want the keystore to unlock", err)
	}

	// A new credential is sealed there: that needs it unlocked.
	body := `{"host":"h","apikey":"new-key"}`
	if _, err := svc.UpdateProviderFromMessage(other, p.Id, user, secretMessage(t, "SecretTestProvider", body)); !isLocked(err) {
		t.Errorf("UpdateProviderFromMessage(new key, other session) = %v, want the keystore to unlock", err)
	}
	if _, err := svc.UpdateProviderFromMessage(session, p.Id, user, secretMessage(t, "SecretTestProvider", body)); err != nil {
		t.Errorf("UpdateProviderFromMessage(new key, unlocked session) = %v", err)
	}

	// Renaming it opens nothing: no need to unlock it.
	renamed := `{"host":"renamed","apikey":"` + happydns.RedactedSecret + `"}`
	if _, err := svc.UpdateProviderFromMessage(other, p.Id, user, secretMessage(t, "SecretTestProvider", renamed)); err != nil {
		t.Errorf("UpdateProviderFromMessage(renamed, other session) = %v", err)
	}
}

// Replacing every credential of a provider keeps it in its keystore: the
// stored version tells where it is.
func Test_Secret_ReplacedCredentialsStayInTheirKeystore(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithPersonalStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	svc := provider.NewService(db, &mockValidator{}, nil, m)
	user := createTestUser(t, db, "replace@example.com")

	personal, err := m.CreateKeystore(ctx, user.Id, secrettest.PersonalKind, "Mine", secrettest.PersonalParams())
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.CreateProvider(secret.WithKeystore(ctx, personal.Ref), user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"first"}`))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"second"}`)); err != nil {
		t.Fatal(err)
	}
	if body := storedBody(t, db, p.Id); !strings.Contains(body, `"apikey":"hds:1:`+personal.Ref+`:`) {
		t.Errorf("stored body = %s, want the new key in %s", body, personal.Ref)
	}
}

func Test_Secret_ReferenceKeystore(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithReferenceStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	// The validator for real: it gets what the reference stands for.
	svc := provider.NewService(db, nil, nil, m)
	user := createTestUser(t, db, "references@example.com")

	refs, err := m.CreateKeystore(ctx, user.Id, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(map[string]string{"ovh/key": "s3cr3t"}))
	if err != nil {
		t.Fatal(err)
	}

	instantiatedWith = nil
	p, err := svc.CreateProvider(secret.WithKeystore(ctx, refs.Ref), user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"ovh/key"}`))
	if err != nil {
		t.Fatalf("CreateProvider = %v", err)
	}
	if instantiatedWith == nil || *instantiatedWith != "s3cr3t" {
		t.Errorf("validated with %v, want the secret the reference stands for", instantiatedWith)
	}
	if body := storedBody(t, db, p.Id); !strings.Contains(body, `"apikey":"hds:1:`+refs.Ref+`:`) {
		t.Fatalf("stored body = %s, want the reference in %s", body, refs.Ref)
	}

	// Its owner reads the reference back.
	shown, err := svc.GetUserProvider(ctx, user, p.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ShowReferences(ctx, shown); err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(shown.Provider); !strings.Contains(string(b), `"apikey":"ovh/key"`) {
		t.Errorf("shown provider = %s, want the reference", b)
	}

	// Sent back as shown, it stays a reference.
	instantiatedWith = nil
	if _, err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h2","apikey":"ovh/key"}`)); err != nil {
		t.Fatalf("UpdateProviderFromMessage(as shown) = %v", err)
	}
	if instantiatedWith == nil || *instantiatedWith != "s3cr3t" {
		t.Errorf("validated with %v, want the secret", instantiatedWith)
	}
	if body := storedBody(t, db, p.Id); !strings.Contains(body, `"apikey":"hds:1:`+refs.Ref+`:`) {
		t.Errorf("stored body = %s, want the reference still in %s", body, refs.Ref)
	}

	// Moved elsewhere as shown, it would become the secret: refused.
	_, err = svc.UpdateProviderFromMessage(secret.WithKeystore(ctx, secret.KindInstance), p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"ovh/key"}`))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) || !strings.Contains(err.Error(), secret.ErrReferenceAsSecret.Error()) {
		t.Errorf("UpdateProviderFromMessage(to instance, as shown) = %v, want a ValidationError for ErrReferenceAsSecret", err)
	}
	if body := storedBody(t, db, p.Id); !strings.Contains(body, `"apikey":"hds:1:`+refs.Ref+`:`) {
		t.Errorf("stored body = %s, want the reference left in %s", body, refs.Ref)
	}

	// A reference the keystore refuses is the owner's to fix: they are
	// told why.
	_, err = svc.CreateProvider(secret.WithKeystore(ctx, refs.Ref), user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"ovh/nope"}`))
	if !errors.As(err, &verr) || !strings.Contains(err.Error(), "unknown reference") {
		t.Errorf("CreateProvider(unknown reference) = %v, want a ValidationError with the reason of the keystore", err)
	}
}

// Sealed before the validation: a provider it refuses is still not stored,
// neither created nor updated.
func Test_Secret_RefusedAfterSealingStoresNothing(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithReferenceStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	accepting := provider.NewService(db, &mockValidator{}, nil, m)
	refusing := provider.NewService(db, failingValidator{}, nil, m)
	user := createTestUser(t, db, "refused@example.com")

	refs, err := m.CreateKeystore(ctx, user.Id, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(map[string]string{"ovh/key": "s3cr3t", "ovh/other": "0th3r"}))
	if err != nil {
		t.Fatal(err)
	}
	inRefs := secret.WithKeystore(ctx, refs.Ref)

	_, err = refusing.CreateProvider(inRefs, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"ovh/key"}`))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("CreateProvider(refused) = %v, want a ValidationError", err)
	}
	if providers, err := refusing.ListUserProviders(ctx, user); err != nil || len(providers) != 0 {
		t.Fatalf("ListUserProviders = %v, %v; want none stored", providers, err)
	}

	p, err := accepting.CreateProvider(inRefs, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"ovh/key"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := storedBody(t, db, p.Id)

	_, err = refusing.UpdateProviderFromMessage(inRefs, p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h2","apikey":"ovh/other"}`))
	if !errors.As(err, &verr) {
		t.Errorf("UpdateProviderFromMessage(refused) = %v, want a ValidationError", err)
	}
	if after := storedBody(t, db, p.Id); after != before {
		t.Errorf("stored body = %s, want it unchanged: %s", after, before)
	}
}

// A secret moved into a keystore of references would be taken for a
// reference: the owner is told to enter one, and nothing is stored.
func Test_Secret_SecretRefusedByReferenceKeystore(t *testing.T) {
	db, _ := inmemory.Instantiate()
	h, _ := secret.GenerateInstanceKeyset()
	key, _ := secret.NewInstanceKey(h)
	m, err := secret.NewManager(secrettest.WithReferenceStore(secret.Config{Policy: secret.PolicyInstance, InstanceKey: key, Safes: db, Defaults: db}))
	if err != nil {
		t.Fatal(err)
	}
	svc := provider.NewService(db, &mockValidator{}, nil, m)
	user := createTestUser(t, db, "references-only@example.com")

	refs, err := m.CreateKeystore(ctx, user.Id, secrettest.ReferenceKind, "My file", secrettest.ReferenceParams(map[string]string{"ovh/key": "s3cr3t"}))
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.CreateProvider(ctx, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"my-key"}`))
	if err != nil {
		t.Fatal(err)
	}
	before := storedBody(t, db, p.Id)

	_, err = svc.UpdateProviderFromMessage(secret.WithKeystore(ctx, refs.Ref), p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h","apikey":"`+happydns.RedactedSecret+`"}`))
	var verr happydns.ValidationError
	if !errors.As(err, &verr) || !strings.Contains(err.Error(), secret.ErrReferencesOnly.Error()) {
		t.Errorf("UpdateProviderFromMessage(secret into references) = %v, want a ValidationError for ErrReferencesOnly", err)
	}
	if after := storedBody(t, db, p.Id); after != before {
		t.Errorf("stored body = %s, want it unchanged: %s", after, before)
	}
}
