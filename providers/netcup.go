// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: David Dernoncourt, et al.
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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/netcup"
)

type NetcupAPI struct {
	ApiKey         string `json:"api_key,omitempty" happydomain:"label=API key,placeholder=your-api-key,required,description=Netcup API key."`
	ApiPassword    string `json:"api_password,omitempty" happydomain:"label=Password,placeholder=api-password,required,description=Netcup API password."`
	CustomerNumber string `json:"customer_number,omitempty" happydomain:"label=Customer number,placeholder=123456,required,description=Netcup customer number."`
}

func (s *NetcupAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"api-key":         s.ApiKey,
		"api-password":    s.ApiPassword,
		"customer-number": s.CustomerNumber,
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *NetcupAPI) DNSControlName() string {
	return "NETCUP"
}

func init() {
	RegisterProvider(func() Provider {
		return &NetcupAPI{}
	}, ProviderInfos{
		Name:        "Netcup",
		Description: "German registrar and hosting company",
	})
}
