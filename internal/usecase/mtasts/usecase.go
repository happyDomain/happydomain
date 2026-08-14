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

// Package mtasts serves the public MTA-STS policy file (RFC 8461) for the
// domains that asked happyDomain to host it.
package mtasts

import (
	"errors"
	"strings"

	"github.com/miekg/dns"

	"git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

// policyHostPrefix is the host RFC 8461 sec. 3.3 mandates the policy be served
// on, relative to the domain it applies to.
const policyHostPrefix = "mta-sts."

// DomainFinder looks up Domains by FQDN across all users.
type DomainFinder interface {
	FindDomainsByName(fqdn string) ([]*happydns.Domain, error)
}

// ZoneGetter retrieves a Zone by its identifier.
type ZoneGetter interface {
	Get(zoneID happydns.Identifier) (*happydns.Zone, error)
}

// Usecase implements happydns.MTASTSUsecase.
type Usecase struct {
	domains DomainFinder
	zones   ZoneGetter
}

// NewUsecase constructs an Usecase wired to the given storage adapters.
func NewUsecase(domains DomainFinder, zones ZoneGetter) *Usecase {
	return &Usecase{domains: domains, zones: zones}
}

// stripPolicyPrefix removes a leading "mta-sts." from the given FQDN,
// returning the domain the policy applies to. If the prefix is absent, the
// original FQDN is returned unchanged.
func stripPolicyPrefix(fqdn string) string {
	fqdn = dns.Fqdn(fqdn)
	if after, ok := strings.CutPrefix(fqdn, policyHostPrefix); ok {
		return after
	}
	return fqdn
}

// findService walks every owner of the given domain, loads the latest zone,
// and returns the first MTASTS service found.
//
// Returns happydns.ErrNotFound if no domain matches or none has the service.
func (uc *Usecase) findService(fqdn string) (*abstract.MTASTS, *happydns.Domain, error) {
	domains, err := uc.domains.FindDomainsByName(fqdn)
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
				if ms, ok := s.Service.(*abstract.MTASTS); ok {
					return ms, d, nil
				}
			}
		}
	}

	return nil, nil, happydns.ErrNotFound
}

// IsManaged reports whether happyDomain hosts the MTA-STS policy for the given
// FQDN. Used by the Caddy on-demand TLS ask endpoint, hence the strict prefix
// check: happyDomain must never authorise a certificate for a name it does not
// answer on.
func (uc *Usecase) IsManaged(fqdn string) (bool, error) {
	fqdn = dns.Fqdn(fqdn)
	if !strings.HasPrefix(fqdn, policyHostPrefix) {
		return false, nil
	}

	_, _, err := uc.findService(stripPolicyPrefix(fqdn))
	if err != nil {
		if errors.Is(err, happydns.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Policy renders the RFC 8461 policy file for the given FQDN, which may be
// given either as the bare domain or as its mta-sts. host.
func (uc *Usecase) Policy(domainFQDN string) ([]byte, error) {
	svc, _, err := uc.findService(stripPolicyPrefix(domainFQDN))
	if err != nil {
		return nil, err
	}

	// A service configured for the DNS half only has nothing to serve; saying
	// so is more useful than serving a policy the user never wrote.
	body := svc.PolicyFile()
	if body == nil {
		return nil, happydns.ErrNotFound
	}

	return body, nil
}
