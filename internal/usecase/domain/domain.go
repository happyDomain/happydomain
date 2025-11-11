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
	"errors"
	"fmt"

	domainLogUC "git.happydns.org/happyDomain/internal/usecase/domain_log"
	zoneUC "git.happydns.org/happyDomain/internal/usecase/zone"
	"git.happydns.org/happyDomain/model"
)

// ProviderGetter is an interface for getting providers.
type ProviderGetter interface {
	GetUserProvider(user *happydns.User, providerID happydns.Identifier) (*happydns.Provider, error)
}

// DomainExistenceTester is an interface for testing domain existence.
type DomainExistenceTester interface {
	TestDomainExistence(provider *happydns.Provider, name string) error
}

type Service struct {
	store             DomainStorage
	providerService   ProviderGetter
	getZone           *zoneUC.GetZoneUsecase
	domainExistence   DomainExistenceTester
	domainLogAppender domainLogUC.DomainLogAppender
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

// CreateDomain creates a new domain for the given user.
func (s *Service) CreateDomain(user *happydns.User, uz *happydns.Domain) error {
	uz, err := happydns.NewDomain(user, uz.Domain, uz.IdProvider)
	if err != nil {
		return err
	}

	provider, err := s.providerService.GetUserProvider(user, uz.IdProvider)
	if err != nil {
		return happydns.ValidationError{Msg: fmt.Sprintf("unable to find the provider.")}
	}

	if err = s.domainExistence.TestDomainExistence(provider, uz.Domain); err != nil {
		return happydns.NotFoundError{Msg: err.Error()}
	}

	if err := s.store.CreateDomain(uz); err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to CreateDomain: %s", err),
			UserMessage: "Sorry, we are unable to create your domain now.",
		}
	}

	// Add a log entry
	if s.domainLogAppender != nil {
		s.domainLogAppender.AppendDomainLog(uz, happydns.NewDomainLog(user, happydns.LOGINFO, fmt.Sprintf("Domain name %s added.", uz.Domain)))
	}

	return nil
}

// GetUserDomain retrieves a domain by ID for the given user.
func (s *Service) GetUserDomain(user *happydns.User, domainID happydns.Identifier) (*happydns.Domain, error) {
	domain, err := s.store.GetDomain(domainID)
	if err != nil {
		return nil, err
	}

	if !user.Id.Equals(domain.IdOwner) {
		return nil, happydns.ErrDomainNotFound
	}

	return domain, nil
}

// GetUserDomainByFQDN retrieves domains by FQDN for the given user.
func (s *Service) GetUserDomainByFQDN(user *happydns.User, fqdn string) ([]*happydns.Domain, error) {
	return s.store.GetDomainByDN(user, fqdn)
}

// ExtendsDomainWithZoneMeta extends a domain with zone metadata.
func (s *Service) ExtendsDomainWithZoneMeta(domain *happydns.Domain) (*happydns.DomainWithZoneMetadata, error) {
	var errs error
	ret := map[string]*happydns.ZoneMeta{}

	for _, zm := range domain.ZoneHistory {
		zoneMeta, err := s.getZone.GetMeta(zm)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to retrieve zone meta history for %q: %w", domain.Domain, err))
		} else {
			ret[zm.String()] = zoneMeta
		}
	}

	return happydns.NewDomainWithZoneMetadata(domain, ret), errs
}

// ListUserDomains retrieves all domains for the given user.
func (s *Service) ListUserDomains(user *happydns.User) ([]*happydns.Domain, error) {
	domains, err := s.store.ListDomains(user)
	if err != nil {
		return nil, fmt.Errorf("an error occurs when trying to GetUserDomains: %s", err.Error())
	}

	if len(domains) == 0 {
		return []*happydns.Domain{}, nil
	}

	return domains, nil
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
		s.domainLogAppender.AppendDomainLog(domain, happydns.NewDomainLog(user, happydns.LOGINFO, fmt.Sprintf("Domain name %s properties changed.", domain.Domain)))
	}

	return nil
}

// UpdateDomain is an alias for Update for backward compatibility.
func (s *Service) UpdateDomain(domainID happydns.Identifier, user *happydns.User, updateFn func(*happydns.Domain)) error {
	return s.Update(domainID, user, updateFn)
}

// DeleteDomain deletes a domain by ID.
func (s *Service) DeleteDomain(domainID happydns.Identifier) error {
	err := s.store.DeleteDomain(domainID)
	if err != nil {
		return happydns.InternalError{
			Err:         fmt.Errorf("unable to DeleteDomain: %w", err),
			UserMessage: fmt.Sprintf("unable to delete your domain: %s", err.Error()),
		}
	}

	return nil
}
