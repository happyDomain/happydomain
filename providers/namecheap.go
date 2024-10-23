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
	_ "github.com/StackExchange/dnscontrol/v4/providers/namecheap"
)

type NamecheapAPI struct {
	APIKey  string `json:"apikey,omitempty" happydoamin:"label=API Key,placeholder=yourApiKeyFromNameCheap,required"`
	APIUser string `json:"apiuser,omitempty" happydomain:"label=API User,placeholder=yourUsername,required"`
}

func (s *NamecheapAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"apikey":  s.APIKey,
		"apiuser": s.APIUser,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *NamecheapAPI) DNSControlName() string {
	return "NAMECHEAP"
}

func init() {
	RegisterProvider(func() Provider {
		return &NamecheapAPI{}
	}, ProviderInfos{
		Name:        "Namecheap",
		Description: "American domain name registrar.",
	})
}
