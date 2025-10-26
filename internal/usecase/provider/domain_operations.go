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

package provider

import (
	"fmt"

	"git.happydns.org/happyDomain/model"
)

// CreateDomainOnProvider creates a domain on the given provider.
func (s *Service) CreateDomainOnProvider(provider *happydns.Provider, fqdn string) error {
	p, err := provider.InstantiateProvider()
	if err != nil {
		return fmt.Errorf("unable to instantiate the provider: %w", err)
	}

	if !p.CanCreateDomain() {
		return fmt.Errorf("the provider doesn't support domain creation")
	}

	return p.CreateDomain(fqdn)
}

// CreateDomainOnProvider for RestrictedService enforces configuration restrictions.
func (s *RestrictedService) CreateDomainOnProvider(provider *happydns.Provider, fqdn string) error {
	if s.config.DisableProviders {
		return happydns.ForbiddenError{Msg: "cannot create domain on provider as DisableProviders parameter is set."}
	}

	return s.Service.CreateDomainOnProvider(provider, fqdn)
}

// ListHostedDomains lists all domains hosted on the given provider.
func (s *Service) ListHostedDomains(provider *happydns.Provider) ([]string, error) {
	p, err := provider.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate the provider: %w", err)
	}

	if !p.CanListZones() {
		return nil, fmt.Errorf("the provider doesn't support domain listing")
	}

	return p.ListZones()
}

// TestDomainExistence tests whether a domain exists on the given provider.
func (s *Service) TestDomainExistence(provider *happydns.Provider, name string) error {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return fmt.Errorf("unable to instantiate provider: %w", err)
	}

	_, err = instance.GetZoneRecords(name)
	return err
}
