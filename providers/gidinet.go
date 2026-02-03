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
	_ "github.com/StackExchange/dnscontrol/v4/providers/gidinet"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type GidinetAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=your-username,required,description=Your Gidinet account username"`
	Password string `json:"password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,secret,description=Your Gidinet account password"`
}

func (s *GidinetAPI) DNSControlName() string {
	return "GIDINET"
}

func (s *GidinetAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *GidinetAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"username": s.Username,
		"password": s.Password,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &GidinetAPI{}
	}, happydns.ProviderInfos{
		Name:        "Gidinet",
		Description: "DNS and domain registrar service with comprehensive management features.",
	}, RegisterProvider)
}
