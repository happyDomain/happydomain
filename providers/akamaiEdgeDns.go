// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/akamaiedgedns"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type AkamaiEdgeDnsAPI struct {
	ClientSecret string `json:"clientsecret,omitempty" happydomain:"label=Client Secret,placeholder=xxxxxxxx,required,description=Your Akamai Client Secret (You must enable API-Access for your account)."`
	Host         string `json:"host,omitempty" happydomain:"label=Host,placeholder=akaa-xxxxxxxxxxx.xxxx.akamaiapis.net,required,description=Your Akamai Host."`
	AccessToken  string `json:"accesstoken,omitempty" happydomain:"label=Access Token,placeholder=akaa-xxxxxxxxxxx,description=Your Akamai Access Token."`
	ClientToken  string `json:"clienttoken,omitempty" happydomain:"label=Client Token,placeholder=akaa-xxxxxxxxxxx,description=Your Akamai Client Token"`
	ContractId   string `json:"contractid,omitempty" happydomain:"label=Contract ID,placeholder=X-XXXX,description=Your Akamai Contract ID."`
	GroupId      string `json:"groupId,omitempty" happydomain:"label=Group ID,placeholder=NNNNNN,description=Your Akamai Group ID."`
}

func (s *AkamaiEdgeDnsAPI) DNSControlName() string {
	return "AKAMAIEDGEDNS"
}

func (s *AkamaiEdgeDnsAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *AkamaiEdgeDnsAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"client_secret": s.ClientSecret,
		"host":          s.Host,
		"access_token":  s.AccessToken,
		"client_token":  s.ClientToken,
		"contract_id":   s.ContractId,
		"group_id":      s.GroupId,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &AkamaiEdgeDnsAPI{}
	}, happydns.ProviderInfos{
		Name:        "Akamai Edge DNS",
		Description: "American content delivery network and cloud service company - https://www.akamai.com",
	}, RegisterProvider)
}
