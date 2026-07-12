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
	_ "github.com/DNSControl/dnscontrol/v4/providers/netnod"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type NetnodAPI struct {
	APIKey string `json:"apiKey,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxxxx,required,secret,description=API key for the Netnod Primary DNS API."`
	APIUrl string `json:"apiUrl,omitempty" happydomain:"label=API URL,placeholder=https://primarydnsapi.netnod.se,description=Base URL of the Netnod Primary DNS API. Leave blank to use the default."`
}

func (s *NetnodAPI) DNSControlName() string {
	return "NETNOD"
}

func (s *NetnodAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *NetnodAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"apiKey": s.APIKey,
		"apiUrl": s.APIUrl,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &NetnodAPI{}
	}, happydns.ProviderInfos{
		Name:        "Netnod",
		Description: "Swedish DNS provider offering enterprise Primary DNS services.",
	}, providerReg.RegisterProvider)
}
