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
	"testing"

	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
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
	m, err := secret.NewManager(secret.PolicyPlaintext)
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

	if err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", `{"host":"h2","apikey":"new"}`)); err != nil {
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
	if err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", body)); err != nil {
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
	if err := svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "OtherSecretTestProvider", body)); err != nil {
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

	err = svc.UpdateProviderFromMessage(ctx, p.Id, user, secretMessage(t, "SecretTestProvider", sealed))
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
