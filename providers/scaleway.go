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
	_ "github.com/DNSControl/dnscontrol/v4/providers/scaleway"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

type ScalewayAPI struct {
	AccessKey      string `json:"access_key,omitempty" happydomain:"label=Access Key,placeholder=SCWXXXXXXXXXXXXXXXXX,required,description=Your Scaleway API access key"`
	SecretKey      string `json:"secret_key,omitempty" happydomain:"label=Secret Key,secret,required,description=Your Scaleway API secret key"`
	OrganizationID string `json:"organization_id,omitempty" happydomain:"label=Project ID,description=Your Scaleway Project ID (optional)"`
}

func (s *ScalewayAPI) DNSControlName() string {
	return "SCALEWAY"
}

func (s *ScalewayAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *ScalewayAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"access_key": s.AccessKey,
		"secret_key": s.SecretKey,
		"project_id": s.OrganizationID,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &ScalewayAPI{}
	}, happydns.ProviderInfos{
		Name:        "Scaleway",
		Description: "French cloud hosting provider.",
	}, providerReg.RegisterProvider)
}
