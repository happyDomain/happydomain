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
	ListProviderShares(providerID happydns.Identifier) ([]happydns.Identifier, error)
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

	// Only fall back to shared domains if the user owns no match, to avoid
	// ambiguous FQDN collisions with domains owned by someone else.
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

// listSharedDomains loads domains shared with the user, skipping ids present
// in exclude (typically domains they already own). A grant-index read
// failure is logged and treated as "no shared domains".
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
func (s *Service) ExtendsDomainWithZoneMeta(user *happydns.User, domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
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
		Domain:            domain,
		ZoneMeta:          ret,
		CanManageProvider: s.CanManageProvider(user, domain),
	}, errs
}

// NewDomainWithCheckStatus wraps a domain for the domain listing endpoint,
// resolving CanManageProvider here so callers never assign it themselves.
// LastCheckStatus is left for the caller to fill in, as it comes from the
// checker usecase.
func (s *Service) NewDomainWithCheckStatus(user *happydns.User, domain *happydns.Domain) *happydns.DomainWithCheckStatus {
	return &happydns.DomainWithCheckStatus{
		Domain:            domain,
		CanManageProvider: s.CanManageProvider(user, domain),
	}
}

// ListUserDomains retrieves all domains for the given user.
func (s *Service) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	domains, err := s.store.ListDomains(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserDomains: %s", err.Error())
	}

	// Append domains shared with this user, skipping any they already own.
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

// getDomainForSharing loads a domain for a sharing operation, wrapping
// unexpected storage errors as InternalError. ErrDomainNotFound passes
// through as-is (the API layer maps it to a 404).
func (s *Service) getDomainForSharing(domainID happydns.Identifier) (*happydns.Domain, error) {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		if errors.Is(err, happydns.ErrDomainNotFound) {
			return nil, err
		}
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to GetDomain: %w", err),
			UserMessage: "Sorry, we are currently unable to retrieve this domain. Please retry later.",
		}
	}
	return domain, nil
}

// resolveGrantee turns an invitation reference (an identifier or an email
// address) into an actual user account.
func (s *Service) resolveGrantee(ref string) (*happydns.User, error) {
	if s.userGetter == nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("sharing dependencies not configured")}
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, happydns.ValidationError{Msg: "No user found with this identifier or email address."}
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

