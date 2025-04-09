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
	_ "github.com/StackExchange/dnscontrol/v4/providers/azureprivatedns"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type AzurePrivateDnsAPI struct {
	SubscriptionID string `json:"SubscriptionID,omitempty" happydomain:"label=Subscription ID,placeholder=xxxxxxxx,required,description=Your Azure Client Subscription ID."`
	ResourceGroup  string `json:"ResourceGroup,omitempty" happydomain:"label=Resource Group,placeholder=xxxxxxxx,required,description=Your Azure Resource Group."`
	TenantID       string `json:"TenantID,omitempty" happydomain:"label=Tenant ID,placeholder=xxxxxxxx,description=Your Azure Tenant ID."`
	ClientID       string `json:"ClientID,omitempty" happydomain:"label=Client ID,placeholder=xxxxxxxx,description=Your Azure Client ID."`
	ClientSecret   string `json:"ClientSecret,omitempty" happydomain:"label=Client Secret,placeholder=xxxxxxxx,description=Your Azure Client Secret."`
}

func (s *AzurePrivateDnsAPI) DNSControlName() string {
	return "AZURE_PRIVATE_DNS"
}

func (s *AzurePrivateDnsAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *AzurePrivateDnsAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"SubscriptionID": s.SubscriptionID,
		"ResourceGroup":  s.ResourceGroup,
		"TenantID":       s.TenantID,
		"ClientID":       s.ClientID,
		"ClientSecret":   s.ClientSecret,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &AzurePrivateDnsAPI{}
	}, happydns.ProviderInfos{
		Name:        "Azure Private DNS",
		Description: "Exclusively to manage Private DNS zones. Use Azure DNS for public zones.",
	}, RegisterProvider)
}
