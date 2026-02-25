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
	_ "github.com/StackExchange/dnscontrol/v4/providers/mikrotik"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type MikrotikAPI struct {
	Host      string `json:"host,omitempty" happydomain:"label=Host,placeholder=http://192.168.88.1:8080,required,description=RouterOS REST API endpoint"`
	Username  string `json:"username,omitempty" happydomain:"label=Username,placeholder=admin,required"`
	Password  string `json:"password,omitempty" happydomain:"label=Password,required,secret"`
	ZoneHints string `json:"zonehints,omitempty" happydomain:"label=Zone Hints,placeholder=internal.corp.local\\,home.arpa,description=Comma-separated list of zone names to help identify multi-label zones"`
}

func (s *MikrotikAPI) DNSControlName() string {
	return "MIKROTIK"
}

func (s *MikrotikAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *MikrotikAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"host":     s.Host,
		"username": s.Username,
		"password": s.Password,
	}

	if s.ZoneHints != "" {
		config["zonehints"] = s.ZoneHints
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &MikrotikAPI{}
	}, happydns.ProviderInfos{
		Name:        "MikroTik",
		Description: "If your zone is hosted on a MikroTik RouterOS device, managed via its REST API static DNS entries.",
	}, RegisterProvider)
}
