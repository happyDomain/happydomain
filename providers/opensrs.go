// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: David Dernoncourt, et al.
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/opensrs"
)

type OpensrsAPI struct {
	ApiKey   string `json:"apiKey,omitempty" happydomain:"label=API key,placeholder=xxxxxxxx,required,description=Your API key."`
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=xxxxxxxx,required,description=Your username."`
	BaseUrl  string `json:"base_url,omitempty" happydomain:"label=Base URL,placeholder=xxxxxxxx,description=Alternate base URL."`
}

func (s *OpensrsAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"apikey":   s.ApiKey,
		"username": s.Username,
		"baseurl":  s.BaseUrl,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *OpensrsAPI) DNSControlName() string {
	return "OPENSRS"
}

func init() {
	RegisterProvider(func() Provider {
		return &OpensrsAPI{}
	}, ProviderInfos{
		Name:        "OpenSRS",
		Description: "Domain registrar platform and wholesale domain services provider based in Toronto, Canada, and part of Tucows.",
	})
}
