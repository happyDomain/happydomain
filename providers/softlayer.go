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
	_ "github.com/StackExchange/dnscontrol/v4/providers/softlayer"

	"git.happydns.org/happyDomain/model"
)

type SoftLayerAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=yourUsername,required"`
	APIKey   string `json:"api_key,omitempty" happydomain:"label=API Key,placeholder=yourApiKeyFromSoftLayer,required"`
}

func (s *SoftLayerAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"api_key":  s.APIKey,
		"username": s.Username,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *SoftLayerAPI) DNSControlName() string {
	return "SOFTLAYER"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &SoftLayerAPI{}
	}, ProviderInfos{
		Name:        "SoftLayer",
		Description: "Hosting provider, now IBM Cloud.",
	})
}
