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

package domain_test

import (
	"fmt"
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/domain"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/providers"
)

// Mock implementations for testing

func init() {
	// Register the mock provider
	providers.RegisterProvider(func() happydns.ProviderBody {
		return &mockProviderBody{}
	}, happydns.ProviderInfos{
		Name:        "Mock Provider",
		Description: "A mock provider for testing",
	})
}

type mockProviderBody struct {
	Name string `json:"name"`
}

func (m *mockProviderBody) InstantiateProvider() (happydns.ProviderActuator, error) {
	return &mockProviderActuator{}, nil
}

type mockProviderActuator struct{}

func (m *mockProviderActuator) CanCreateDomain() bool {
	return true
}

func (m *mockProviderActuator) CanListZones() bool {
	return true
}

func (m *mockProviderActuator) CreateDomain(fqdn string) error {
	return nil
}

func (m *mockProviderActuator) GetZoneRecords(domain string) ([]happydns.Record, error) {
	return []happydns.Record{}, nil
}

func (m *mockProviderActuator) GetZoneCorrections(domain string, wantedRecords []happydns.Record) ([]*happydns.Correction, error) {
	return []*happydns.Correction{}, nil
}

func (m *mockProviderActuator) ListZones() ([]string, error) {
	return []string{}, nil
}

type mockDomainLogAppender struct {
	logs []*happydns.DomainLog
}

func (m *mockDomainLogAppender) AppendDomainLog(d *happydns.Domain, log *happydns.DomainLog) error {
	m.logs = append(m.logs, log)
	return nil
}

// Helper functions

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

func createTestProvider(t *testing.T, store *inmemory.InMemoryStorage, user *happydns.User, name string) happydns.Identifier {
	provider := &happydns.Provider{
		ProviderMeta: happydns.ProviderMeta{
			Type:    "mockProviderBody",
			Owner:   user.Id,
			Comment: name,
		},
		Provider: &mockProviderBody{
			Name: name,
		},
	}
	if err := store.CreateProvider(provider); err != nil {
		t.Fatalf("failed to create test provider: %v", err)
	}
	return provider.Id
}

func setupTestService(store *inmemory.InMemoryStorage) (*domain.Service, *mockDomainLogAppender) {
	// Create the provider service
	providerService := providerUC.NewService(store)

	// Create the zone usecase
	getZone := zoneUC.NewGetZoneUsecase(store)

	// Create the mock domain log appender
	logAppender := &mockDomainLogAppender{
		logs: make([]*happydns.DomainLog, 0),
	}

	// Create the domain service
	service := domain.NewService(
		store,
		providerService,
		getZone,
		providerService,
		logAppender,
	)

	return service, logAppender
}

// Tests

func Test_CreateDomain(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, logAppender := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}

	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify domain was created
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}

	if len(domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(domains))
	}

	if domains[0].DomainName != "example.com." {
		t.Errorf("expected domain name 'example.com.', got %s", domains[0].DomainName)
	}

	if !domains[0].Owner.Equals(user.Id) {
		t.Errorf("expected owner to be %v, got %v", user.Id, domains[0].Owner)
	}

	if !domains[0].ProviderId.Equals(providerId) {
		t.Errorf("expected provider ID to be %v, got %v", providerId, domains[0].ProviderId)
	}

	// Verify log entry was created
	if len(logAppender.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logAppender.logs))
	}
}

func Test_CreateDomain_InvalidProvider(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	invalidProviderId := happydns.Identifier([]byte("invalid-provider"))

	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: invalidProviderId,
	}

	err := service.CreateDomain(user, domainToCreate)
	if err == nil {
		t.Error("expected error when creating domain with invalid provider")
	}
}

