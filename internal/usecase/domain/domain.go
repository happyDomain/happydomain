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

package domain

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/miekg/dns"

	domainLogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// ProviderGetter is an interface for getting providers.
type ProviderGetter interface {
	GetUserProvider(ctx context.Context, user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error)
}

// UserGetter resolves users by identifier or email, used to turn an invitation
// reference into an actual account when sharing a domain.
type UserGetter interface {
	GetUser(happydns.Identifier) (*happydns.User, error)
	GetUserByEmail(string) (*happydns.User, error)
}

// ProviderSharer manages provider-level sharing grants. Sharing a domain can
// optionally extend the grant to its provider so the invitee may run zone
// operations (retrieve/apply/diff) that need the provider credentials.
type ProviderSharer interface {
	AddProviderShare(providerID, granteeID happydns.Identifier) error
	DeleteProviderShare(providerID, granteeID happydns.Identifier) error
	IsProviderSharedWith(providerID, granteeID happydns.Identifier) (bool, error)
}

// DomainExistenceTester is an interface for testing domain existence.
type DomainExistenceTester interface {
	TestDomainExistence(ctx context.Context, provider *happydns.Provider, name string) error
}

type Service struct {
	store             DomainStorage
	providerService   ProviderGetter
	getZone           *zoneUC.GetZoneUsecase
	domainExistence   DomainExistenceTester
	domainLogAppender domainLogUC.DomainLogAppender
	schedulerNotifier happydns.SchedulerDomainNotifier
	userGetter        UserGetter
	providerSharer    ProviderSharer
}

func NewService(
	store DomainStorage,
	providerService ProviderGetter,
	getZone *zoneUC.GetZoneUsecase,
	domainExistence DomainExistenceTester,
	domainLogAppender domainLogUC.DomainLogAppender,
) *Service {
	return &Service{
		store:             store,
		providerService:   providerService,
		getZone:           getZone,
		domainExistence:   domainExistence,
		domainLogAppender: domainLogAppender,
	}
}

// SetSchedulerNotifier sets the optional scheduler notifier for incremental
// queue updates on domain creation/deletion.
func (s *Service) SetSchedulerNotifier(notifier happydns.SchedulerDomainNotifier) {
	s.schedulerNotifier = notifier
}

// SetSharingDeps wires the dependencies needed by the domain sharing feature:
// a user resolver (to turn an invitation reference into an account) and a
// provider sharer (to optionally extend the grant to the domain's provider).
func (s *Service) SetSharingDeps(users UserGetter, providers ProviderSharer) {
	s.userGetter = users
	s.providerSharer = providers
}

// CreateDomain creates a new domain for the given user.
func (s *Service) CreateDomain(ctx context.Context, user *happydns.User, input *happydns.DomainCreationInput) (*happydns.Domain, error) {
	uz, err := happydns.NewDomain(user, input.DomainName, input.ProviderId)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerService.GetUserProvider(ctx, user, uz.ProviderId)
	if err != nil {
		return nil, happydns.ValidationError{Msg: fmt.Sprintf("unable to find the provider.")}
	}

	if err = s.domainExistence.TestDomainExistence(ctx, provider, uz.DomainName); err != nil {
		return nil, happydns.NotFoundError{Msg: err.Error()}
	}

	if err := s.store.CreateDomain(uz); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateDomain: %s", err),
			UserMessage: "Sorry, we are unable to create your domain now.",
		}
	}

	// Add a log entry
	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(uz, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s added.", uz.DomainName)))
	}

	if s.schedulerNotifier != nil {
		s.schedulerNotifier.NotifyDomainChange(uz)
	}

	return uz, nil
}

// GetUserDomain retrieves a domain by ID for the given user.
func (s *Service) GetUserDomain(user *happydns.User, domainID happydns.Identifier) (*happydns.Domain, error) {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(domain.Owner) {
		shared, serr := s.store.IsDomainSharedWith(domainID, user.Id)
		if serr != nil || !shared {
			return nil, happydns.ErrDomainNotFound
		}
	}

	return domain, nil
}

