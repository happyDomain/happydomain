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
	"git.happydns.org/happyDomain/internal/usecase/hostedservice"
	happydns "git.happydns.org/happyDomain/model"
	"git.happydns.org/happyDomain/services/abstract"
)

// Usecase implements happydns.MTASTSUsecase.
type Usecase struct {
	store hostedservice.Store
}

// NewUsecase constructs an Usecase wired to the given storage adapters.
func NewUsecase(domains hostedservice.DomainFinder, zones hostedservice.ZoneGetter) *Usecase {
	return &Usecase{store: hostedservice.NewStore(domains, zones)}
}

// IsManaged reports whether happyDomain hosts the MTA-STS policy for the given
// FQDN. Used by the Caddy on-demand TLS ask endpoint, hence the strict prefix
// check: happyDomain must never authorise a certificate for a name it does not
// answer on.
func (uc *Usecase) IsManaged(fqdn string) (bool, error) {
	return hostedservice.IsManaged[*abstract.MTASTS](uc.store, fqdn, abstract.MTASTSPolicyHostPrefix)
}

// Policy renders the RFC 8461 policy file for the given FQDN, which may be
// given either as the bare domain or as its mta-sts. host.
func (uc *Usecase) Policy(domainFQDN string) ([]byte, error) {
	domain, _ := hostedservice.StripPrefix(domainFQDN, abstract.MTASTSPolicyHostPrefix)

	svc, err := hostedservice.FindService[*abstract.MTASTS](uc.store, domain)
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
