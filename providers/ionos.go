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
	"github.com/libdns/ionos"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/provider"
	"git.happydns.org/happyDomain/model"
)

type IonosAPI struct {
	AuthAPIToken string `json:"auth_api_token,omitempty" happydomain:"label=Auth API Token,secret,required,description=Your IONOS Auth API token"`
}

func (s *IonosAPI) LibdnsProvider() any {
	return &ionos.Provider{
		AuthAPIToken: s.AuthAPIToken,
	}
}

func (s *IonosAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewLibdnsProviderAdapter(s)
}

func init() {
	adapter.RegisterLibdnsProviderAdapter(func() happydns.ProviderBody {
		return &IonosAPI{}
	}, happydns.ProviderInfos{
		Name:        "IONOS",
		Description: "German hosting provider (1&1 IONOS)",
		Website:     "https://www.ionos.com",
	}, providerReg.RegisterProvider)
}
