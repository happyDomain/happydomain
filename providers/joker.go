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
	_ "github.com/StackExchange/dnscontrol/v4/providers/joker"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type JokerAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,description=Required if not using API key"`
	Password string `json:"password,omitempty" happydomain:"label=Password,secret,description=Required if not using API key"`
	APIKey   string `json:"api-key,omitempty" happydomain:"label=API Key,secret,description=Alternative to username/password authentication"`
}

func (s *JokerAPI) DNSControlName() string {
	return "JOKER"
}

func (s *JokerAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *JokerAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{}

	if s.Username != "" {
		config["username"] = s.Username
	}
	if s.Password != "" {
		config["password"] = s.Password
	}
	if s.APIKey != "" {
		config["api-key"] = s.APIKey
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &JokerAPI{}
	}, happydns.ProviderInfos{
		Name:        "Joker.com",
		Description: "Domain registrar and DNS hosting service.",
	}, RegisterProvider)
}
