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

package provider_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"testing"

	"git.happydns.org/happyDomain/internal/secret"
	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

var ctx = context.Background()

func createTestUser(t *testing.T, store storage.Storage, email string) *happydns.User {
	user := &happydns.User{
		Id:    happydns.Identifier([]byte("user-" + email)),
		Email: email,
	}
	if err := store.CreateOrUpdateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestProviderMessage(t *testing.T, providerType string, comment string) *happydns.ProviderMessage {
	// Create a simple DDNS provider for testing
	ddnsProvider := &providers.DDNSServer{
		Server:  "127.0.0.1",
		KeyName: "testkey",
		KeyAlgo: "hmac-sha256",
		KeyBlob: []byte("testkey"),
	}

	providerJSON, err := json.Marshal(ddnsProvider)
	if err != nil {
		t.Fatalf("failed to marshal provider: %v", err)
	}

	return &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type:    providerType,
			Comment: comment,
		},
		Provider: providerJSON,
	}
}

// mockValidator is a validator that always succeeds
type mockValidator struct{}

func (v *mockValidator) Validate(p *happydns.Provider) error {
	return nil
}

func newTestService(t *testing.T) (*provider.Service, storage.Storage) {
	db, _ := inmemory.Instantiate()
	return provider.NewService(nil, nil, db, &mockValidator{}), db
}

func Test_CreateProvider(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test DDNS Provider")

	p, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(p.Id) == 0 {
		t.Error("expected provider ID to be set")
	}
	if !p.Owner.Equals(user.Id) {
		t.Errorf("expected provider owner to be %v, got %v", user.Id, p.Owner)
	}
	if p.Comment != "Test DDNS Provider" {
		t.Errorf("expected comment 'Test DDNS Provider', got %s", p.Comment)
	}

	// Verify provider is stored in database
	stored, err := db.GetProvider(p.Id)
	if err != nil {
		t.Fatalf("expected stored provider, got error: %v", err)
	}
	if stored.Comment != "Test DDNS Provider" {
		t.Errorf("expected stored comment to be 'Test DDNS Provider', got %s", stored.Comment)
	}
}

