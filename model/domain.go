// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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

package happydns

import (
	"errors"
	"strings"

	"github.com/miekg/dns"
)

func NewDomain(user *User, name string, providerID Identifier) (*Domain, error) {
	name = dns.Fqdn(strings.TrimSpace(name))

	if name == "" {
		return nil, errors.New("empty domain name")
	}

	if _, ok := dns.IsDomainName(name); !ok {
		return nil, errors.New("invalid domain name")
	}

	d := &Domain{
		IdOwner:    user.Id,
		IdProvider: providerID,
		Domain:     name,
	}

	return d, nil
}

// HasZone checks if the given Zone's identifier is part of this Domain
// history.
func (d *Domain) HasZone(zoneId Identifier) (found bool) {
	for _, v := range d.ZoneHistory {
		if v.Equals(zoneId) {
			return true
		}
	}
	return
}

type Origin string

func NewDomainWithZoneMetadata(domain *Domain, meta map[string]*ZoneMeta) *DomainWithZoneMetadata {
	return &DomainWithZoneMetadata{
		Id:          domain.Id,
		IdOwner:     domain.IdOwner,
		IdProvider:  domain.IdProvider,
		Domain:      domain.Domain,
		Group:       domain.Group,
		ZoneHistory: domain.ZoneHistory,
		ZoneMeta:    meta,
	}
}

type DomainUsecase interface {
	CreateDomain(*User, *Domain) error
	DeleteDomain(Identifier) error
	ExtendsDomainWithZoneMeta(*Domain) (*DomainWithZoneMetadata, error)
	GetUserDomain(*User, Identifier) (*Domain, error)
	GetUserDomainByFQDN(*User, string) ([]*Domain, error)
	ListUserDomains(*User) ([]*Domain, error)
	UpdateDomain(Identifier, *User, func(*Domain)) error
}
