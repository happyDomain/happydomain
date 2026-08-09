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
	"strings"
	"testing"

	"git.happydns.org/happyDomain/internal/storage"
	"git.happydns.org/happyDomain/internal/storage/inmemory"
	"git.happydns.org/happyDomain/internal/usecase/domain"
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
	grantee, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", true)
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
	if _, err := providerService.GetProviderForZone(ctx, guest, dom); err != nil {
		t.Fatalf("expected guest to resolve provider for zone ops: %v", err)
	}
	if _, err := providerService.GetProviderForZone(ctx, stranger, dom); err == nil {
		t.Fatal("expected stranger to be denied provider for zone ops")
	}
	// The owner-only management API stays owner-only for the guest.
	if _, err := providerService.GetUserProvider(ctx, guest, providerId); err == nil {
		t.Fatal("expected guest to be denied provider management API")
	}

	// Revoke and verify everything is cleaned up.
	if err := service.UnshareDomain(ctx, owner, dom.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain failed: %v", err)
	}
	if _, err := service.GetUserDomain(guest, dom.Id); err == nil {
		t.Fatal("expected guest to be denied after revoke")
	}
	if service.CanManageProvider(guest, dom) {
		t.Fatal("expected guest provider access to be revoked")
	}
}

func Test_UpdateDomain_DeniedForGrantee(t *testing.T) {
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

	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}

	// A grantee must not be able to alter the domain's own properties, even
	// though they can read it via GetUserDomain.
	err = service.Update(dom.Id, guest, func(d *happydns.Domain) {
		d.Group = "hijacked"
	})
	if err == nil {
		t.Fatal("expected guest to be denied when updating a shared domain")
	}

	updated, err := service.GetUserDomain(guest, dom.Id)
	if err != nil {
		t.Fatalf("GetUserDomain failed: %v", err)
	}
	if updated.Group == "hijacked" {
		t.Fatal("guest's update should not have been applied")
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
	if _, err := service.ShareDomain(ctx, guest, dom.Id, "target@example.com", false); err == nil {
		t.Fatal("expected non-owner to be denied sharing")
	}

	// Unknown grantee reference is rejected.
	if _, err := service.ShareDomain(ctx, owner, dom.Id, "nobody@example.com", false); err == nil {
		t.Fatal("expected sharing with unknown user to fail")
	}
}

// Test_ShareDomain_ProviderNotOwned covers a domain whose owner does not
// actually own its provider (e.g. created out-of-band through the admin
// API). Domain ownership alone must not be enough to extend a share to the
// provider.
func Test_ShareDomain_ProviderNotOwned(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	other := createTestUser(t, db, "other@example.com")
	guest := createTestUser(t, db, "guest@example.com")

	// The provider belongs to a different user than the domain owner.
	providerId := createTestProvider(t, db, other, "Other's Provider")

	dom := &happydns.Domain{
		Owner:      owner.Id,
		ProviderId: providerId,
		DomainName: "mismatched.example.com",
	}
	if err := service.AdminCreateDomain(dom); err != nil {
		t.Fatalf("failed to admin-create domain: %v", err)
	}

	// Sharing the domain itself is still allowed.
	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", false); err != nil {
		t.Fatalf("expected domain-only share to succeed: %v", err)
	}
	if err := service.UnshareDomain(ctx, owner, dom.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain failed: %v", err)
	}

	// But extending the grant to the provider must be denied, since the
	// domain owner does not own the provider.
	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", true); err == nil {
		t.Fatal("expected sharing with provider access to be denied when the owner does not own the provider")
	}
	if shared, err := db.IsProviderSharedWith(providerId, guest.Id); err != nil {
		t.Fatalf("IsProviderSharedWith failed: %v", err)
	} else if shared {
		t.Fatal("expected provider not to be shared with the guest")
	}
}

