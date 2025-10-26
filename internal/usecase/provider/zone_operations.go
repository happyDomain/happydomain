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

// RetrieveZone retrieves the current zone records for the given domain from the provider.
func (s *Service) RetrieveZone(provider *happydns.Provider, name string) ([]happydns.Record, error) {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}

	return instance.GetZoneRecords(name)
}

// ListZoneCorrections lists the corrections needed to synchronize the zone with the given records.
func (s *Service) ListZoneCorrections(provider *happydns.Provider, domain *happydns.Domain, records []happydns.Record) ([]*happydns.Correction, error) {
	instance, err := provider.InstantiateProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate provider: %w", err)
	}

	return instance.GetZoneCorrections(domain.DomainName, records)
}
