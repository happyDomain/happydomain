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
	_ "github.com/StackExchange/dnscontrol/v4/providers/hexonet"

	"git.happydns.org/happyDomain/model"
)

type HexonetAPI struct {
	APILogin    string `json:"apilogin,omitempty" happydomain:"label=API Login,placeholder=your-hexonet-account-id,required"`
	APIPassword string `json:"apipassword,omitempty" happydomain:"label=API Password,placeholder=your-hexonet-account-password,required"`
	APIEntity   string `json:"apientity,omitempty" happydomain:"label=API Entity,default=LIVE,choices=LIVE;OTE,description=Choose between the LIVE and the OT&E system"`
}

func (s *HexonetAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"apilogin":    s.APILogin,
		"apipassword": s.APIPassword,
		"apientity":   s.APIEntity,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *HexonetAPI) DNSControlName() string {
	return "HEXONET"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &HexonetAPI{}
	}, ProviderInfos{
		Name:        "Hexonet",
		Description: "Service providers for the domain industry.",
	})
}
