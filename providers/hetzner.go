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
	_ "github.com/StackExchange/dnscontrol/v4/providers/hetzner"
	_ "github.com/StackExchange/dnscontrol/v4/providers/hetznerv2"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type HetznerAPI struct {
	APIVersion string `json:"api_version,omitempty" happydomain:"label=API Version,choices=v2;v1,default=v2,required,description=API version to use (v2 recommended for Cloud DNS)"`
	APIToken   string `json:"api_token,omitempty" happydomain:"label=API Token,placeholder=xxxxxxxxxx,required,secret,description=Hetzner API token from https://dns.hetzner.com/settings/api-token (v1) or Cloud Console (v2)"`
}

func (s *HetznerAPI) DNSControlName() string {
	if s.APIVersion == "v1" {
		return "HETZNER"
	}
	return "HETZNER_V2"
}

func (s *HetznerAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *HetznerAPI) ToDNSControlConfig() (map[string]string, error) {
	// v1 uses api_key, v2 uses api_token
	if s.APIVersion == "v1" {
		return map[string]string{
			"api_key": s.APIToken,
		}, nil
	}
	return map[string]string{
		"api_token": s.APIToken,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &HetznerAPI{}
	}, happydns.ProviderInfos{
		Name:        "Hetzner DNS",
		Description: "German hosting provider with DNS services.",
	}, RegisterProvider)
}
