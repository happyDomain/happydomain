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
	"context"
	"errors"
	"strings"

	"github.com/miekg/dns"
)

// DomainCreationInput is used for swagger documentation as Domain add.
type DomainCreationInput struct {
	// ProviderId is the identifier of the Provider used to access and edit the
	// Domain. It is optional: a Domain with no Provider is monitor-only (checks
	// run, but no zone is managed).
	ProviderId Identifier `json:"id_provider,omitempty" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain"`
}

// Domain holds information about a domain name own by a User.
type Domain struct {
	// Id is the Domain's identifier in the database.
	Id Identifier `json:"id" swaggertype:"string" binding:"required" readonly:"true"`

	// Owner is the identifier of the Domain's Owner.
	Owner Identifier `json:"id_owner" swaggertype:"string" binding:"required"`

	// ProviderId is the identifier of the Provider used to access and edit the
	// Domain. It is optional: a Domain with no Provider is monitor-only (checks
	// run against the public DNS/WHOIS, but no zone is managed).
	ProviderId Identifier `json:"id_provider,omitempty" swaggertype:"string"`

	// DomainName is the FQDN of the managed Domain.
	DomainName string `json:"domain" binding:"required"`

	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`

	// ZoneHistory are the identifiers to the Zone attached to the current
	// Domain.
	ZoneHistory []Identifier `json:"zone_history" swaggertype:"array,string" binding:"required" readonly:"true"`
}

// DomainUpdateInput is used for swagger documentation as Domain update.
type DomainUpdateInput struct {
	// Group is a hint string aims to group domains.
	Group string `json:"group,omitempty"`
}

// NormalizeDomainName trims, fully-qualifies and validates a domain name,
// returning the canonical FQDN or an error when the name is empty or invalid.
func NormalizeDomainName(name string) (string, error) {
	name = dns.Fqdn(strings.TrimSpace(name))

	if name == "." {
		return "", errors.New("empty domain name")
	}

	if _, ok := dns.IsDomainName(name); !ok {
		return "", errors.New("invalid domain name")
	}

	return name, nil
}

func NewDomain(user *User, name string, providerID Identifier) (*Domain, error) {
	name, err := NormalizeDomainName(name)
	if err != nil {
		return nil, err
	}

	// Nothing more is checked here: a domain name is not a path, and RFC 2317
	// classless reverse delegations (0/25.2.0.192.in-addr.arpa) legitimately
	// contain characters that are separators on a file system. The providers
	// turning a zone name into a file name, namely BIND, are the ones
	// restricting it further.

	d := &Domain{
		Owner:      user.Id,
		ProviderId: providerID,
		DomainName: name,
	}

	return d, nil
}

// IsManaged reports whether the Domain is backed by a DNS Provider. A
// monitor-only Domain (no Provider) has its checks run against the public
// DNS/WHOIS but exposes no zone management.
func (d *Domain) IsManaged() bool {
	return len(d.ProviderId) > 0
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
	LastCheckStatus *Status `json:"last_check_status,omitempty"`
}

type Subdomain string
type Origin string

// SchedulerDomainNotifier is an optional callback to notify the scheduler
// about domain changes so it can incrementally update its job queue.
type SchedulerDomainNotifier interface {
	NotifyDomainChange(domain *Domain)
	NotifyDomainRemoved(domainID Identifier)
}

type DomainUsecase interface {
	CreateDomain(context.Context, *User, *DomainCreationInput) (*Domain, error)
	DeleteDomain(Identifier) error
	ExtendsDomainWithZoneMeta(*Domain) (*DomainWithZoneMetadata, error)
	GetUserDomain(*User, Identifier) (*Domain, error)
	GetUserDomainByFQDN(*User, string) ([]*Domain, error)
	ListUserDomains(*User) ([]*Domain, error)
	UpdateDomain(Identifier, *User, func(*Domain)) error
}

// AdminDomainUsecase exposes administrative domain operations that bypass
// the caller-scoped checks of DomainUsecase. Admin callers can list every
// domain, fetch any domain by ID or by FQDN+owner, create or replace one
// from a raw Domain, and wipe the table.
type AdminDomainUsecase interface {
	ListAllDomains() ([]*Domain, error)
	GetDomainByID(Identifier) (*Domain, error)
	GetDomainsByFQDN(*User, string) ([]*Domain, error)
	AdminCreateDomain(*Domain) error
	AdminUpdateDomain(*Domain) error
	ClearDomains() error
}
