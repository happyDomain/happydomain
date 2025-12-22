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
	_ "github.com/StackExchange/dnscontrol/v4/providers/huaweicloud"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type HuaweiCloudAPI struct {
	KeyID     string `json:"KeyId,omitempty" happydomain:"label=Access Key ID,required"`
	SecretKey string `json:"SecretKey,omitempty" happydomain:"label=Secret Access Key,required,secret"`
	Region    string `json:"Region,omitempty" happydomain:"label=Region,placeholder=cn-north-1,required"`
}

func (s *HuaweiCloudAPI) DNSControlName() string {
	return "HUAWEICLOUD"
}

func (s *HuaweiCloudAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *HuaweiCloudAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"KeyId":     s.KeyID,
		"SecretKey": s.SecretKey,
		"Region":    s.Region,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &HuaweiCloudAPI{}
	}, happydns.ProviderInfos{
		Name:        "Huawei Cloud DNS",
		Description: "Huawei Cloud's global DNS service.",
	}, RegisterProvider)
}
