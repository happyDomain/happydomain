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
	_ "github.com/DNSControl/dnscontrol/v4/providers/openwrt"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type OpenWRTAPI struct {
	Host     string `json:"host,omitempty" happydomain:"label=Host,placeholder=http://192.168.1.1,required,description=URL of your OpenWRT router (http:// is assumed if no scheme is given)"`
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=root,required,description=OpenWRT LuCI username"`
	Password string `json:"password,omitempty" happydomain:"label=Password,required,secret,description=OpenWRT LuCI password"`
}

func (s *OpenWRTAPI) DNSControlName() string {
	return "OPENWRT"
}

func (s *OpenWRTAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *OpenWRTAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"host":     s.Host,
		"username": s.Username,
		"password": s.Password,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &OpenWRTAPI{}
	}, happydns.ProviderInfos{
		Name:        "OpenWRT",
		Description: "DNS records hosted on an OpenWRT router.",
	}, providerReg.RegisterProvider)
}