// GetUserDomainByFQDN retrieves domains by FQDN for the given user, matching
// both owned domains and domains shared with the user, so a shared domain can
// be resolved by name just like an owned one.
func (s *Service) GetUserDomainByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	domains, err := s.store.GetDomainByDN(user, fqdn)
	if err != nil && !errors.Is(err, happydns.ErrNotFound) && !errors.Is(err, happydns.ErrDomainNotFound) {
		return nil, err
	}

	seen := make(map[string]bool, len(domains))
	for _, d := range domains {
		seen[d.Id.String()] = true
	}

	// If the user already owns a domain matching this FQDN, don't also pull in
	// shared domains of the same name: a shared domain owned by someone else
	// could otherwise collide with the user's own and make the FQDN ambiguous.
	if len(domains) == 0 {
		target := dns.Fqdn(strings.TrimSpace(fqdn))
		for _, d := range s.listSharedDomains(user, seen) {
			if strings.EqualFold(dns.Fqdn(d.DomainName), target) {
				domains = append(domains, d)
			}
		}
	}

	if len(domains) == 0 {
		return nil, happydns.ErrDomainNotFound
	}

	return domains, nil
}

// listSharedDomains loads the domains shared with the user, skipping any whose
// identifier is present in the exclude set (typically the domains they already
// own). A failure to read the grant index is logged and treated as "no shared
// domains" so the caller still gets the user's owned domains rather than an
// error. Loading each domain by id follows the same per-key access pattern as
// the storage layer's own ListDomains: the grant index holds only identifiers.
func (s *Service) listSharedDomains(user *happydns.User, exclude map[string]bool) []*happydns.Domain {
	sharedIDs, err := s.store.ListSharedDomainIDs(user.Id)
	if err != nil {
		log.Printf("listSharedDomains: unable to list shared domains for user %s: %v", user.Id.String(), err)
		return nil
	}

	var ret []*happydns.Domain
	for _, id := range sharedIDs {
		if exclude != nil && exclude[id.String()] {
			continue
		}
		d, gerr := s.store.GetDomain(id)
		if gerr != nil {
			log.Printf("listSharedDomains: stale grant index -> missing domain %s: %v", id.String(), gerr)
			continue
		}
		ret = append(ret, d)
	}
	return ret
}

// ExtendsDomainWithZoneMeta extends a domain with zone metadata.
func (s *Service) ExtendsDomainWithZoneMeta(domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	var errs error
	ret := map[string]*happydns.ZoneMeta{}

	for _, zm := range domain.ZoneHistory {
		zoneMeta, err := s.getZone.GetMeta(zm)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to retrieve zone meta history for %q: %w", domain.DomainName, err))
		} else {
			ret[zm.String()] = zoneMeta
		}
	}

	return &happydns.DomainWithZoneMetadata{
		Domain:   domain,
		ZoneMeta: ret,
	}, errs
}

// ListUserDomains retrieves all domains for the given user.
func (s *Service) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	domains, err := s.store.ListDomains(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserDomains: %s", err.Error())
	}

	// Append domains shared with this user, skipping any they already own. A
	// failure to read the grant index is logged (inside the helper) rather than
	// silently dropping the shared domains.
	seen := make(map[string]bool, len(domains))
	for _, d := range domains {
		seen[d.Id.String()] = true
	}
	domains = append(domains, s.listSharedDomains(user, seen)...)

	if len(domains) == 0 {
		return []*happydns.Domain{}, nil
	}

	return domains, nil
}

// resolveGrantee turns an invitation reference (an identifier or an email
// address) into an actual user account.
func (s *Service) resolveGrantee(ref string) (*happydns.User, error) {
	if s.userGetter == nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("sharing dependencies not configured")}
	}
	if id, err := happydns.NewIdentifierFromString(ref); err == nil {
		if u, uerr := s.userGetter.GetUser(id); uerr == nil {
			return u, nil
		}
	}
	u, err := s.userGetter.GetUserByEmail(ref)
	if err != nil {
		return nil, happydns.ValidationError{Msg: "No user found with this identifier or email address."}
	}
	return u, nil
}

