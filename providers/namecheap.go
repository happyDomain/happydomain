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
	_ "github.com/StackExchange/dnscontrol/v4/providers/namecheap"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type NamecheapAPI struct {
	APIKey  string `json:"apikey,omitempty" happydoamin:"label=API Key,placeholder=yourApiKeyFromNameCheap,required"`
	APIUser string `json:"apiuser,omitempty" happydomain:"label=API User,placeholder=yourUsername,required"`
}

func (s *NamecheapAPI) DNSControlName() string {
	return "NAMECHEAP"
}

func (s *NamecheapAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *NamecheapAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"apikey":  s.APIKey,
		"apiuser": s.APIUser,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &NamecheapAPI{}
	}, happydns.ProviderInfos{
		Name:        "Namecheap",
		Description: "American domain name registrar.",
	}, RegisterProvider)
}
