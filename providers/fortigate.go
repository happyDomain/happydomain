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
	_ "github.com/StackExchange/dnscontrol/v4/providers/fortigate"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type FortiGateAPI struct {
	Host        string `json:"host,omitempty" happydomain:"label=Host,placeholder=https://fortigate.example.com,required"`
	VDOM        string `json:"vdom,omitempty" happydomain:"label=Virtual Domain,placeholder=root,required"`
	APIKey      string `json:"apiKey,omitempty" happydomain:"label=API Key,required,secret"`
	InsecureTLS bool   `json:"insecure_tls,omitempty" happydomain:"label=Insecure TLS,description=Skip TLS certificate verification"`
}

func (s *FortiGateAPI) DNSControlName() string {
	return "FORTIGATE"
}

func (s *FortiGateAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *FortiGateAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"host":   s.Host,
		"vdom":   s.VDOM,
		"apiKey": s.APIKey,
	}

	if s.InsecureTLS {
		config["insecure_tls"] = "true"
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &FortiGateAPI{}
	}, happydns.ProviderInfos{
		Name:        "FortiGate",
		Description: "Fortinet FortiGate firewall with internal DNS server.",
	}, RegisterProvider)
}
