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
	_ "github.com/DNSControl/dnscontrol/v4/providers/hetznerv2"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type HetznerAPI struct {
	APIToken string `json:"api_token,omitempty" happydomain:"label=API Token,placeholder=xxxxxxxxxx,required,secret,description=Hetzner API token from the Cloud Console"`
}

func (s *HetznerAPI) DNSControlName() string {
	return "HETZNER_V2"
}

func (s *HetznerAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *HetznerAPI) ToDNSControlConfig() (map[string]string, error) {
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
		Website:     "https://www.hetzner.com",
	}, providerReg.RegisterProvider)
}
