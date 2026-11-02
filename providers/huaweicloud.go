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
	"git.happydns.org/happyDomain/model"
)

// HuaweiCloudAPI is shared by both implementations of the provider:
// DNSControl in the full build, libdns in the nocloud one. Keeping a single
// struct lets a stored provider be read by either build.
type HuaweiCloudAPI struct {
	KeyID     string          `json:"KeyId,omitempty" happydomain:"label=Access Key ID,required"`
	SecretKey happydns.Secret `json:"SecretKey,omitzero" happydomain:"label=Secret Access Key,required,secret"`
	Region    string          `json:"Region,omitempty" happydomain:"label=Region,placeholder=cn-north-1,required"`
}

var huaweiCloudInfos = happydns.ProviderInfos{
	Name:        "Huawei Cloud DNS",
	Description: "Huawei Cloud's global DNS service.",
	Website:     "https://www.huaweicloud.com",
}
