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
// HTTP endpoints (Mozilla Autoconfig + Microsoft Autodiscover) and the Caddy
// on-demand TLS validation hook.
package emailautoconfig

import (
	"strings"

	"git.happydns.org/happyDomain/internal/usecase/hostedservice"
	"git.happydns.org/happyDomain/services/abstract"
)

// discoveryPrefixes are the host names mail clients probe, and the only ones
// happyDomain answers auto-configuration on.
var discoveryPrefixes = []string{"autoconfig.", "autodiscover."}

// Usecase implements happydns.EmailAutoconfigUsecase.
type Usecase struct {
	store hostedservice.Store
}

// NewUsecase constructs an Usecase wired to the given storage adapters.
func NewUsecase(domains hostedservice.DomainFinder, zones hostedservice.ZoneGetter) *Usecase {
	return &Usecase{store: hostedservice.NewStore(domains, zones)}
}

// findService returns the EmailAutoConfig service of the domain behind the
// given FQDN, whose autoconfig./autodiscover. prefix is stripped if present.
func (uc *Usecase) findService(fqdn string) (*abstract.EmailAutoConfig, string, error) {
	parent, _ := hostedservice.StripPrefix(fqdn, discoveryPrefixes...)

	svc, err := hostedservice.FindService[*abstract.EmailAutoConfig](uc.store, parent)
	if err != nil {
		return nil, "", err
	}

	return svc, strings.TrimSuffix(parent, "."), nil
}

// IsManaged reports whether happyDomain hosts the email auto-configuration
// for the given FQDN. Used by the Caddy on-demand TLS ask endpoint.
func (uc *Usecase) IsManaged(fqdn string) (bool, error) {
	return hostedservice.IsManaged[*abstract.EmailAutoConfig](uc.store, fqdn, discoveryPrefixes...)
}

// MozillaConfig renders the Thunderbird-style XML for the given FQDN.
// emailAddress is optional and only used for the <emailProvider id=...>
// attribute when the domain itself isn't enough.
func (uc *Usecase) MozillaConfig(domainFQDN, emailAddress string) ([]byte, error) {
	svc, bareDomain, err := uc.findService(domainFQDN)
	if err != nil {
		return nil, err
	}

	return RenderMozillaXML(svc, bareDomain, emailAddress)
}

// AutodiscoverConfig renders the Outlook-style XML for the given FQDN.
func (uc *Usecase) AutodiscoverConfig(domainFQDN, emailAddress string) ([]byte, error) {
	svc, bareDomain, err := uc.findService(domainFQDN)
	if err != nil {
		return nil, err
	}

	return RenderAutodiscoverXML(svc, bareDomain, emailAddress)
}
