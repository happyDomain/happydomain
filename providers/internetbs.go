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
	_ "github.com/StackExchange/dnscontrol/v4/providers/internetbs"

	"git.happydns.org/happyDomain/model"
)

type InternetbsAPI struct {
	ApiKey string `json:"apikey,omitempty" happydomain:"label=API Key,placeholder=xxxxxxxx,required,description=Your API key."`
	Password  string `json:"password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,description=Your account password."`
}

func (s *InternetbsAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"api-key": s.ApiKey,
		"password":  s.Password,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *InternetbsAPI) DNSControlName() string {
	return "INTERNETBS"
}

func init() {
	RegisterProvider(func() happydns.Provider {
		return &InternetbsAPI{}
	}, ProviderInfos{
		Name:        "Internet.bs",
		Description: "Domain registration and web hosting company based in the Bahamas",
	})
}
