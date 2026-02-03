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
	_ "github.com/StackExchange/dnscontrol/v4/providers/dnscale"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type DNScaleAPI struct {
	ApiKey string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxx,required,secret,description=Your DNScale API key"`
	ApiURL string `json:"api_url,omitempty" happydomain:"label=API URL,placeholder=https://api.dnscale.eu/v1,description=Custom API endpoint if needed"`
}

func (s *DNScaleAPI) DNSControlName() string {
	return "DNSCALE"
}

func (s *DNScaleAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *DNScaleAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"api_key": s.ApiKey,
	}

	if s.ApiURL != "" {
		config["api_url"] = s.ApiURL
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &DNScaleAPI{}
	}, happydns.ProviderInfos{
		Name:        "DNScale",
		Description: "European DNS hosting provider with advanced features and API support.",
	}, RegisterProvider)
}