// ShareDomain grants a user (resolved from granteeRef, an identifier or
// email) access to the domain. Owner-only. If withProvider is set, the grant
// extends to the provider so the invitee can run zone operations. Returns
// the effective withProvider state, which may differ from the request (e.g.
// the actor doesn't own the provider).
func (s *Service) ShareDomain(ctx context.Context, actor *happydns.User, domainID happydns.Identifier, granteeRef string, withProvider bool) (*happydns.DomainShareUser, error) {
	domain, err := s.getDomainForSharing(domainID)
	if err != nil {
		return nil, err
	}
	if !actor.Id.Equals(domain.Owner) {
		return nil, happydns.ErrDomainNotFound
	}

	logFailure := func(reason string) {
		if s.domainLogAppender != nil {
			s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(actor, happydns.LOG_WARN, fmt.Sprintf("Failed attempt to share domain name %s: %s.", domain.DomainName, reason)))
		}
	}

	// Provider ownership doesn't follow from domain ownership (e.g. domains
	// created through the admin API); an actor who doesn't own the provider
	// must not be able to touch its grants.
	ownsProvider := s.actorOwnsProvider(ctx, actor, domain.ProviderId)
	if withProvider && !ownsProvider {
		logFailure("attempted to share the provider without owning it")
		return nil, happydns.ErrDomainNotFound
	}

	// Check before any write: withProvider=false may still need to revoke an
	// existing grant from another domain of the same provider.
	if s.providerSharer == nil {
		return nil, happydns.InternalError{Err: fmt.Errorf("sharing dependencies not configured")}
	}

	grantee, err := s.resolveGrantee(granteeRef)
	if err != nil {
		logFailure("could not resolve the requested grantee")
		return nil, err
	}
	if grantee.Id.Equals(domain.Owner) {
		logFailure("attempted to share with the owner")
		return nil, happydns.ValidationError{Msg: "You cannot share a domain with its owner."}
	}

	if err := s.store.AddDomainShare(domainID, grantee.Id); err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to AddDomainShare: %w", err),
			UserMessage: "Sorry, we are currently unable to share your domain. Please retry later.",
		}
	}

	alreadyShared, err := s.providerSharer.IsProviderSharedWith(domain.ProviderId, grantee.Id)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to IsProviderSharedWith: %w", err),
			UserMessage: "Sorry, we are currently unable to share your domain. Please retry later.",
		}
	}

	var providerStatus string
	effectiveWithProvider := false
	if !ownsProvider {
		// Can't grant/revoke here, but the grantee may already hold a grant
		// from another domain of the same provider; report the real state.
		effectiveWithProvider = alreadyShared
		providerStatus = "provider not managed by you"
	} else {
		switch {
		case withProvider && alreadyShared:
			providerStatus = "provider already shared"
			effectiveWithProvider = true
		case withProvider:
			if err := s.providerSharer.AddProviderShare(domain.ProviderId, grantee.Id); err != nil {
				return nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to AddProviderShare: %w", err),
					UserMessage: "The domain was shared, but we were unable to share the provider. Please retry later.",
				}
			}
			providerStatus = "provider just shared"
			effectiveWithProvider = true
		case alreadyShared:
			if err := s.providerSharer.DeleteProviderShare(domain.ProviderId, grantee.Id); err != nil {
				return nil, happydns.InternalError{
					Err:         fmt.Errorf("unable to DeleteProviderShare: %w", err),
					UserMessage: "The domain was shared, but we were unable to revoke the provider access. Please retry later.",
				}
			}
			providerStatus = "provider grant revoked, for every domain of this provider shared with them"
		default:
			providerStatus = "provider not shared"
		}
	}

	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(actor, happydns.LOG_INFO, fmt.Sprintf("Domain name %s shared with user %s (%s).", domain.DomainName, grantee.Id.String(), providerStatus)))
	}

	return &happydns.DomainShareUser{Id: grantee.Id, Email: grantee.Email, WithProvider: effectiveWithProvider}, nil
}

// UnshareDomain revokes a grantee's access to the domain. The owner may revoke
// any grantee; a grantee may remove only their own access (self-leave). The
// associated provider grant, if any, is revoked as well, but only when the
// actor owns the provider.
func (s *Service) UnshareDomain(ctx context.Context, actor *happydns.User, domainID, granteeID happydns.Identifier) error {
	domain, err := s.getDomainForSharing(domainID)
	if err != nil {
		return err
	}
	if !actor.Id.Equals(domain.Owner) && !actor.Id.Equals(granteeID) {
		return happydns.ErrDomainNotFound
	}

	shared, serr := s.store.IsDomainSharedWith(domainID, granteeID)
	if serr != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to IsDomainSharedWith: %w", serr),
			UserMessage: "Sorry, we are currently unable to update the sharing. Please retry later.",
		}
	}

	// Only touch the provider grant if the owner actually owns the provider
	// (see actorOwnsProvider). Self-leave is exempt: revoking your own grant
	// needs no provider ownership.
	selfLeave := actor.Id.Equals(granteeID)
	ownsProvider := selfLeave || s.actorOwnsProvider(ctx, actor, domain.ProviderId)

	hadProviderGrant := false
	if s.providerSharer != nil && ownsProvider {
		var perr error
		hadProviderGrant, perr = s.providerSharer.IsProviderSharedWith(domain.ProviderId, granteeID)
		if perr != nil {
			return happydns.InternalError{
				Err:         fmt.Errorf("unable to IsProviderSharedWith: %w", perr),
				UserMessage: "Sorry, we are currently unable to update the sharing. Please retry later.",
			}
		}
	}

	// Checked before any write, so a stale request 404s instead of a no-op success.
	if !shared && !hadProviderGrant {
		return happydns.ErrDomainNotFound
	}

	if shared {
		if err := s.store.DeleteDomainShare(domainID, granteeID); err != nil {
			return happydns.InternalError{
				Err:         fmt.Errorf("unable to DeleteDomainShare: %w", err),
				UserMessage: "Sorry, we are currently unable to update the sharing. Please retry later.",
			}
		}
	}

	// Only revoke the provider grant if no other shared domain on the same
	// provider still needs it (provider shares are keyed by (provider,
	// grantee), not per-domain). DeleteProviderShare is idempotent, so a
	// retry after a partial failure safely finishes the cleanup.
	providerRevoked := false
	if hadProviderGrant && !s.providerStillNeededBy(granteeID, domain.ProviderId) {
		if err := s.providerSharer.DeleteProviderShare(domain.ProviderId, granteeID); err != nil {
			return happydns.InternalError{
				Err:         fmt.Errorf("unable to DeleteProviderShare: %w", err),
				UserMessage: "Sorry, we are currently unable to fully revoke the sharing. Please retry later.",
			}
		}
		providerRevoked = true
	}

	if s.domainLogAppender != nil {
		who := "owner"
		if actor.Id.Equals(granteeID) {
			who = "grantee (self-leave)"
		}
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(actor, happydns.LOG_INFO, fmt.Sprintf("Domain name %s sharing revoked for user %s by %s; provider grant revoked: %t.", domain.DomainName, granteeID.String(), who, providerRevoked)))
	}

	return nil
}

