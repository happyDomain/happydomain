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

// EmailAutoconfigUsecase serves the public mail-client auto-configuration
// endpoints (Mozilla Autoconfig + Microsoft Autodiscover) and the Caddy
// on-demand TLS validation hook.
//
// All methods take fully-qualified domain names. The usecase looks up the
// owning Domain in storage, finds the latest Zone, and reads the
// EmailAutoConfig service body to render the appropriate response.
type EmailAutoconfigUsecase interface {
	// IsManaged returns true if the given FQDN is hosted by happyDomain
	// for the email auto-configuration purpose. It strips an
	// "autoconfig." or "autodiscover." prefix and checks that the parent
	// domain has a configured EmailAutoConfig service.
	IsManaged(fqdn string) (bool, error)

	// MozillaConfig renders the Thunderbird-style XML for the given
	// domain. emailAddress may be empty.
	MozillaConfig(domainFQDN, emailAddress string) ([]byte, error)

	// AutodiscoverConfig renders the Outlook-style XML for the given
	// domain. emailAddress may be empty.
	AutodiscoverConfig(domainFQDN, emailAddress string) ([]byte, error)
}