func Test_ShareDomain_LogsProviderStatus(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, logAppender := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	domA, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "a.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}
	domB, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "b.example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	// Case 1: sharing without provider access logs "provider not shared".
	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}
	if n := len(logAppender.logs); n == 0 || !strings.Contains(logAppender.logs[n-1].Content, "provider not shared") {
		t.Fatalf("expected a 'provider not shared' log, got %+v", logAppender.logs)
	}

	if err := service.UnshareDomain(ctx, owner, domA.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain failed: %v", err)
	}

	// Case 2: sharing with provider access, first time, logs "provider just shared".
	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}
	if n := len(logAppender.logs); n == 0 || !strings.Contains(logAppender.logs[n-1].Content, "provider just shared") {
		t.Fatalf("expected a 'provider just shared' log, got %+v", logAppender.logs)
	}

	// Case 3: sharing another domain on the same provider with the same
	// grantee logs "provider already shared".
	if _, err := service.ShareDomain(ctx, owner, domB.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}
	if n := len(logAppender.logs); n == 0 || !strings.Contains(logAppender.logs[n-1].Content, "provider already shared") {
		t.Fatalf("expected a 'provider already shared' log, got %+v", logAppender.logs)
	}
}

// Test_ShareDomain_WithProviderFalseRevokes ensures withProvider is an explicit
// intent: because the provider grant is provider-wide, sharing a second domain
// of the same provider without provider access must revoke the grant the
// grantee received through the first one, rather than silently keeping it.
func Test_ShareDomain_WithProviderFalseRevokes(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, logAppender := setupTestService(db)
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

	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if !service.CanManageProvider(guest, domA) {
		t.Fatal("expected guest to manage the provider after sharing A with it")
	}

	// Sharing B without provider access revokes the grant, provider-wide.
	if _, err := service.ShareDomain(ctx, owner, domB.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain B failed: %v", err)
	}
	if service.CanManageProvider(guest, domB) {
		t.Fatal("expected the guest to be denied provider management on B")
	}
	if service.CanManageProvider(guest, domA) {
		t.Fatal("expected the provider grant to be revoked for A too, since it is provider-wide")
	}
	if n := len(logAppender.logs); n == 0 || !strings.Contains(logAppender.logs[n-1].Content, "provider grant revoked") {
		t.Fatalf("expected a 'provider grant revoked' log, got %+v", logAppender.logs)
	}

	// Both domains remain shared: only the provider access changed.
	if _, err := service.GetUserDomain(guest, domA.Id); err != nil {
		t.Fatalf("expected the guest to keep access to domain A: %v", err)
	}
	if _, err := service.GetUserDomain(guest, domB.Id); err != nil {
		t.Fatalf("expected the guest to keep access to domain B: %v", err)
	}

	// Re-sharing A with provider access grants it back.
	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A (again) failed: %v", err)
	}
	if !service.CanManageProvider(guest, domA) {
		t.Fatal("expected the provider grant to be restored")
	}
}

// Test_ShareDomain_WithProviderFalseWithoutGrant ensures the revoke branch stays
// a no-op when the grantee holds no provider grant at all.
func Test_ShareDomain_WithProviderFalseWithoutGrant(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, logAppender := setupTestService(db)
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

	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}
	if shared, err := db.IsProviderSharedWith(providerId, guest.Id); err != nil {
		t.Fatalf("IsProviderSharedWith failed: %v", err)
	} else if shared {
		t.Fatal("expected no provider grant to appear")
	}
	if n := len(logAppender.logs); n == 0 || !strings.Contains(logAppender.logs[n-1].Content, "provider not shared") {
		t.Fatalf("expected a 'provider not shared' log, got %+v", logAppender.logs)
	}
}

