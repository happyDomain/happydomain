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
	_ "github.com/DNSControl/dnscontrol/v4/providers/gigahost"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type GigahostAPI struct {
	APIKey string `json:"apikey,omitempty" happydomain:"label=API Key,placeholder=flux_live_xxxxxxxxxx,required,secret,description=Gigahost API key with DNS read-write permission (flux_live_...)"`
}

func (s *GigahostAPI) DNSControlName() string {
	return "GIGAHOST"
}

func (s *GigahostAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *GigahostAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"apikey": s.APIKey,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &GigahostAPI{}
	}, happydns.ProviderInfos{
		Name:        "Gigahost",
		Description: "Danish hosting provider with DNS services.",
	}, providerReg.RegisterProvider)
}
