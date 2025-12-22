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
	_ "github.com/StackExchange/dnscontrol/v4/providers/alidns"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type AliDNSAPI struct {
	AccessKeyID     string `json:"access_key_id,omitempty" happydomain:"label=Access Key ID,required"`
	AccessKeySecret string `json:"access_key_secret,omitempty" happydomain:"label=Access Key Secret,required,secret"`
	RegionID        string `json:"region_id,omitempty" happydomain:"label=Region ID,placeholder=cn-hangzhou,description=Defaults to cn-hangzhou if not specified"`
}

func (s *AliDNSAPI) DNSControlName() string {
	return "ALIDNS"
}

func (s *AliDNSAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *AliDNSAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"access_key_id":     s.AccessKeyID,
		"access_key_secret": s.AccessKeySecret,
	}

	if s.RegionID != "" {
		config["region_id"] = s.RegionID
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &AliDNSAPI{}
	}, happydns.ProviderInfos{
		Name:        "Alibaba Cloud DNS",
		Description: "Alibaba Cloud's global DNS service (formerly Aliyun DNS).",
	}, RegisterProvider)
}