func Test_GetUserDomain(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Retrieve the domain
	retrieved, err := service.GetUserDomain(user, createdDomain.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !retrieved.Id.Equals(createdDomain.Id) {
		t.Errorf("expected domain ID %v, got %v", createdDomain.Id, retrieved.Id)
	}

	if retrieved.DomainName != "example.com." {
		t.Errorf("expected domain name 'example.com.', got %s", retrieved.DomainName)
	}
}

func Test_GetUserDomain_WrongUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")
	providerId := createTestProvider(t, mem, user1, "Test Provider")

	// Create a domain for user1
	domainToCreate := &happydns.Domain{
		DomainName: "user1-domain.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user1, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user1)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Try to retrieve the domain as user2
	_, err = service.GetUserDomain(user2, createdDomain.Id)
	if err == nil {
		t.Error("expected error when retrieving another user's domain")
	}
	if err != happydns.ErrDomainNotFound {
		t.Errorf("expected ErrDomainNotFound, got %v", err)
	}
}

func Test_GetUserDomain_NotFound(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	nonexistentId := happydns.Identifier([]byte("nonexistent-domain"))

	_, err := service.GetUserDomain(user, nonexistentId)
	if err == nil {
		t.Error("expected error when retrieving nonexistent domain")
	}
	if err != happydns.ErrDomainNotFound {
		t.Errorf("expected ErrDomainNotFound, got %v", err)
	}
}

func Test_GetUserDomainByFQDN(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com.",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Retrieve by FQDN
	domains, err := service.GetUserDomainByFQDN(user, "example.com.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(domains))
	}

	if domains[0].DomainName != "example.com." {
		t.Errorf("expected domain name 'example.com.', got %s", domains[0].DomainName)
	}
}

func Test_ListUserDomains(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create multiple domains
	domainNames := []string{"example1.com", "example2.com", "example3.com"}
	for _, name := range domainNames {
		domainToCreate := &happydns.Domain{
			DomainName: name,
			ProviderId: providerId,
		}
		err := service.CreateDomain(user, domainToCreate)
		if err != nil {
			t.Fatalf("failed to create domain %s: %v", name, err)
		}
	}

	// List domains
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(domains) != 3 {
		t.Errorf("expected 3 domains, got %d", len(domains))
	}
}

func Test_ListUserDomains_MultipleUsers(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")
	providerId1 := createTestProvider(t, mem, user1, "Provider 1")
	providerId2 := createTestProvider(t, mem, user2, "Provider 2")

	// Create domains for user1
	for i := 1; i <= 2; i++ {
		domainToCreate := &happydns.Domain{
			DomainName: fmt.Sprintf("user1-domain%d.com", i),
			ProviderId: providerId1,
		}
		err := service.CreateDomain(user1, domainToCreate)
		if err != nil {
			t.Fatalf("failed to create domain: %v", err)
		}
	}

	// Create domain for user2
	domainToCreate := &happydns.Domain{
		DomainName: "user2-domain.com",
		ProviderId: providerId2,
	}
	err := service.CreateDomain(user2, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// List domains for user1
	user1Domains, err := service.ListUserDomains(user1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user1Domains) != 2 {
		t.Errorf("expected 2 domains for user1, got %d", len(user1Domains))
	}

	// List domains for user2
	user2Domains, err := service.ListUserDomains(user2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(user2Domains) != 1 {
		t.Errorf("expected 1 domain for user2, got %d", len(user2Domains))
	}
}

func Test_ListUserDomains_Empty(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")

	// List domains (should be empty)
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(domains) != 0 {
		t.Errorf("expected 0 domains, got %d", len(domains))
	}
}

func Test_UpdateDomain(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, logAppender := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Clear logs from creation
	logAppender.logs = make([]*happydns.DomainLog, 0)

	// Update the domain
	err = service.Update(createdDomain.Id, user, func(d *happydns.Domain) {
		d.Group = "test-group"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the domain was updated
	updated, err := service.GetUserDomain(user, createdDomain.Id)
	if err != nil {
		t.Fatalf("failed to retrieve updated domain: %v", err)
	}

	if updated.Group != "test-group" {
		t.Errorf("expected group 'test-group', got %s", updated.Group)
	}

	// Verify log entry was created
	if len(logAppender.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logAppender.logs))
	}
}

func Test_UpdateDomain_PreventIdChange(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Try to change the domain ID
	err = service.Update(createdDomain.Id, user, func(d *happydns.Domain) {
		d.Id = happydns.Identifier([]byte("new-id"))
	})
	if err == nil {
		t.Error("expected error when trying to change domain ID")
	}

	// Check the error message
	validationErr, ok := err.(happydns.ValidationError)
	if !ok {
		t.Errorf("expected ValidationError, got %T", err)
	} else if validationErr.Msg != "you cannot change the domain identifier" {
		t.Errorf("expected specific error message, got: %s", validationErr.Msg)
	}
}

func Test_UpdateDomain_WrongUser(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user1 := createTestUser(t, mem, "user1@example.com")
	user2 := createTestUser(t, mem, "user2@example.com")
	providerId := createTestProvider(t, mem, user1, "Test Provider")

	// Create a domain for user1
	domainToCreate := &happydns.Domain{
		DomainName: "user1-domain.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user1, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user1)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Try to update the domain as user2
	err = service.Update(createdDomain.Id, user2, func(d *happydns.Domain) {
		d.Group = "hijacked"
	})
	if err == nil {
		t.Error("expected error when updating another user's domain")
	}
}

func Test_DeleteDomain(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Delete the domain
	err = service.DeleteDomain(createdDomain.Id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the domain was deleted
	_, err = service.GetUserDomain(user, createdDomain.Id)
	if err == nil {
		t.Error("expected error when retrieving deleted domain")
	}
	if err != happydns.ErrDomainNotFound {
		t.Errorf("expected ErrDomainNotFound, got %v", err)
	}
}

func Test_UpdateDomain_Alias(t *testing.T) {
	mem, _ := inmemory.NewInMemoryStorage()
	service, _ := setupTestService(mem)

	user := createTestUser(t, mem, "test@example.com")
	providerId := createTestProvider(t, mem, user, "Test Provider")

	// Create a domain
	domainToCreate := &happydns.Domain{
		DomainName: "example.com",
		ProviderId: providerId,
	}
	err := service.CreateDomain(user, domainToCreate)
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Get the created domain
	domains, err := service.ListUserDomains(user)
	if err != nil {
		t.Fatalf("failed to list domains: %v", err)
	}
	createdDomain := domains[0]

	// Test the UpdateDomain alias method
	err = service.UpdateDomain(createdDomain.Id, user, func(d *happydns.Domain) {
		d.Group = "alias-test"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the domain was updated
	updated, err := service.GetUserDomain(user, createdDomain.Id)
	if err != nil {
		t.Fatalf("failed to retrieve updated domain: %v", err)
	}

	if updated.Group != "alias-test" {
		t.Errorf("expected group 'alias-test', got %s", updated.Group)
	}
}
