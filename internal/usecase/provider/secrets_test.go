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
