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

// Package hostedservice holds what every service happyDomain serves content
// for (mail-client auto-configuration, MTA-STS policies, …) needs in common:
// finding, from a public FQDN, the service body configured in the latest zone
// of whichever domain owns that name.
package hostedservice

import (
	"errors"
	"strings"

	"github.com/miekg/dns"

	happydns "git.happydns.org/happyDomain/model"
)

// DomainFinder looks up Domains by FQDN across all users.
type DomainFinder interface {
	FindDomainsByName(fqdn string) ([]*happydns.Domain, error)
}

// ZoneGetter retrieves a Zone by its identifier.
type ZoneGetter interface {
	Get(zoneID happydns.Identifier) (*happydns.Zone, error)
}

// Store bundles the two storage adapters a hosted service needs.
type Store struct {
	Domains DomainFinder
	Zones   ZoneGetter
}

// NewStore wires a Store to the given storage adapters.
func NewStore(domains DomainFinder, zones ZoneGetter) Store {
	return Store{Domains: domains, Zones: zones}
}

// StripPrefix removes the first of the given host prefixes that fqdn carries,
// returning the domain the hosted content applies to and whether a prefix was
// actually found. The returned name is always fully qualified.
func StripPrefix(fqdn string, prefixes ...string) (string, bool) {
	fqdn = dns.Fqdn(fqdn)
	for _, prefix := range prefixes {
		if len(fqdn) >= len(prefix) && strings.EqualFold(fqdn[:len(prefix)], prefix) {
			return fqdn[len(prefix):], true
		}
	}
	return fqdn, false
}

// FindService walks every owner of the given domain, loads the latest zone,
// and returns the first service of type T found in it.
//
// Returns happydns.ErrNotFound if no domain matches or none has the service.
func FindService[T happydns.ServiceBody](s Store, fqdn string) (T, error) {
	var zero T

	domains, err := s.Domains.FindDomainsByName(fqdn)
	if err != nil {
		return zero, err
	}

	for _, d := range domains {
		if len(d.ZoneHistory) == 0 {
			continue
		}
		zone, err := s.Zones.Get(d.ZoneHistory[0])
		if err != nil {
			continue
		}
		for _, services := range zone.Services {
			for _, svc := range services {
				if body, ok := svc.Service.(T); ok {
					return body, nil
				}
			}
		}
	}

	return zero, happydns.ErrNotFound
}

// hosted is implemented by service bodies that can be configured for the DNS
// half only, without asking happyDomain to actually serve content: IsManaged
// must not claim such a service as hosted here.
type hosted interface {
	IsHosted() bool
}

// IsManaged reports whether a service of type T is configured for the domain
// behind fqdn, which must carry one of the given host prefixes.
//
// Names outside those prefixes yield false rather than an error: this backs
// the Caddy on-demand TLS hook, so happyDomain must never claim a name it does
// not answer on. Likewise, a service found but not actually hosted (see the
// hosted interface) yields false.
func IsManaged[T happydns.ServiceBody](s Store, fqdn string, prefixes ...string) (bool, error) {
	domain, ok := StripPrefix(fqdn, prefixes...)
	if !ok {
		return false, nil
	}

	svc, err := FindService[T](s, domain)
	if err != nil {
		if errors.Is(err, happydns.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	if h, ok := any(svc).(hosted); ok && !h.IsHosted() {
		return false, nil
	}

	return true, nil
}