// actorOwnsProvider reports whether the actor owns the domain's provider.
// Domain ownership doesn't imply provider ownership (e.g. domains created
// through the admin API).
func (s *Service) actorOwnsProvider(ctx context.Context, actor *happydns.User, providerID happydns.Identifier) bool {
	_, err := s.providerService.GetUserProvider(ctx, actor, providerID)
	return err == nil
}

// providerStillNeededBy reports whether any domain still shared with the
// grantee uses the given provider. Call after removing the domain share so
// that domain isn't counted.
func (s *Service) providerStillNeededBy(granteeID, providerID happydns.Identifier) bool {
	domains, err := s.sharedDomainsOnProvider(granteeID, providerID)
	if err != nil {
		// Keep the grant on a lookup error rather than risk revoking access
		// another shared domain still depends on.
		log.Printf("providerStillNeededBy: unable to list shared domains for grantee %s: %v", granteeID.String(), err)
		return true
	}
	return len(domains) > 0
}

// sharedDomainsOnProvider lists the domains shared with the grantee that use
// the given provider (provider shares are keyed by (provider, grantee), not
// per-domain).
func (s *Service) sharedDomainsOnProvider(granteeID, providerID happydns.Identifier) ([]happydns.DomainShareRef, error) {
	sharedIDs, err := s.store.ListSharedDomainIDs(granteeID)
	if err != nil {
		return nil, err
	}

	ret := []happydns.DomainShareRef{}
	for _, id := range sharedIDs {
		d, gerr := s.store.GetDomain(id)
		if gerr != nil {
			continue
		}
		if d.ProviderId.Equals(providerID) {
			ret = append(ret, happydns.DomainShareRef{Id: d.Id, DomainName: d.DomainName})
		}
	}
	return ret, nil
}

