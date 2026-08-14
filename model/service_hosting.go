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

package happydns

// HostedDomainValidator reports whether happyDomain serves content for a given
// FQDN, and therefore whether a TLS certificate should be obtained for it.
//
// Each hosted service (email auto-configuration, MTA-STS, …) implements it for
// the host names it answers on; the Caddy on-demand TLS hook asks them all.
type HostedDomainValidator interface {
	// IsManaged returns true if the given FQDN is hosted by happyDomain. It
	// must return false — not an error — for any name outside the prefixes
	// the service is responsible for, so that happyDomain never authorises a
	// certificate for an arbitrary domain.
	IsManaged(fqdn string) (bool, error)
}

// MTASTSUsecase serves the public MTA-STS policy file (RFC 8461) for the
// domains that asked happyDomain to host it.
//
// All methods take fully-qualified domain names. The usecase looks up the
// owning Domain in storage, finds the latest Zone, and reads the MTASTS
// service body to render the policy.
type MTASTSUsecase interface {
	HostedDomainValidator

	// Policy renders the RFC 8461 policy file for the given domain, which may
	// be given either as the bare domain or as its mta-sts. host.
	Policy(domainFQDN string) ([]byte, error)
}