func Test_GetUserProvider(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Retrieve the provider
	retrievedProvider, err := providerService.GetUserProvider(ctx, user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !retrievedProvider.Id.Equals(createdProvider.Id) {
		t.Errorf("expected provider ID %s, got %s", createdProvider.Id, retrievedProvider.Id)
	}
	if retrievedProvider.Comment != "Test Provider" {
		t.Errorf("expected comment 'Test Provider', got %s", retrievedProvider.Comment)
	}
}

func Test_GetUserProvider_WrongUser(t *testing.T) {
	providerService, db := newTestService(t)

	user1 := createTestUser(t, db, "user1@example.com")
	user2 := createTestUser(t, db, "user2@example.com")

	// Create a provider for user1
	msg := createTestProviderMessage(t, "DDNSServer", "User1 Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user1, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to retrieve the provider as user2
	_, err = providerService.GetUserProvider(ctx, user2, createdProvider.Id)
	if err == nil {
		t.Error("expected error when retrieving another user's provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func Test_GetUserProvider_NotFound(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	nonexistentID := happydns.Identifier([]byte("nonexistent-id"))
	_, err := providerService.GetUserProvider(ctx, user, nonexistentID)
	if err == nil {
		t.Error("expected error when retrieving nonexistent provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func Test_GetUserProviderMeta(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider Meta")
	createdProvider, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Retrieve the provider metadata
	meta, err := providerService.GetUserProviderMeta(ctx, user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !meta.Id.Equals(createdProvider.Id) {
		t.Errorf("expected meta ID %s, got %s", createdProvider.Id, meta.Id)
	}
	if meta.Comment != "Test Provider Meta" {
		t.Errorf("expected comment 'Test Provider Meta', got %s", meta.Comment)
	}
}

func Test_ListUserProviders(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create multiple providers
	_, err := providerService.CreateProvider(ctx, user, createTestProviderMessage(t, "DDNSServer", "Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 1: %v", err)
	}
	_, err = providerService.CreateProvider(ctx, user, createTestProviderMessage(t, "DDNSServer", "Provider 2"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 2: %v", err)
	}
	_, err = providerService.CreateProvider(ctx, user, createTestProviderMessage(t, "DDNSServer", "Provider 3"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 3: %v", err)
	}

	// List providers
	providers, err := providerService.ListUserProviders(ctx, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}
}

func Test_ListUserProviders_MultipleUsers(t *testing.T) {
	providerService, db := newTestService(t)

	user1 := createTestUser(t, db, "user1@example.com")
	user2 := createTestUser(t, db, "user2@example.com")

	// Create providers for user1
	_, err := providerService.CreateProvider(ctx, user1, createTestProviderMessage(t, "DDNSServer", "User1 Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = providerService.CreateProvider(ctx, user1, createTestProviderMessage(t, "DDNSServer", "User1 Provider 2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create provider for user2
	_, err = providerService.CreateProvider(ctx, user2, createTestProviderMessage(t, "DDNSServer", "User2 Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List providers for user1
	user1Providers, err := providerService.ListUserProviders(ctx, user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Providers) != 2 {
		t.Errorf("expected 2 providers for user1, got %d", len(user1Providers))
	}

	// List providers for user2
	user2Providers, err := providerService.ListUserProviders(ctx, user2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Providers) != 1 {
		t.Errorf("expected 1 provider for user2, got %d", len(user2Providers))
	}
}

func Test_UpdateProvider(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Original comment")
	createdProvider, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Update the provider
	err = providerService.UpdateProvider(ctx, createdProvider.Id, user, func(p *happydns.Provider) {
		p.Comment = "Updated comment"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the provider was updated
	updated, err := providerService.GetUserProvider(ctx, user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error retrieving updated provider: %v", err)
	}
	if updated.Comment != "Updated comment" {
		t.Errorf("expected comment 'Updated comment', got %s", updated.Comment)
	}
}

func Test_UpdateProvider_PreventIdChange(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to change the provider ID
	newID := happydns.Identifier([]byte("new-provider-id"))
	err = providerService.UpdateProvider(ctx, createdProvider.Id, user, func(p *happydns.Provider) {
		p.Id = newID
	})
	if err == nil {
		t.Error("expected error when trying to change provider ID")
	}
	if _, ok := err.(happydns.ValidationError); !ok {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func Test_UpdateProvider_WrongUser(t *testing.T) {
	providerService, db := newTestService(t)

	user1 := createTestUser(t, db, "user1@example.com")
	user2 := createTestUser(t, db, "user2@example.com")

	// Create a provider for user1
	msg := createTestProviderMessage(t, "DDNSServer", "User1 Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user1, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to update the provider as user2
	err = providerService.UpdateProvider(ctx, createdProvider.Id, user2, func(p *happydns.Provider) {
		p.Comment = "Hijacked"
	})
	if err == nil {
		t.Error("expected error when updating another user's provider")
	}
}

func Test_DeleteProvider(t *testing.T) {
	providerService, db := newTestService(t)

	user := createTestUser(t, db, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Delete the provider
	err = providerService.DeleteProvider(ctx, user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the provider was deleted
	_, err = providerService.GetUserProvider(ctx, user, createdProvider.Id)
	if err == nil {
		t.Error("expected error when retrieving deleted provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func Test_DeleteProvider_WrongUser(t *testing.T) {
	providerService, db := newTestService(t)

	user1 := createTestUser(t, db, "user1@example.com")
	user2 := createTestUser(t, db, "user2@example.com")

	// Create a provider for user1
	msg := createTestProviderMessage(t, "DDNSServer", "User1 Provider")
	createdProvider, err := providerService.CreateProvider(ctx, user1, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to delete the provider as user2
	err = providerService.DeleteProvider(ctx, user2, createdProvider.Id)
	if err == nil {
		t.Error("expected error when deleting another user's provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}

	// Verify the provider still exists for user1
	_, err = providerService.GetUserProvider(ctx, user1, createdProvider.Id)
	if err != nil {
		t.Errorf("provider should still exist for user1, got error: %v", err)
	}
}

func Test_ParseProvider(t *testing.T) {
	msg := createTestProviderMessage(t, "DDNSServer", "Test Parse")

	p, err := provider.ParseProvider(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Comment != "Test Parse" {
		t.Errorf("expected comment 'Test Parse', got %s", p.Comment)
	}
	if p.Type != "DDNSServer" {
		t.Errorf("expected type 'DDNSServer', got %s", p.Type)
	}
	if p.Provider == nil {
		t.Error("expected provider to be instantiated")
	}
}

func Test_ParseProvider_InvalidType(t *testing.T) {
	msg := &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type: "NonExistentProvider",
		},
		Provider: json.RawMessage(`{}`),
	}

	_, err := provider.ParseProvider(msg)
	if err == nil {
		t.Error("expected error when parsing invalid provider type")
	}
}

func Test_RestrictedService_CreateProvider_Disabled(t *testing.T) {
	db, _ := inmemory.Instantiate()
	config := &happydns.Options{
		DisableProviders: true,
	}
	providerService := provider.NewRestrictedService(config, db, nil)

	user := createTestUser(t, db, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")

	_, err := providerService.CreateProvider(ctx, user, msg)
	if err == nil {
		t.Error("expected error when creating provider with DisableProviders=true")
	}
	if _, ok := err.(happydns.ForbiddenError); !ok {
		t.Errorf("expected ForbiddenError, got: %T", err)
	}
}

func Test_RestrictedService_UpdateProvider_Disabled(t *testing.T) {
	db, _ := inmemory.Instantiate()

	// First create a provider without restrictions
	unrestricted := provider.NewService(nil, nil, db, &mockValidator{})
	user := createTestUser(t, db, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := unrestricted.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Now try to update with restricted service
	config := &happydns.Options{
		DisableProviders: true,
	}
	restrictedService := provider.NewRestrictedService(config, db, nil)

	err = restrictedService.UpdateProvider(ctx, createdProvider.Id, user, func(p *happydns.Provider) {
		p.Comment = "Updated"
	})
	if err == nil {
		t.Error("expected error when updating provider with DisableProviders=true")
	}
	if _, ok := err.(happydns.ForbiddenError); !ok {
		t.Errorf("expected ForbiddenError, got: %T", err)
	}
}

func Test_RestrictedService_DeleteProvider_Disabled(t *testing.T) {
	db, _ := inmemory.Instantiate()

	// First create a provider without restrictions
	unrestricted := provider.NewService(nil, nil, db, &mockValidator{})
	user := createTestUser(t, db, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := unrestricted.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Now try to delete with restricted service
	config := &happydns.Options{
		DisableProviders: true,
	}
	restrictedService := provider.NewRestrictedService(config, db, nil)

	err = restrictedService.DeleteProvider(ctx, user, createdProvider.Id)
	if err == nil {
		t.Error("expected error when deleting provider with DisableProviders=true")
	}
	if _, ok := err.(happydns.ForbiddenError); !ok {
		t.Errorf("expected ForbiddenError, got: %T", err)
	}
}

// ---- Secret management integration tests ----

// testSecretKey32 is a 32-byte key (64 hex chars) used for instance-key tests.
var testSecretKey32 = hex.EncodeToString(bytes.Repeat([]byte{0xDE}, 32))

func newTestServiceWithSecrets(t *testing.T, mgr *secret.Manager, cfg *happydns.Options) (*provider.Service, storage.Storage) {
	t.Helper()
	db, _ := inmemory.Instantiate()
	return provider.NewService(cfg, mgr, db, &mockValidator{}), db
}

func newPlaintextManager(t *testing.T) *secret.Manager {
	t.Helper()
	mgr, err := secret.NewManagerFromConfig(&happydns.Options{SecretMethod: "plaintext"})
	if err != nil {
		t.Fatalf("failed to create plaintext manager: %v", err)
	}
	return mgr
}

func newInstanceKeyManager(t *testing.T) *secret.Manager {
	t.Helper()
	if err := flag.Set("secret-key", testSecretKey32); err != nil {
		t.Fatalf("failed to set secret-key flag: %v", err)
	}
	mgr, err := secret.NewManagerFromConfig(&happydns.Options{SecretMethod: "instance-key"})
	if err != nil {
		t.Fatalf("failed to create instance-key manager: %v", err)
	}
	return mgr
}

// Test_Secrets_Plaintext_RoundTrip verifies that provider credentials survive
// a create→get round-trip when the plaintext secret manager is active.
func Test_Secrets_Plaintext_RoundTrip(t *testing.T) {
	mgr := newPlaintextManager(t)
	svc, db := newTestServiceWithSecrets(t, mgr, &happydns.Options{})
	user := createTestUser(t, db, "plaintext@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Plaintext provider")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	got, err := svc.GetUserProvider(ctx, user, created.Id)
	if err != nil {
		t.Fatalf("GetUserProvider error: %v", err)
	}
	if got.Comment != "Plaintext provider" {
		t.Errorf("expected comment 'Plaintext provider', got %q", got.Comment)
	}
}

// Test_Secrets_Plaintext_StoredAsEnvelope verifies that even with the plaintext
// backend, credentials are wrapped in a SecretEnvelope in the database.
func Test_Secrets_Plaintext_StoredAsEnvelope(t *testing.T) {
	mgr := newPlaintextManager(t)
	cfg := &happydns.Options{}
	svc, db := newTestServiceWithSecrets(t, mgr, cfg)
	user := createTestUser(t, db, "envelope@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Envelope test")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	// Read the raw stored message directly from storage.
	stored, err := db.GetProvider(created.Id)
	if err != nil {
		t.Fatalf("GetProvider (raw) error: %v", err)
	}

	// The stored Provider field should be a SecretEnvelope, not raw credentials.
	var envelope happydns.SecretEnvelope
	if err := json.Unmarshal(stored.Provider, &envelope); err != nil {
		t.Fatalf("failed to unmarshal stored Provider as SecretEnvelope: %v", err)
	}
	if envelope.Version != 1 {
		t.Errorf("expected envelope Version=1, got %d", envelope.Version)
	}
	if envelope.Method != "plaintext" {
		t.Errorf("expected envelope Method=plaintext, got %q", envelope.Method)
	}
}

// Test_Secrets_InstanceKey_RoundTrip verifies encrypt→decrypt round-trip with
// the AES-256-GCM instance-key backend.
func Test_Secrets_InstanceKey_RoundTrip(t *testing.T) {
	mgr := newInstanceKeyManager(t)
	cfg := &happydns.Options{SecretMethod: "instance-key"}
	svc, db := newTestServiceWithSecrets(t, mgr, cfg)
	user := createTestUser(t, db, "instancekey@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "InstanceKey provider")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	got, err := svc.GetUserProvider(ctx, user, created.Id)
	if err != nil {
		t.Fatalf("GetUserProvider error: %v", err)
	}
	if got.Comment != "InstanceKey provider" {
		t.Errorf("expected comment 'InstanceKey provider', got %q", got.Comment)
	}
}

// Test_Secrets_InstanceKey_EncryptedAtRest verifies that credentials are not
// stored in cleartext when the instance-key backend is active.
func Test_Secrets_InstanceKey_EncryptedAtRest(t *testing.T) {
	mgr := newInstanceKeyManager(t)
	cfg := &happydns.Options{SecretMethod: "instance-key"}
	svc, db := newTestServiceWithSecrets(t, mgr, cfg)
	user := createTestUser(t, db, "encrypted@example.com")

	ddns := &providers.DDNSServer{
		Server:  "192.168.1.1",
		KeyName: "mysecretkey",
		KeyAlgo: "hmac-sha256",
		KeyBlob: []byte("supersecretvalue"),
	}
	providerJSON, _ := json.Marshal(ddns)
	msg := &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{Type: "DDNSServer"},
		Provider:     providerJSON,
	}

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	stored, err := db.GetProvider(created.Id)
	if err != nil {
		t.Fatalf("GetProvider (raw) error: %v", err)
	}

	// The raw stored bytes should NOT contain the secret value in plaintext.
	if bytes.Contains(stored.Provider, []byte("supersecretvalue")) {
		t.Error("secret value found in plaintext in the stored provider data")
	}
	if bytes.Contains(stored.Provider, []byte("mysecretkey")) {
		t.Error("key name found in plaintext in the stored provider data")
	}
}

// Test_Secrets_LegacyPlaintext_BackwardCompat verifies that providers stored
// before the secret management system (raw JSON, no envelope) are still readable.
func Test_Secrets_LegacyPlaintext_BackwardCompat(t *testing.T) {
	mgr := newPlaintextManager(t)
	cfg := &happydns.Options{}
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(cfg, mgr, db, &mockValidator{})
	user := createTestUser(t, db, "legacy@example.com")

	// Insert a provider directly with raw JSON (simulating pre-envelope data).
	ddns := &providers.DDNSServer{
		Server:  "10.0.0.1",
		KeyName: "legacykey",
		KeyAlgo: "hmac-sha256",
		KeyBlob: []byte("legacyvalue"),
	}
	rawCredentials, _ := json.Marshal(ddns)
	legacy := &happydns.ProviderMessage{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "DDNSServer",
			Comment: "Legacy provider",
			Owner:   user.Id,
		},
		Provider: rawCredentials,
	}
	if err := db.CreateProviderFromMessage(legacy); err != nil {
		t.Fatalf("failed to insert legacy provider: %v", err)
	}

	// GetUserProvider should transparently handle the raw (non-envelope) JSON.
	got, err := svc.GetUserProvider(ctx, user, legacy.Id)
	if err != nil {
		t.Fatalf("GetUserProvider for legacy provider error: %v", err)
	}
	if got.Comment != "Legacy provider" {
		t.Errorf("expected comment 'Legacy provider', got %q", got.Comment)
	}
}

// Test_Secrets_Update_ReEncrypts verifies that updating a provider re-seals
// credentials with the active secret method.
func Test_Secrets_Update_ReEncrypts(t *testing.T) {
	mgr := newPlaintextManager(t)
	cfg := &happydns.Options{}
	svc, db := newTestServiceWithSecrets(t, mgr, cfg)
	user := createTestUser(t, db, "update@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Original comment")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	err = svc.UpdateProvider(ctx, created.Id, user, func(p *happydns.Provider) {
		p.Comment = "Updated comment"
	})
	if err != nil {
		t.Fatalf("UpdateProvider error: %v", err)
	}

	got, err := svc.GetUserProvider(ctx, user, created.Id)
	if err != nil {
		t.Fatalf("GetUserProvider error: %v", err)
	}
	if got.Comment != "Updated comment" {
		t.Errorf("expected 'Updated comment', got %q", got.Comment)
	}

	// Updated provider should still be stored as an envelope.
	stored, err := db.GetProvider(created.Id)
	if err != nil {
		t.Fatalf("db.GetProvider error: %v", err)
	}
	var envelope happydns.SecretEnvelope
	if err := json.Unmarshal(stored.Provider, &envelope); err != nil {
		t.Fatalf("stored Provider after update is not an envelope: %v", err)
	}
	if envelope.Version != 1 {
		t.Errorf("expected envelope Version=1 after update, got %d", envelope.Version)
	}
}

// Test_Secrets_PerUserMethod verifies that a user's SecretMethod setting
// overrides the instance default.
func Test_Secrets_PerUserMethod(t *testing.T) {
	if err := flag.Set("secret-key", testSecretKey32); err != nil {
		t.Fatalf("failed to set secret-key flag: %v", err)
	}
	ikMgr, err := secret.NewManagerFromConfig(&happydns.Options{SecretMethod: "instance-key"})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create a service where instance default is instance-key,
	// but one user prefers plaintext.
	cfg := &happydns.Options{SecretMethod: "instance-key"}
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(cfg, ikMgr, db, &mockValidator{})

	user := createTestUser(t, db, "peruser@example.com")
	user.Settings.SecretMethod = "plaintext"
	msg := createTestProviderMessage(t, "DDNSServer", "Per-user plaintext")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	stored, err := db.GetProvider(created.Id)
	if err != nil {
		t.Fatalf("db.GetProvider error: %v", err)
	}
	var envelope happydns.SecretEnvelope
	if err := json.Unmarshal(stored.Provider, &envelope); err != nil {
		t.Fatalf("stored Provider is not an envelope: %v", err)
	}
	// Should be stored as plaintext, not instance-key.
	if envelope.Method != "plaintext" {
		t.Errorf("expected user's preferred method 'plaintext', got %q", envelope.Method)
	}
}

// Test_Secrets_DisableUserSecretMethod verifies that the DisableUserSecretMethod
// config flag forces the instance-level method even when the user has a preference.
func Test_Secrets_DisableUserSecretMethod(t *testing.T) {
	if err := flag.Set("secret-key", testSecretKey32); err != nil {
		t.Fatalf("failed to set secret-key flag: %v", err)
	}
	ikMgr, err := secret.NewManagerFromConfig(&happydns.Options{SecretMethod: "instance-key"})
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// DisableUserSecretMethod=true means per-user overrides are ignored.
	cfg := &happydns.Options{
		SecretMethod:            "instance-key",
		DisableUserSecretMethod: true,
	}
	db, _ := inmemory.Instantiate()
	svc := provider.NewService(cfg, ikMgr, db, &mockValidator{})

	user := createTestUser(t, db, "disabled@example.com")
	user.Settings.SecretMethod = "plaintext" // user wants plaintext, but it's disabled
	msg := createTestProviderMessage(t, "DDNSServer", "DisableUserMethod test")

	created, err := svc.CreateProvider(ctx, user, msg)
	if err != nil {
		t.Fatalf("CreateProvider error: %v", err)
	}

	stored, err := db.GetProvider(created.Id)
	if err != nil {
		t.Fatalf("db.GetProvider error: %v", err)
	}
	var envelope happydns.SecretEnvelope
	if err := json.Unmarshal(stored.Provider, &envelope); err != nil {
		t.Fatalf("stored Provider is not an envelope: %v", err)
	}
	// Should be stored with instance-key despite the user's plaintext preference.
	if envelope.Method != "instance-key" {
		t.Errorf("expected instance-level method 'instance-key', got %q", envelope.Method)
	}
}