// Test_GetDomainShareStatus reports both the domain's grantees and the grantees
// of its provider, the latter including users invited through another domain of
// the same provider: that is what lets the owner see, before choosing, which
// domains a provider-wide change would affect.
func Test_GetDomainShareStatus(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, _ := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	guest := createTestUser(t, db, "guest@example.com")
	stranger := createTestUser(t, db, "stranger@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	otherProviderId := createTestProvider(t, db, owner, "Other Provider")

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
	domC, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "c.example.com",
		ProviderId: otherProviderId,
	})
	if err != nil {
		t.Fatalf("failed to create domain C: %v", err)
	}

	// The guest holds the provider through A; C is on another provider.
	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if _, err := service.ShareDomain(ctx, owner, domC.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain C failed: %v", err)
	}

	// B, the domain being inspected, is not shared with anyone yet.
	status, err := service.GetDomainShareStatus(owner, domB.Id)
	if err != nil {
		t.Fatalf("GetDomainShareStatus failed: %v", err)
	}
	if len(status.Shares) != 0 {
		t.Fatalf("expected no share on domain B, got %+v", status.Shares)
	}
	if len(status.ProviderShares) != 1 {
		t.Fatalf("expected exactly one provider grantee, got %+v", status.ProviderShares)
	}
	if !status.ProviderShares[0].Id.Equals(guest.Id) || status.ProviderShares[0].Email != "guest@example.com" {
		t.Fatalf("unexpected provider grantee: %+v", status.ProviderShares[0])
	}
	// Only the domains of *this* provider are listed: C is left out.
	if domains := status.ProviderShares[0].Domains; len(domains) != 1 || !domains[0].Id.Equals(domA.Id) || domains[0].DomainName != "a.example.com." {
		t.Fatalf("expected only domain A to be listed, got %+v", domains)
	}

	// On domain A, the guest shows up in both views.
	status, err = service.GetDomainShareStatus(owner, domA.Id)
	if err != nil {
		t.Fatalf("GetDomainShareStatus failed: %v", err)
	}
	if len(status.Shares) != 1 || !status.Shares[0].Id.Equals(guest.Id) || !status.Shares[0].WithProvider {
		t.Fatalf("expected the guest with provider access, got %+v", status.Shares)
	}

	// Owner-only.
	if _, err := service.GetDomainShareStatus(guest, domA.Id); err == nil {
		t.Fatal("expected a grantee to be denied the share status")
	}
	if _, err := service.GetDomainShareStatus(stranger, domA.Id); err == nil {
		t.Fatal("expected a stranger to be denied the share status")
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

	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", false); err != nil {
		t.Fatalf("ShareDomain failed: %v", err)
	}

	// The guest can remove their own access.
	if err := service.UnshareDomain(ctx, guest, dom.Id, guest.Id); err != nil {
		t.Fatalf("expected guest self-leave to succeed: %v", err)
	}
	if _, err := service.GetUserDomain(guest, dom.Id); err == nil {
		t.Fatal("expected guest to be denied after self-leave")
	}
}