// ShareDomain grants another user (resolved from granteeRef, an identifier or
// email) access to the domain. Only the owner may share. When withProvider is
// set and the domain is managed, the grant is extended to the provider so the
// invitee can run zone operations that need the provider credentials. It
// returns the resolved grantee.
func (s *Service) ShareDomain(actor *happydns.User, domainID happydns.Identifier, granteeRef string, withProvider bool) (*happydns.User, error) {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return nil, err
	}
	if !actor.Id.Equals(domain.Owner) {
		return nil, happydns.ErrDomainNotFound
	}

	grantee, err := s.resolveGrantee(granteeRef)
	if err != nil {
		return nil, err
	}
	if grantee.Id.Equals(domain.Owner) {
		return nil, happydns.ValidationError{Msg: "You cannot share a domain with its owner."}
	}

	if err := s.store.AddDomainShare(domainID, grantee.Id); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to AddDomainShare: %w", err),
			UserMessage: "Sorry, we are currently unable to share your domain. Please retry later.",
		}
	}

	providerStatus := "provider not shared"
	if withProvider && s.providerSharer != nil {
		alreadyShared, err := s.providerSharer.IsProviderSharedWith(domain.ProviderId, grantee.Id)
		if err != nil {
			return nil, happydns.InternalError{
				Err:         fmt.Errorf("unable to IsProviderSharedWith: %w", err),
				UserMessage: "Sorry, we are currently unable to share your domain. Please retry later.",
			}
		}

		if alreadyShared {
			providerStatus = "provider already shared"
		} else {
			if err := s.providerSharer.AddProviderShare(domain.ProviderId, grantee.Id); err != nil {
				return nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to AddProviderShare: %w", err),
					UserMessage: "The domain was shared, but we were unable to share the provider. Please retry later.",
				}
			}
			providerStatus = "provider just shared"
		}
	}

	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(actor, happydns.LOG_INFO, fmt.Sprintf("Domain name %s shared with user %s (%s).", domain.DomainName, grantee.Id.String(), providerStatus)))
	}

	return grantee, nil
}

// UnshareDomain revokes a grantee's access to the domain. The owner may revoke
// any grantee; a grantee may remove only their own access (self-leave). The
// associated provider grant, if any, is revoked as well.
func (s *Service) UnshareDomain(actor *happydns.User, domainID, granteeID happydns.Identifier) error {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return err
	}
	if !actor.Id.Equals(domain.Owner) && !actor.Id.Equals(granteeID) {
		return happydns.ErrDomainNotFound
	}

	if err := s.store.DeleteDomainShare(domainID, granteeID); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomainShare: %w", err),
			UserMessage: "Sorry, we are currently unable to update the sharing. Please retry later.",
		}
	}

	// Revoke the provider grant, but only if no other domain still shared with
	// this grantee is backed by the same provider. Provider shares are keyed by
	// (provider, grantee) with no domain component, so a blind delete would
	// break provider access for every other shared domain on that provider.
	if s.providerSharer != nil && !s.providerStillNeededBy(granteeID, domain.ProviderId) {
		if err := s.providerSharer.DeleteProviderShare(domain.ProviderId, granteeID); err != nil {
			log.Printf("UnshareDomain: failed to delete provider share for grantee %s: %v", granteeID.String(), err)
		}
	}

	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(actor, happydns.LOG_INFO, fmt.Sprintf("Domain name %s sharing revoked.", domain.DomainName)))
	}

	return nil
}

// providerStillNeededBy reports whether any domain currently shared with the
// grantee is backed by the given provider. It is used to decide whether a
// provider grant may be revoked: because provider shares are keyed only by
// (provider, grantee), the grant must survive as long as one shared domain
// still relies on it. Call it after the domain share has been removed so the
// domain being revoked is no longer counted.
func (s *Service) providerStillNeededBy(granteeID, providerID happydns.Identifier) bool {
	sharedIDs, err := s.store.ListSharedDomainIDs(granteeID)
	if err != nil {
		// On a lookup error, keep the provider grant rather than risk revoking
		// access another shared domain still depends on.
		log.Printf("providerStillNeededBy: unable to list shared domains for grantee %s: %v", granteeID.String(), err)
		return true
	}
	for _, id := range sharedIDs {
		d, gerr := s.store.GetDomain(id)
		if gerr != nil {
			continue
		}
		if d.ProviderId.Equals(providerID) {
			return true
		}
	}
	return false
}

