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

package domain_test

import (
	"testing"

	"git.happydns.org/happyDomain/internal/storage/inmemory"
	providerUC "git.happydns.org/happyDomain/internal/usecase/provider"
	"git.happydns.org/happyDomain/model"
)

func Test_ShareDomain(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)
	providerService := providerUC.NewService(db, nil, nil)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")
	stranger := createTestUser(t, db, "stranger@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	dom, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Before sharing: the guest cannot see the domain.
	if _, err := service.GetUserDomain(guest, dom.Id); err == nil {
		t.Fatal("expected guest to be denied before sharing")
	}

	// Share with the guest, including provider access, resolving by email.
	grantee, err := service.ShareDomain(owner, dom.Id, "guest@example.com", true)
	if err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}
	if !grantee.Id.Equals(guest.Id) {
		t.Fatalf("expected grantee to be the guest, got %v", grantee.Id)
	}

	// The guest can now access the domain and it appears in their list.
	if _, err := service.GetUserDomain(guest, dom.Id); err != nil {
		t.Fatalf("expected guest to access shared domain: %v", err)
	}
	domains, err := service.ListUserDomains(guest)
	if err != nil {
		t.Fatalf("ListUserDomains failed: %v", err)
	}
	if len(domains) != 1 || !domains[0].Id.Equals(dom.Id) {
		t.Fatalf("expected shared domain in guest's list, got %d domains", len(domains))
	}

	// A stranger remains denied.
	if _, err := service.GetUserDomain(stranger, dom.Id); err == nil {
		t.Fatal("expected stranger to remain denied")
	}

	// Provider access follows the grant.
	if !service.CanManageProvider(guest, dom) {
		t.Fatal("expected guest to be allowed to manage the provider")
	}
	if service.CanManageProvider(stranger, dom) {
		t.Fatal("expected stranger to be denied provider management")
	}
	if _, err := providerService.GetProviderForZone(ctx, guest, providerId); err != nil {
		t.Fatalf("expected guest to resolve provider for zone ops: %v", err)
	}
	if _, err := providerService.GetProviderForZone(ctx, stranger, providerId); err == nil {
		t.Fatal("expected stranger to be denied provider for zone ops")
	}
	// The owner-only management API stays owner-only for the guest.
	if _, err := providerService.GetUserProvider(ctx, guest, providerId); err == nil {
		t.Fatal("expected guest to be denied provider management API")
	}

	// Revoke and verify everything is cleaned up.
	if err := service.UnshareDomain(owner, dom.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain failed: %v", err)
	}
	if _, err := service.GetUserDomain(guest, dom.Id); err == nil {
		t.Fatal("expected guest to be denied after revoke")
	}
	if service.CanManageProvider(guest, dom) {
		t.Fatal("expected guest provider access to be revoked")
	}
}

func Test_ShareDomain_NotOwner(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")
	_ = createTestUser(t, db, "target@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	dom, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// A non-owner cannot share the domain.
	if _, err := service.ShareDomain(guest, dom.Id, "target@example.com", false); err == nil {
		t.Fatal("expected non-owner to be denied sharing")
	}

	// Unknown grantee reference is rejected.
	if _, err := service.ShareDomain(owner, dom.Id, "nobody@example.com", false); err == nil {
		t.Fatal("expected sharing with unknown user to fail")
	}
}

func Test_UnshareDomain_SelfLeave(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	dom, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	if _, err := service.ShareDomain(owner, dom.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}

	// The guest can remove their own access.
	if err := service.UnshareDomain(guest, dom.Id, guest.Id); err != nil {
		t.Fatalf("expected guest self-leave to succeed: %v", err)
	}
	if _, err := service.GetUserDomain(guest, dom.Id); err == nil {
		t.Fatal("expected guest to be denied after self-leave")
	}
}

// Test_UnshareDomain_KeepsProviderForOtherDomain ensures that revoking one
// shared domain does not strip provider access from another domain the same
// grantee still has, when both are backed by the same provider.
func Test_UnshareDomain_KeepsProviderForOtherDomain(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")

	// Two domains backed by the same provider.
	providerId := createTestProvider(t, db, owner, "Owner Provider")
	domA, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "a.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain A: %v", err)
	}
	domB, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "b.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain B: %v", err)
	}

	// Share both with the guest, including provider access.
	if _, err := service.ShareDomain(owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if _, err := service.ShareDomain(owner, domB.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain B failed: %v", err)
	}

	// Revoking A must not remove provider access still needed by B.
	if err := service.UnshareDomain(owner, domA.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain A failed: %v", err)
	}
	if !service.CanManageProvider(guest, domB) {
		t.Fatal("expected guest to keep provider access for domain B after unsharing A")
	}

	// Revoking the last domain finally drops the provider grant.
	if err := service.UnshareDomain(owner, domB.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain B failed: %v", err)
	}
	if service.CanManageProvider(guest, domB) {
		t.Fatal("expected guest provider access to be revoked once no shared domain needs it")
	}
}

// Test_GetUserDomainByFQDN_Shared ensures a domain shared with a user can be
// resolved by FQDN, just like an owned domain, and that a stranger cannot.
func Test_GetUserDomainByFQDN_Shared(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")
	stranger := createTestUser(t, db, "stranger@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	dom, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Before sharing: the guest cannot resolve the domain by FQDN.
	if _, err := service.GetUserDomainByFQDN(guest, "example.com."); err == nil {
		t.Fatal("expected guest FQDN lookup to be denied before sharing")
	}

	if _, err := service.ShareDomain(owner, dom.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}

	// After sharing: the guest resolves it by FQDN (case-insensitively).
	domains, err := service.GetUserDomainByFQDN(guest, "Example.COM")
	if err != nil {
		t.Fatalf("expected guest to resolve shared domain by FQDN: %v", err)
	}
	if len(domains) != 1 || !domains[0].Id.Equals(dom.Id) {
		t.Fatalf("expected the shared domain, got %d domains", len(domains))
	}

	// A stranger still cannot resolve it.
	if _, err := service.GetUserDomainByFQDN(stranger, "example.com."); err == nil {
		t.Fatal("expected stranger FQDN lookup to remain denied")
	}
}

// Test_DeleteDomain_CleansProviderShare ensures deleting a shared managed
// domain revokes the grantee's provider grant, but only when no other shared
// domain on the same provider still needs it.
func Test_DeleteDomain_CleansProviderShare(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	domA, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "a.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain A: %v", err)
	}
	domB, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "b.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain B: %v", err)
	}

	if _, err := service.ShareDomain(owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if _, err := service.ShareDomain(owner, domB.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain B failed: %v", err)
	}

	// Deleting A must not strip provider access still needed by B.
	if err := service.DeleteDomain(domA.Id); err != nil {
		t.Fatalf("DeleteDomain A failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); !shared {
		t.Fatal("expected provider share to survive while domain B still needs it")
	}

	// Deleting the last shared domain drops the now-orphaned provider grant.
	if err := service.DeleteDomain(domB.Id); err != nil {
		t.Fatalf("DeleteDomain B failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); shared {
		t.Fatal("expected provider share to be cleaned up once no shared domain needs it")
	}
}