// Test_UnshareDomain_SelfLeave_NotShared ensures that a user who was never
// granted access to a domain cannot self-leave it: UnshareDomain must reject
// the call instead of silently no-opping and writing a "sharing revoked" log
// entry into a domain the caller has no relation to.
func Test_UnshareDomain_SelfLeave_NotShared(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, logAppender := setupTestService(db)
	service.SetSharingDeps(db, db)

	owner := createTestUser(t, db, "owner@example.com")
	stranger := createTestUser(t, db, "stranger@example.com")

	providerId := createTestProvider(t, db, owner, "Owner Provider")
	dom, err := service.CreateDomain(ctx, owner, &happydns.DomainCreationInput{
		DomainName: "example.com",
		ProviderId: providerId,
	})
	if err != nil {
		t.Fatalf("failed to create domain: %v", err)
	}

	logsBefore := len(logAppender.logs)

	if err := service.UnshareDomain(ctx, stranger, dom.Id, stranger.Id); err == nil {
		t.Fatal("expected self-leave to fail for a user never shared with")
	}

	if len(logAppender.logs) != logsBefore {
		t.Fatalf("expected no log entry to be written, got %+v", logAppender.logs[logsBefore:])
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
	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if _, err := service.ShareDomain(ctx, owner, domB.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain B failed: %v", err)
	}

	// Revoking A must not remove provider access still needed by B.
	if err := service.UnshareDomain(ctx, owner, domA.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain A failed: %v", err)
	}
	if !service.CanManageProvider(guest, domB) {
		t.Fatal("expected guest to keep provider access for domain B after unsharing A")
	}

	// Revoking the last domain finally drops the provider grant.
	if err := service.UnshareDomain(ctx, owner, domB.Id, guest.Id); err != nil {
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

	if _, err := service.ShareDomain(ctx, owner, dom.Id, "guest@example.com", false); err != nil {
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

	if _, err := service.ShareDomain(ctx, owner, domA.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain A failed: %v", err)
	}
	if _, err := service.ShareDomain(ctx, owner, domB.Id, "guest@example.com", true); err != nil {
		t.Fatalf("ShareDomain B failed: %v", err)
	}

	// Deleting A must not strip provider access still needed by B.
	if err := service.DeleteDomain(ctx, owner, domA.Id); err != nil {
		t.Fatalf("DeleteDomain A failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); !shared {
		t.Fatal("expected provider share to survive while domain B still needs it")
	}

	// Deleting the last shared domain drops the now-orphaned provider grant.
	if err := service.DeleteDomain(ctx, owner, domB.Id); err != nil {
		t.Fatalf("DeleteDomain B failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); shared {
		t.Fatal("expected provider share to be cleaned up once no shared domain needs it")
	}
}

// setupMismatchedProviderDomain creates a domain whose owner does not own
// its provider (e.g. created out-of-band through the admin API, or the
// provider changed hands since the grant was made), and seeds a domain share
// plus a provider grant for guest directly (bypassing ShareDomain, which
// would refuse to extend the grant to a provider the owner does not own).
func setupMismatchedProviderDomain(t *testing.T, db storage.Storage) (service *domain.Service, dom *happydns.Domain, owner, guest *happydns.User) {
	t.Helper()

	service, _ = setupTestService(db)
	service.SetSharingDeps(db, db)

	owner = createTestUser(t, db, "owner@example.com")
	other := createTestUser(t, db, "other@example.com")
	guest = createTestUser(t, db, "guest@example.com")

	// The provider belongs to a different user than the domain owner.
	providerId := createTestProvider(t, db, other, "Other's Provider")

	dom = &happydns.Domain{
		Owner:      owner.Id,
		ProviderId: providerId,
		DomainName: "mismatched.example.com",
	}
	if err := service.AdminCreateDomain(dom); err != nil {
		t.Fatalf("failed to admin-create domain: %v", err)
	}

	if err := db.AddDomainShare(dom.Id, guest.Id); err != nil {
		t.Fatalf("failed to seed domain share: %v", err)
	}
	if err := db.AddProviderShare(providerId, guest.Id); err != nil {
		t.Fatalf("failed to seed provider share: %v", err)
	}

	return service, dom, owner, guest
}

// Test_UnshareDomain_ProviderNotOwned covers a domain whose owner does not
// actually own its provider. Domain ownership alone must not be enough to
// revoke a provider grant that may belong to, and be relied on by, the
// actual provider owner.
func Test_UnshareDomain_ProviderNotOwned(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, dom, owner, guest := setupMismatchedProviderDomain(t, db)
	providerId := dom.ProviderId

	// The owner may still revoke the domain share, but must not be able to
	// touch the provider grant, since they do not own the provider.
	if err := service.UnshareDomain(ctx, owner, dom.Id, guest.Id); err != nil {
		t.Fatalf("UnshareDomain failed: %v", err)
	}
	if shared, _ := db.IsDomainSharedWith(dom.Id, guest.Id); shared {
		t.Fatal("expected the domain share to be revoked")
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); !shared {
		t.Fatal("expected the provider grant to survive, since the owner does not own the provider")
	}

	// The grantee may still self-leave, which needs no provider ownership:
	// it renounces their own grant rather than revoking someone else's.
	if err := service.UnshareDomain(ctx, guest, dom.Id, guest.Id); err != nil {
		t.Fatalf("self-leave UnshareDomain failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); shared {
		t.Fatal("expected self-leave to revoke the guest's own provider grant")
	}
}

// Test_DeleteDomain_ProviderNotOwned covers deleting a domain whose owner
// does not own its provider. The dangling domain share must be removed, but
// the provider grant, which may belong to the actual provider owner, must
// be left alone.
func Test_DeleteDomain_ProviderNotOwned(t *testing.T) {
	db, _ := inmemory.Instantiate()
	service, dom, owner, guest := setupMismatchedProviderDomain(t, db)
	providerId := dom.ProviderId

	if err := service.DeleteDomain(ctx, owner, dom.Id); err != nil {
		t.Fatalf("DeleteDomain failed: %v", err)
	}
	if shared, _ := db.IsProviderSharedWith(providerId, guest.Id); !shared {
		t.Fatal("expected the provider grant to survive, since the owner does not own the provider")
	}
}