// ListDomainShares returns the users the domain is shared with, along with
// whether each grantee also received access to the Domain's Provider.
// Owner-only.
func (s *Service) ListDomainShares(actor *happydns.User, domainID happydns.Identifier) ([]*happydns.DomainShareUser, error) {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return nil, err
	}
	if !actor.Id.Equals(domain.Owner) {
		return nil, happydns.ErrDomainNotFound
	}

	ids, err := s.store.ListDomainShares(domainID)
	if err != nil {
		return nil, err
	}

	ret := []*happydns.DomainShareUser{}
	if s.userGetter == nil {
		return ret, nil
	}
	for _, id := range ids {
		u, uerr := s.userGetter.GetUser(id)
		if uerr != nil {
			log.Printf("ListDomainShares: stale share index -> missing user %s: %v", id.String(), uerr)
			continue
		}

		var withProvider bool
		if s.providerSharer != nil {
			withProvider, err = s.providerSharer.IsProviderSharedWith(domain.ProviderId, u.Id)
			if err != nil {
				log.Printf("ListDomainShares: unable to check provider share for grantee %s: %v", u.Id.String(), err)
			}
		}

		ret = append(ret, &happydns.DomainShareUser{Id: u.Id, Email: u.Email, WithProvider: withProvider})
	}
	return ret, nil
}

// CanManageProvider reports whether the user may run provider-backed zone
// operations (retrieve/apply/diff) on the domain: true for the owner, or for an
// invitee the provider was shared with.
func (s *Service) CanManageProvider(user *happydns.User, domain *happydns.Domain) bool {
	if user.Id.Equals(domain.Owner) {
		return true
	}
	if s.providerSharer == nil {
		return false
	}
	shared, err := s.providerSharer.IsProviderSharedWith(domain.ProviderId, user.Id)
	return err == nil && shared
}

// Update updates a domain using the provided update function.
func (s *Service) Update(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	domain, err := s.GetUserDomain(user, domainID)
	if err != nil {
		return err
	}

	updateFn(domain)
	//domain.ModifiedOn = time.Now()

	if !domain.Id.Equals(domainID) {
		return happydns.ValidationError{Msg: "you cannot change the domain identifier"}
	}

	err = s.store.UpdateDomain(domain)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to UpdateDomain in UpdateDomain: %w", err),
			UserMessage: "Sorry, we are currently unable to update your domain. Please retry later.",
		}
	}

	// Add a log entry
	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOG_INFO, fmt.Sprintf("Domain name %s properties changed.", domain.DomainName)))
	}

	return nil
}

// UpdateDomain is an alias for Update for backward compatibility.
func (s *Service) UpdateDomain(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	return s.Update(domainID, user, updateFn)
}

// DeleteDomain deletes a domain by ID.
func (s *Service) DeleteDomain(domainID happydns.Identifier) error {
	// Capture the domain and its grantees before deletion: the storage layer
	// removes the domain-share index entries, but the associated provider
	// shares (keyed by (provider, grantee), with no domain component) must be
	// cleaned up here, each revoked only when no other shared domain needs it.
	var providerID happydns.Identifier
	var grantees []happydns.Identifier
	if domain, derr := s.store.GetDomain(domainID); derr == nil && s.providerSharer != nil {
		providerID = domain.ProviderId
		grantees, _ = s.store.ListDomainShares(domainID)
	}

	err := s.store.DeleteDomain(domainID)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomain: %w", err),
			UserMessage: fmt.Sprintf("unable to delete your domain: %s", err.Error()),
		}
	}

	// Now that the domain (and its share/grant index entries) is gone, drop
	// each grantee's provider grant unless another shared domain still relies
	// on it.
	for _, granteeID := range grantees {
		if s.providerStillNeededBy(granteeID, providerID) {
			continue
		}
		if err := s.providerSharer.DeleteProviderShare(providerID, granteeID); err != nil {
			log.Printf("DeleteDomain: failed to delete provider share for grantee %s: %v", granteeID.String(), err)
		}
	}

	if s.schedulerNotifier != nil {
		s.schedulerNotifier.NotifyDomainRemoved(domainID)
	}

	return nil
}
