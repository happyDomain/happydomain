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
// Each service happyDomain hosts content for implements it for the host names
// it answers on; the Caddy on-demand TLS hook asks them all.
type HostedDomainValidator interface {
	// IsManaged returns true if the given FQDN is hosted by happyDomain. It
	// must return false, not an error, for any name outside the prefixes the
	// service is responsible for, so that happyDomain never authorises a
	// certificate for an arbitrary domain.
	IsManaged(fqdn string) (bool, error)
}
