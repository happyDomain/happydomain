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
	_ "github.com/StackExchange/dnscontrol/v4/providers/bunnydns"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type BunnyDNSAPI struct {
	APIKey string `json:"api_key,omitempty" happydomain:"label=API Key,required,secret"`
}

func (s *BunnyDNSAPI) DNSControlName() string {
	return "BUNNY_DNS"
}

func (s *BunnyDNSAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *BunnyDNSAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"api_key": s.APIKey,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &BunnyDNSAPI{}
	}, happydns.ProviderInfos{
		Name:        "Bunny DNS",
		Description: "High-performance DNS service from Bunny.net CDN.",
	}, RegisterProvider)
}
