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

// Package emailautoconfig serves the public mail-client auto-configuration
// HTTP endpoints (Mozilla Autoconfig + Microsoft Autodiscover).
package emailautoconfig

import (
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/services/abstract"
	"git.happydns.org/happyDomain/model"
)

// DomainFinder looks up Domains by FQDN across all users.
type DomainFinder interface {
	FindDomainsByName(fqdn string) ([]*happydns.Domain, error)
}

// ZoneGetter retrieves a Zone by its identifier.
type ZoneGetter interface {
	Get(zoneID happydns.Identifier) (*happydns.Zone, error)
}

// Usecase implements happydns.EmailAutoconfigUsecase.
type Usecase struct {
	domains DomainFinder
	zones   ZoneGetter
}

// NewUsecase constructs an Usecase wired to the given storage adapters.
func NewUsecase(domains DomainFinder, zones ZoneGetter) *Usecase {
	return &Usecase{domains: domains, zones: zones}
}

// stripDiscoveryPrefix removes a leading "autoconfig." or "autodiscover."
// from the given FQDN, returning the parent domain. If the prefix is absent,
// the original FQDN is returned unchanged.
func stripDiscoveryPrefix(fqdn string) string {
	fqdn = dns.Fqdn(fqdn)
	for _, prefix := range []string{"autoconfig.", "autodiscover."} {
		if strings.HasPrefix(fqdn, prefix) {
			return fqdn[len(prefix):]
		}
	}
	return fqdn
}

// findService walks every owner of the given parent domain, loads the latest
// zone, and returns the first EmailAutoConfig service found at the apex.
//
// Returns happydns.ErrNotFound if no domain matches or none has the service.
func (uc *Usecase) findService(parentFQDN string) (*abstract.EmailAutoConfig, *happydns.Domain, error) {
	domains, err := uc.domains.FindDomainsByName(parentFQDN)
	if err != nil {
		return nil, nil, err
	}

	for _, d := range domains {
		if len(d.ZoneHistory) == 0 {
			continue
		}
		zone, err := uc.zones.Get(d.ZoneHistory[0])
		if err != nil {
			continue
		}
		for _, services := range zone.Services {
			for _, s := range services {
				if ec, ok := s.Service.(*abstract.EmailAutoConfig); ok {
					return ec, d, nil
				}
			}
		}
	}

	return nil, nil, happydns.ErrNotFound
}

// MozillaConfig renders the Thunderbird-style XML for the given FQDN.
// emailAddress is optional and only used for the <emailProvider id=...>
// attribute when the domain itself isn't enough.
func (uc *Usecase) MozillaConfig(domainFQDN, emailAddress string) ([]byte, error) {
	parent := stripDiscoveryPrefix(domainFQDN)
	svc, _, err := uc.findService(parent)
	if err != nil {
		return nil, err
	}

	bareDomain := strings.TrimSuffix(parent, ".")
	return RenderMozillaXML(svc, bareDomain, emailAddress)
}

// AutodiscoverConfig renders the Outlook-style XML for the given FQDN.
func (uc *Usecase) AutodiscoverConfig(domainFQDN, emailAddress string) ([]byte, error) {
	parent := stripDiscoveryPrefix(domainFQDN)
	svc, _, err := uc.findService(parent)
	if err != nil {
		return nil, err
	}

	bareDomain := strings.TrimSuffix(parent, ".")
	return RenderAutodiscoverXML(svc, bareDomain, emailAddress)
}
