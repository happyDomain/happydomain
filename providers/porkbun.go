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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/porkbun"
)

type PorkbunAPI struct {
	APIKey    string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxxxx,required,description=Get your API key on https://porkbun.com/account/api."`
	SecretKey string `json:"secret_key,omitempty" happydomain:"label=Secret Key,placeholder=xxxxxxxxxx,required,description=Write the secret key corresponding to your API key."`
}

func (s *PorkbunAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"api_key":    s.APIKey,
		"secret_key": s.SecretKey,
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *PorkbunAPI) DNSControlName() string {
	return "PORKBUN"
}

func init() {
	RegisterProvider(func() Provider {
		return &PorkbunAPI{}
	}, ProviderInfos{
		Name:        "Porkbun",
		Description: "US Name Registrar.",
	})
}
