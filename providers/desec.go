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
	_ "github.com/StackExchange/dnscontrol/v4/providers/desec"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type DeSECAPI struct {
	Token string `json:"token,omitempty" happydomain:"label=API Token,placeholder=your-api-key,required,secret,description=deSEC API authentication token"`
}

func (s *DeSECAPI) DNSControlName() string {
	return "DESEC"
}

func (s *DeSECAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *DeSECAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"auth-token": s.Token,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &DeSECAPI{}
	}, happydns.ProviderInfos{
		Name:        "deSEC",
		Description: "Free and secure DNS hosting with DNSSEC support.",
	}, RegisterProvider)
}
