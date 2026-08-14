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

// Package hostedservicetest provides shared fakes for testing usecases built
// on top of hostedservice.Store.
package hostedservicetest

import (
	"testing"

	happydns "git.happydns.org/happyDomain/model"
)

type FakeDomains struct {
	ByName map[string][]*happydns.Domain
}

func (f *FakeDomains) FindDomainsByName(fqdn string) ([]*happydns.Domain, error) {
	if domains, ok := f.ByName[fqdn]; ok {
		return domains, nil
	}
	return nil, happydns.ErrNotFound
}

type FakeZones struct {
	ByID map[string]*happydns.Zone
}

func (f *FakeZones) Get(zoneID happydns.Identifier) (*happydns.Zone, error) {
	if zone, ok := f.ByID[zoneID.String()]; ok {
		return zone, nil
	}
	return nil, happydns.ErrNotFound
}

// NewStore wires a domain/zone pair holding the given service, and returns
// the fakes backing it.
func NewStore(t *testing.T, fqdn string, service happydns.ServiceBody) (*FakeDomains, *FakeZones) {
	t.Helper()

	zoneID := happydns.Identifier{0x42}
	domain := &happydns.Domain{
		DomainName:  fqdn,
		ZoneHistory: []happydns.Identifier{zoneID},
	}
	zone := &happydns.Zone{
		ZoneMeta: happydns.ZoneMeta{Id: zoneID},
		Services: map[happydns.Subdomain][]*happydns.Service{
			"": {{Service: service}},
		},
	}

	return &FakeDomains{ByName: map[string][]*happydns.Domain{fqdn: {domain}}},
		&FakeZones{ByID: map[string]*happydns.Zone{zoneID.String(): zone}}
}
