// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: Pirre-Olivier Mercier, et al.
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/sakuracloud"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type SakuraCloudAPI struct {
	AccessToken       string `json:"access_token,omitempty" happydomain:"label=Access Token,placeholder=xxxxxxxx,required,description=Your access token"`
	AccessTokenSecret string `json:"access_token_secret,omitempty" happydomain:"label=Access Token Secret,placeholder=xxxxxxxx,required,description=Your secret"`
	Endpoint          string `json:"endpoint,omitempty" happydomain:"label=Endpoint,placeholder=https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1,description=Any zone endpoint (as DNS service is independent of zone)"`
}

func (s *SakuraCloudAPI) DNSControlName() string {
	return "SAKURACLOUD"
}

func (s *SakuraCloudAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *SakuraCloudAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"access_token":        s.AccessToken,
		"access_token_secret": s.AccessTokenSecret,
		"endpoint":            s.Endpoint,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &SakuraCloudAPI{}
	}, happydns.ProviderInfos{
		Name:        "Sakura Cloud",
		Description: "Japanees Cloud Provider",
	}, RegisterProvider)
}
