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
	_ "github.com/DNSControl/dnscontrol/v4/providers/netbird"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type NetbirdAPI struct {
	Token string `json:"token,omitempty" happydomain:"label=API Token,placeholder=xxxxxxxxxx,required,secret,description=NetBird API token from https://app.netbird.io/settings"`
}

func (s *NetbirdAPI) DNSControlName() string {
	return "NETBIRD"
}

func (s *NetbirdAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *NetbirdAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"token": s.Token,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &NetbirdAPI{}
	}, happydns.ProviderInfos{
		Name:        "NetBird",
		Description: "Peer-to-peer DNS service for NetBird networks.",
	}, providerReg.RegisterProvider)
}
