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
	_ "github.com/StackExchange/dnscontrol/v4/providers/powerdns"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type PowerdnsAPI struct {
	ApiUrl        string `json:"apiurl,omitempty" happydomain:"label=API Server Endpoint,placeholder=http://12.34.56.78"`
	ApiKey        string `json:"apikey,omitempty" happydomain:"label=API Key,placeholder=a0b1c2d3e4f5=="`
	ServerID      string `json:"server_id,omitempty" happydomain:"label=Server ID,placeholder=localhost,description=Unless you are using a specially configured reverse proxy leave blank"`
	Certificate   string `json:"certificate,omitempty" happydomain:"label=Certificate,placeholder=-----BEGIN CERTIFICATE-----,description=If you use a self-signed certificate paste it here,textarea"`
	SkipTLSVerify bool   `json:"skip_tls_verify,omitempty" happydomain:"label=Skip TLS Verify,description=Don't check the validity of the presented certificate (THIS IS INSECURE)"`
}

func (s *PowerdnsAPI) DNSControlName() string {
	return "POWERDNS"
}

func (s *PowerdnsAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *PowerdnsAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"apiKey":     s.ApiKey,
		"apiUrl":     s.ApiUrl,
		"serverName": s.ServerID,
	}

	if s.SkipTLSVerify {
		config["skipTLSVerify"] = "true"
	}

	if s.Certificate != "" {
		config["cert"] = s.Certificate
	}

	if s.ServerID == "" {
		config["serverName"] = "localhost"
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &PowerdnsAPI{}
	}, happydns.ProviderInfos{
		Name:        "PowerDNS",
		Description: "If your zone is hosted on an authoritative name server that runs PowerDNS, with available HTTP API",
	}, RegisterProvider)
}
