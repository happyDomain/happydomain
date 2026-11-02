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

//go:build nocloud

package providers // import "git.happydns.org/happyDomain/providers"

import (
	"github.com/libdns/huaweicloud"

	"git.happydns.org/happyDomain/internal/adapters"
	providerReg "git.happydns.org/happyDomain/internal/providerregistry"
	"git.happydns.org/happyDomain/model"
)

// The DNSControl provider pulls the whole Huawei Cloud SDK (about 3 MB), the
// nocloud build relies on libdns instead, which talks to the API itself.

func (s *HuaweiCloudAPI) LibdnsProvider() any {
	return &huaweicloud.Provider{
		AccessKeyId:     s.KeyID,
		SecretAccessKey: s.SecretKey.Reveal(),
		RegionId:        s.Region,
	}
}

func (s *HuaweiCloudAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewLibdnsProviderAdapter(s)
}

func init() {
	adapter.RegisterLibdnsProviderAdapter(func() happydns.ProviderBody {
		return &HuaweiCloudAPI{}
	}, huaweiCloudInfos, providerReg.RegisterProvider)
}
