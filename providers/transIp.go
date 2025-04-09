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
	_ "github.com/StackExchange/dnscontrol/v4/providers/transip"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type TransIpAPI struct {
	AccountName string `json:"account_name,omitempty" happydomain:"label=Account name,placeholder=xxxxxxxx,description=Your account name."`
	PrivateKey  string `json:"private_key,omitempty" happydomain:"label=Private key,placeholder=xxxxxxxx,description=Your account private key."`
	AccessToken string `json:"access_token,omitempty" happydomain:"label=Access token,placeholder=xxxxxxxx,description=Your access roken."`
}

func (s *TransIpAPI) DNSControlName() string {
	return "TRANSIP"
}

func (s *TransIpAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *TransIpAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"AccountName": s.AccountName,
		"PrivateKey":  s.PrivateKey,
		"AccessToken": s.AccessToken,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &TransIpAPI{}
	}, happydns.ProviderInfos{
		Name:        "TransIP B.V.",
		Description: "Dutch hosting company",
	}, RegisterProvider)
}
