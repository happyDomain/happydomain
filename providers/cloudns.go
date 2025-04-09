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
	_ "github.com/StackExchange/dnscontrol/v4/providers/cloudns"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type ClouDNSAPI struct {
	AuthID    string `json:"AuthID,omitempty" happydomain:"label=Auth ID,placeholder=xxxxxxxx,required,description=Your ClouDNS auth ID"`
	SubAuthID string `json:"SubAuthID,omitempty" happydomain:"label=Sub Auth ID,placeholder=xxxxxxxx,description=Your ClouDNS subauth token"`
	Password  string `json:"Password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,description=Your ClouDNS API password token"`
}

func (s *ClouDNSAPI) DNSControlName() string {
	return "CLOUDNS"
}

func (s *ClouDNSAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *ClouDNSAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"auth-id":       s.AuthID,
		"sub-auth-id":   s.SubAuthID,
		"auth-password": s.Password,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &ClouDNSAPI{}
	}, happydns.ProviderInfos{
		Name:        "ClouDNS",
		Description: "ClouDNS LTD is provider of global Managed DNS services, including GeoDNS, Anycast DNS and DDoS protected DNS",
	}, RegisterProvider)
}