// GetDomainShareStatus returns the domain's grantees, whether each also has
// provider access, and the provider's own grantees (who may hold it through
// another domain). Owner-only.
func (s *Service) GetDomainShareStatus(actor *happydns.User, domainID happydns.Identifier) (*happydns.DomainShareStatus, error) {
	domain, err := s.getDomainForSharing(domainID)
	if err != nil {
		return nil, err
	}
	if !actor.Id.Equals(domain.Owner) {
		return nil, happydns.ErrDomainNotFound
	}

	ids, err := s.store.ListDomainShares(domainID)
	if err != nil {
		return nil, happydns.InternalError{
			Err:         fmt.Errorf("unable to ListDomainShares: %w", err),
			UserMessage: "Sorry, we are currently unable to retrieve the sharing status. Please retry later.",
		}
	}

	status := &happydns.DomainShareStatus{
		Shares:         []*happydns.DomainShareUser{},
		ProviderShares: []*happydns.ProviderShareUser{},
	}
	if s.userGetter == nil {
		return status, nil
	}

	for _, id := range ids {
		u, uerr := s.userGetter.GetUser(id)
		if uerr != nil {
			log.Printf("GetDomainShareStatus: stale share index -> missing user %s: %v", id.String(), uerr)
			continue
		}

		var withProvider bool
		if s.providerSharer != nil {
			withProvider, err = s.providerSharer.IsProviderSharedWith(domain.ProviderId, u.Id)
			if err != nil {
				log.Printf("GetDomainShareStatus: unable to check provider share for grantee %s: %v", u.Id.String(), err)
			}
		}

		status.Shares = append(status.Shares, &happydns.DomainShareUser{Id: u.Id, Email: u.Email, WithProvider: withProvider})
	}

	if s.providerSharer == nil {
		return status, nil
	}

	prvdGrantees, err := s.providerSharer.ListProviderShares(domain.ProviderId)
	if err != nil {
		log.Printf("GetDomainShareStatus: unable to list provider shares for provider %s: %v", domain.ProviderId.String(), err)
		return status, nil
	}

	for _, id := range prvdGrantees {
		u, uerr := s.userGetter.GetUser(id)
		if uerr != nil {
			log.Printf("GetDomainShareStatus: stale provider share index -> missing user %s: %v", id.String(), uerr)
			continue
		}

		domains, derr := s.sharedDomainsOnProvider(u.Id, domain.ProviderId)
		if derr != nil {
			log.Printf("GetDomainShareStatus: unable to list shared domains for grantee %s: %v", u.Id.String(), derr)
			domains = []happydns.DomainShareRef{}
		}

		status.ProviderShares = append(status.ProviderShares, &happydns.ProviderShareUser{Id: u.Id, Email: u.Email, Domains: domains})
	}

	return status, nil
}

// CanManageProvider reports whether the user may run provider-backed zone
// operations (retrieve/apply/diff) on the domain: true for the owner, or for an
// invitee the provider was shared with.
func (s *Service) CanManageProvider(user *happydns.User, domain *happydns.Domain) bool {
	if user == nil {
		return false
	}
	if user.Id.Equals(domain.Owner) {
		return true
	}
	if s.providerSharer == nil {
		return false
	}
	shared, err := s.providerSharer.IsProviderSharedWith(domain.ProviderId, user.Id)
	return err == nil && shared
}

// Update updates a domain using the provided update function. Owner-only: a
// grantee must not be able to alter the domain's own properties.
func (s *Service) Update(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	domain, err := s.GetUserDomain(user, domainID)
	if err != nil {
		return err
	}

	if !user.Id.Equals(domain.Owner) {
		return happydns.ForbiddenError{Msg: "you are not allowed to update this domain"}
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

// DeleteDomain deletes a domain by ID. Owner-only: an invited (shared) user
// must not be able to stop managing the domain for everyone; they remove
// their own access via UnshareDomain instead.
func (s *Service) DeleteDomain(ctx context.Context, actor *happydns.User, domainID happydns.Identifier) error {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return err
	}
	if !actor.Id.Equals(domain.Owner) {
		return happydns.ErrDomainNotFound
	}

	// Only clean up shared users' provider grants when the owner actually
	// owns the provider (see actorOwnsProvider).
	return s.deleteDomain(domain, s.actorOwnsProvider(ctx, actor, domain.ProviderId))
}

// deleteDomain performs the actual deletion, without any ownership check.
// Shared by the owner-checked DeleteDomain and the unchecked
// AdminDeleteDomain, which always passes revokeProviderShares=true.
func (s *Service) deleteDomain(domain *happydns.Domain, revokeProviderShares bool) error {
	domainID := domain.Id

	// Capture the grantees before deletion: their provider shares (keyed by
	// (provider, grantee), not per-domain) must be cleaned up separately.
	var grantees []happydns.Identifier
	if s.providerSharer != nil && revokeProviderShares {
		var lerr error
		grantees, lerr = s.store.ListDomainShares(domainID)
		if lerr != nil {
			// Nothing left afterwards to reconcile an orphaned provider grant
			// once the domain-share index is gone; log so the leak is visible.
			log.Printf("deleteDomain: unable to list domain shares for %s, provider grants may be orphaned: %v", domainID.String(), lerr)
		}
	}
	providerID := domain.ProviderId

	err := s.store.DeleteDomain(domainID)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomain: %w", err),
			UserMessage: fmt.Sprintf("unable to delete your domain: %s", err.Error()),
		}
	}

	// Drop each grantee's provider grant unless another shared domain still relies on it.
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
