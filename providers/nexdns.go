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

package providers // import "git.happydns.org/happyDomain/providers"

import (
	_ "github.com/DNSControl/dnscontrol/v4/providers/nexdns"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type NexDNSAPI struct {
	ApiToken string `json:"api_token,omitempty" happydomain:"label=API Token,placeholder=xxxxxxxx,required,secret,description=An API key carrying the zones.read zones.write records.read and records.write scopes."`
	ApiURL   string `json:"api_url,omitempty" happydomain:"label=API URL,placeholder=https://api.nexdns.tech/v1,description=Custom API endpoint if needed,endpoint=https://api.nexdns.tech/v1"`
}

func (s *NexDNSAPI) DNSControlName() string {
	return "NEXDNS"
}

func (s *NexDNSAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *NexDNSAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"api_token": s.ApiToken,
	}

	if s.ApiURL != "" {
		config["api_url"] = s.ApiURL
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &NexDNSAPI{}
	}, happydns.ProviderInfos{
		Name:        "NexDNS",
		Description: "DNS hosting provider, requires a plan including API access.",
		Website:     "https://nexdns.io",
	}, providerReg.RegisterProvider)
}
