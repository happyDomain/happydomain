// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2025 happyDomain
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/cloudflare"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type CloudflareAPI struct {
	AccountID string `json:"AccountID,omitempty" happydomain:"label=Account ID,placeholder=xxxxxxxx,required,description=Your Cloudflare account ID"`
	ApiToken  string `json:"ApiToken,omitempty" happydomain:"label=API Token,placeholder=xxxxxxxx,required,secret,description=Your Cloudflare API token"`
}

func (s *CloudflareAPI) DNSControlName() string {
	return "CLOUDFLAREAPI"
}

func (s *CloudflareAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *CloudflareAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"accountid": s.AccountID,
		"apitoken":  s.ApiToken,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &CloudflareAPI{}
	}, happydns.ProviderInfos{
		Name:        "Cloudflare",
		Description: "Global CDN and DNS provider with advanced features like proxy and DNSSEC support.",
	}, RegisterProvider)
}
