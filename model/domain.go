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
	"github.com/miekg/dns"
)

// DomainMinimal is used for swagger documentation as Domain add.
type DomainMinimal struct {
	// IsProvider is the identifier of the Provider used to access and edit the
	// Domain.
	IdProvider Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`
}

// Domain holds information about a domain name own by a User.
type Domain struct {
	// Id is the Domain's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string"`

	// IdUser is the identifier of the Domain's Owner.
	IdUser Identifier `json:"id_owner" swaggertype:"string"`

	// IsProvider is the identifier of the Provider used to access and edit the
	// Domain.
	IdProvider Identifier `json:"id_provider" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`

	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`

	// ZoneHistory are the identifiers to the Zone attached to the current
	// Domain.
	ZoneHistory []Identifier `json:"zone_history" swaggertype:"array,string"`
}

// Domains is an array of Domain.
type Domains []*Domain

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

// NewDomain fills a new Domain structure.
func NewDomain(u *User, st *ProviderMeta, dn string) (d *Domain) {
	d = &Domain{
		IdUser:     u.Id,
		IdProvider: st.Id,
		DomainName: dns.Fqdn(dn),
	}

	return
}
