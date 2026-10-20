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

// Package hostedservice holds what the services happyDomain serves content for
// have in common: finding, from a public FQDN, the service body configured in
// the latest zone of whichever domain owns that name.
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

// FindService returns the first service of type T configured at fqdn itself.
// See FindServices for how fqdn is looked up.
//
// Returns happydns.ErrNotFound if no domain matches or none has the service.
func FindService[T happydns.ServiceBody](s Store, fqdn string) (T, error) {
	var zero T

	svcs, err := FindServices[T](s, fqdn)
	if err != nil {
		return zero, err
	}
	return svcs[0], nil
}

// FindServices returns the services of type T configured at fqdn itself: the
// one of the zone being edited first, then the one of the last published zone,
// which the live DNS records still refer to until the first is applied.
//
// fqdn is looked up as a registered domain, then as a subdomain of each of its
// parents, closest first: the service must never come from another subdomain
// of the same zone.
//
// Returns happydns.ErrNotFound if no domain matches or none has the service.
func FindServices[T happydns.ServiceBody](s Store, fqdn string) ([]T, error) {
	labels := dns.SplitDomainName(strings.ToLower(dns.Fqdn(fqdn)))
	for i := range labels {
		parent := dns.Fqdn(strings.Join(labels[i:], "."))
		subdomain := happydns.Subdomain(strings.Join(labels[:i], "."))

		domains, err := s.Domains.FindDomainsByName(parent)
		if errors.Is(err, happydns.ErrNotFound) {
			continue
		} else if err != nil {
			return nil, err
		}

		for _, d := range domains {
			// ZoneHistory is newest first: [0] is the zone being edited, [1]
			// the last published one.
			var found []T
			for _, zoneID := range d.ZoneHistory[:min(len(d.ZoneHistory), 2)] {
				zone, err := s.Zones.Get(zoneID)
				if err != nil {
					continue
				}
				for _, svc := range zone.Services[subdomain] {
					if body, ok := svc.Service.(T); ok {
						found = append(found, body)
						break
					}
				}
			}
			if len(found) > 0 {
				return found, nil
			}
		}
	}

	return nil, happydns.ErrNotFound
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
