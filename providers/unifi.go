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
	_ "github.com/StackExchange/dnscontrol/v4/providers/unifi"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type UnifiAPI struct {
	Host          string `json:"host,omitempty" happydomain:"label=Host,placeholder=https://192.168.1.1,description=Local UniFi controller URL (use either Host or Console ID)"`
	ConsoleID     string `json:"console_id,omitempty" happydomain:"label=Console ID,placeholder=28704E24...:1008810555,description=UniFi cloud console ID (use either Host or Console ID)"`
	APIKey        string `json:"api_key,omitempty" happydomain:"label=API Key,required,secret"`
	Site          string `json:"site,omitempty" happydomain:"label=Site,placeholder=default,description=UniFi site name (defaults to 'default')"`
	APIVersion    string `json:"api_version,omitempty" happydomain:"label=API Version,choices=auto;new;legacy,default=auto,description=API version to use (auto detects automatically)"`
	SkipTLSVerify bool   `json:"skip_tls_verify,omitempty" happydomain:"label=Skip TLS Verify,description=Don't check the validity of the presented certificate (THIS IS INSECURE)"`
}

func (s *UnifiAPI) DNSControlName() string {
	return "UNIFI"
}

func (s *UnifiAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *UnifiAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"api_key": s.APIKey,
	}

	if s.Host != "" {
		config["host"] = s.Host
	}

	if s.ConsoleID != "" {
		config["console_id"] = s.ConsoleID
	}

	if s.Site != "" {
		config["site"] = s.Site
	}

	if s.APIVersion != "" {
		config["api_version"] = s.APIVersion
	}

	if s.SkipTLSVerify {
		config["skip_tls_verify"] = "true"
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &UnifiAPI{}
	}, happydns.ProviderInfos{
		Name:        "UniFi",
		Description: "If your local DNS is managed by a UniFi Network controller (local or cloud access).",
	}, RegisterProvider)
}
