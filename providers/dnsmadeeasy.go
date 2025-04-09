// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/dnsmadeeasy"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type DNSMadeEasyAPI struct {
	ApiKey    string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx,required,description=API Key to retrieve from your account: See https://api-docs.dnsmadeeasy.com/."`
	SecretKey string `json:"secret_key,omitempty" happydomain:"label=Secret Key,placeholder=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxx,required,description=Secret key that comes with your API Key."`
}

func (s *DNSMadeEasyAPI) DNSControlName() string {
	return "DNSMADEEASY"
}

func (s *DNSMadeEasyAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *DNSMadeEasyAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"api_key":    s.ApiKey,
		"secret_key": s.SecretKey,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &DNSMadeEasyAPI{}
	}, happydns.ProviderInfos{
		Name:        "DNSMadeEasy",
		Description: "Fast and reliable DNS service provider.",
	}, RegisterProvider)
}
