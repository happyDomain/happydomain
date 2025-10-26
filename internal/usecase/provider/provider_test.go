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
	"encoding/json"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

func createTestUser(t *testing.T, store *inmemory.InMemoryStorage, email string) *happydns.User {
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

func Test_CreateProvider(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	// Replace validator with mock to avoid actual DNS validation
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test DDNS Provider")

	p, err := providerService.CreateProvider(user, msg)
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
	stored, err := mem.GetProvider(p.Id)
	if err != nil {
		t.Fatalf("expected stored provider, got error: %v", err)
	}
	if stored.Comment != "Test DDNS Provider" {
		t.Errorf("expected stored comment to be 'Test DDNS Provider', got %s", stored.Comment)
	}
}

func Test_GetUserProvider(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Retrieve the provider
	retrievedProvider, err := providerService.GetUserProvider(user, createdProvider.Id)
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
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create a provider for user1
	msg := createTestProviderMessage(t, "DDNSServer", "User1 Provider")
	createdProvider, err := providerService.CreateProvider(user1, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to retrieve the provider as user2
	_, err = providerService.GetUserProvider(user2, createdProvider.Id)
	if err == nil {
		t.Error("expected error when retrieving another user's provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func Test_GetUserProvider_NotFound(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)

	user := createTestUser(t, mem, "test@example.com")

	nonexistentID := happydns.Identifier([]byte("nonexistent-id"))
	_, err := providerService.GetUserProvider(user, nonexistentID)
	if err == nil {
		t.Error("expected error when retrieving nonexistent provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func Test_GetUserProviderMeta(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider Meta")
	createdProvider, err := providerService.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Retrieve the provider metadata
	meta, err := providerService.GetUserProviderMeta(user, createdProvider.Id)
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
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create multiple providers
	_, err := providerService.CreateProvider(user, createTestProviderMessage(t, "DDNSServer", "Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 1: %v", err)
	}
	_, err = providerService.CreateProvider(user, createTestProviderMessage(t, "DDNSServer", "Provider 2"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 2: %v", err)
	}
	_, err = providerService.CreateProvider(user, createTestProviderMessage(t, "DDNSServer", "Provider 3"))
	if err != nil {
		t.Fatalf("unexpected error creating provider 3: %v", err)
	}

	// List providers
	providers, err := providerService.ListUserProviders(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(providers))
	}
}

func Test_ListUserProviders_MultipleUsers(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create providers for user1
	_, err := providerService.CreateProvider(user1, createTestProviderMessage(t, "DDNSServer", "User1 Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = providerService.CreateProvider(user1, createTestProviderMessage(t, "DDNSServer", "User1 Provider 2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create provider for user2
	_, err = providerService.CreateProvider(user2, createTestProviderMessage(t, "DDNSServer", "User2 Provider 1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// List providers for user1
	user1Providers, err := providerService.ListUserProviders(user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Providers) != 2 {
		t.Errorf("expected 2 providers for user1, got %d", len(user1Providers))
	}

	// List providers for user2
	user2Providers, err := providerService.ListUserProviders(user2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Providers) != 1 {
		t.Errorf("expected 1 provider for user2, got %d", len(user2Providers))
	}
}

func Test_UpdateProvider(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Original comment")
	createdProvider, err := providerService.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Update the provider
	err = providerService.UpdateProvider(createdProvider.Id, user, func(p *happydns.Provider) {
		p.Comment = "Updated comment"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the provider was updated
	updated, err := providerService.GetUserProvider(user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error retrieving updated provider: %v", err)
	}
	if updated.Comment != "Updated comment" {
		t.Errorf("expected comment 'Updated comment', got %s", updated.Comment)
	}
}

func Test_UpdateProvider_PreventIdChange(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to change the provider ID
	newID := happydns.Identifier([]byte("new-provider-id"))
	err = providerService.UpdateProvider(createdProvider.Id, user, func(p *happydns.Provider) {
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
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")

	// Create a provider for user1
	msg := createTestProviderMessage(t, "DDNSServer", "User1 Provider")
	createdProvider, err := providerService.CreateProvider(user1, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Try to update the provider as user2
	err = providerService.UpdateProvider(createdProvider.Id, user2, func(p *happydns.Provider) {
		p.Comment = "Hijacked"
	})
	if err == nil {
		t.Error("expected error when updating another user's provider")
	}
}

func Test_DeleteProvider(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	providerService := provider.NewService(mem)
	providerService.SetValidator(&mockValidator{})

	user := createTestUser(t, mem, "test@example.com")

	// Create a provider
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := providerService.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Delete the provider
	err = providerService.DeleteProvider(user, createdProvider.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the provider was deleted
	_, err = providerService.GetUserProvider(user, createdProvider.Id)
	if err == nil {
		t.Error("expected error when retrieving deleted provider")
	}
	if err != happydns.ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
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
	mem, _ := inmemory.NewInMemoryStorage()
	config := &happydns.Options{
		DisableProviders: true,
	}
	providerService := provider.NewRestrictedService(config, mem)

	user := createTestUser(t, mem, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")

	_, err := providerService.CreateProvider(user, msg)
	if err == nil {
		t.Error("expected error when creating provider with DisableProviders=true")
	}
	if _, ok := err.(happydns.ForbiddenError); !ok {
		t.Errorf("expected ForbiddenError, got: %T", err)
	}
}

func Test_RestrictedService_UpdateProvider_Disabled(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()

	// First create a provider without restrictions
	unrestricted := provider.NewService(mem)
	unrestricted.SetValidator(&mockValidator{})
	user := createTestUser(t, mem, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := unrestricted.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Now try to update with restricted service
	config := &happydns.Options{
		DisableProviders: true,
	}
	restrictedService := provider.NewRestrictedService(config, mem)

	err = restrictedService.UpdateProvider(createdProvider.Id, user, func(p *happydns.Provider) {
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
	mem, _ := inmemory.NewInMemoryStorage()

	// First create a provider without restrictions
	unrestricted := provider.NewService(mem)
	unrestricted.SetValidator(&mockValidator{})
	user := createTestUser(t, mem, "test@example.com")
	msg := createTestProviderMessage(t, "DDNSServer", "Test Provider")
	createdProvider, err := unrestricted.CreateProvider(user, msg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	// Now try to delete with restricted service
	config := &happydns.Options{
		DisableProviders: true,
	}
	restrictedService := provider.NewRestrictedService(config, mem)

	err = restrictedService.DeleteProvider(user, createdProvider.Id)
	if err == nil {
		t.Error("expected error when deleting provider with DisableProviders=true")
	}
	if _, ok := err.(happydns.ForbiddenError); !ok {
		t.Errorf("expected ForbiddenError, got: %T", err)
	}
}
