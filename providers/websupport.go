// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	_ "github.com/DNSControl/dnscontrol/v4/providers/websupport"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type WebsupportAPI struct {
	APIKey string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxxxx,required,secret,description=WebSupport API key (generated in the Security section of the admin console)."`
	Secret string `json:"secret,omitempty" happydomain:"label=API Secret,placeholder=xxxxxxxxxx,required,secret,description=WebSupport API secret used to sign requests."`
}

func (s *WebsupportAPI) DNSControlName() string {
	return "WEBSUPPORT"
}

func (s *WebsupportAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *WebsupportAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"api_key": s.APIKey,
		"secret":  s.Secret,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &WebsupportAPI{}
	}, happydns.ProviderInfos{
		Name:        "WebSupport",
		Description: "Slovak domain registrar and hosting provider (websupport.sk).",
	}, providerReg.RegisterProvider)
}
