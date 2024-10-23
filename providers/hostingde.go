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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/hostingde"
)

type HostingdeAPI struct {
	Token          string `json:"token,omitempty" happydomain:"label=Token,placeholder=your-api-key,required,description=Provide your Hosting.de account access token."`
	OwnerAccountId string `json:"ownerAccountId,omitempty" happydomain:"label=Owner Account,placeholder=xxxxxxxxx,description=Identifier of the account owner."`
}

func (s *HostingdeAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"authToken": s.Token,
	}

	if s.OwnerAccountId != "" {
		config["ownerAccountId"] = s.OwnerAccountId
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *HostingdeAPI) DNSControlName() string {
	return "HOSTINGDE"
}

func init() {
	RegisterProvider(func() Provider {
		return &HostingdeAPI{}
	}, ProviderInfos{
		Name:        "Hosting.de",
		Description: "German hosting provider.",
	})
}
