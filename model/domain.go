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

// DomainCreationInput is used for swagger documentation as Domain add.
type DomainCreationInput struct {
	// ProviderId is the identifier of the Provider used to access and edit the
	// Domain.
	ProviderId Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`
}

// Domain holds information about a domain name own by a User.
type Domain struct {
	// Id is the Domain's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string"`

	// Owner is the identifier of the Domain's Owner.
	Owner Identifier `json:"id_owner" swaggertype:"string"`

	// ProviderId is the identifier of the Provider used to access and edit the
	// Domain.
	ProviderId Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`

	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`

	// ZoneHistory are the identifiers to the Zone attached to the current
	// Domain.
	ZoneHistory []Identifier `json:"zone_history" swaggertype:"array,string"`
}

func NewDomain(user *User, name string, providerID Identifier) (*Domain, error) {
	name = dns.Fqdn(strings.TrimSpace(name))

	if name == "." {
		return nil, errors.New("empty domain name")
	}

	if _, ok := dns.IsDomainName(name); !ok {
		return nil, errors.New("invalid domain name")
	}

	d := &Domain{
		Owner:      user.Id,
		ProviderId: providerID,
		DomainName: name,
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

type DomainWithZoneMetadata struct {
	*Domain
	ZoneMeta map[string]*ZoneMeta `json:"zone_meta"`
}

type DomainWithCheckStatus struct {
	*Domain
	// LastCheckStatus is the worst status across the most recent result of each
	// checker that has run on this domain. Nil if no results exist yet.
	LastCheckStatus *CheckResultStatus `json:"last_check_status,omitempty"`
}

type Subdomain string
type Origin string

type DomainUsecase interface {
	CreateDomain(*User, *Domain) error
	DeleteDomain(Identifier) error
	ExtendsDomainWithZoneMeta(*Domain) (*DomainWithZoneMetadata, error)
	GetUserDomain(*User, Identifier) (*Domain, error)
	GetUserDomainByFQDN(*User, string) ([]*Domain, error)
	ListUserDomains(*User) ([]*Domain, error)
	UpdateDomain(Identifier, *User, func(*Domain)) error
}
