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
	spaceship "github.com/libdns/spaceship"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type SpaceshipAPI struct {
	APIKey    string `json:"api_key,omitempty" happydomain:"label=API Key,secret,required,description=Your Spaceship API key"`
	APISecret string `json:"api_secret,omitempty" happydomain:"label=API Secret,secret,required,description=Your Spaceship API secret"`
}

func (s *SpaceshipAPI) LibdnsProvider() any {
	return &spaceship.Provider{
		APIKey:    s.APIKey,
		APISecret: s.APISecret,
	}
}

func (s *SpaceshipAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewLibdnsProviderAdapter(s)
}

func init() {
	adapter.RegisterLibdnsProviderAdapter(func() happydns.ProviderBody {
		return &SpaceshipAPI{}
	}, happydns.ProviderInfos{
		Name:        "Spaceship",
		Description: "Domain registrar and DNS provider",
	}, providerReg.RegisterProvider)
}
