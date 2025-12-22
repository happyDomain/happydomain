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
	_ "github.com/StackExchange/dnscontrol/v4/providers/realtimeregister"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type RealtimeRegisterAPI struct {
	APIKey  string `json:"apikey,omitempty" happydomain:"label=API Key,required,secret"`
	Premium bool   `json:"premium,omitempty" happydomain:"label=Premium Service,description=Enable PREMIUM service type instead of BASIC"`
	Sandbox bool   `json:"sandbox,omitempty" happydomain:"label=Sandbox Mode,description=Use sandbox API for testing"`
}

func (s *RealtimeRegisterAPI) DNSControlName() string {
	return "REALTIMEREGISTER"
}

func (s *RealtimeRegisterAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *RealtimeRegisterAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"apikey": s.APIKey,
	}

	if s.Premium {
		config["premium"] = "1"
	}
	if s.Sandbox {
		config["sandbox"] = "1"
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &RealtimeRegisterAPI{}
	}, happydns.ProviderInfos{
		Name:        "Realtime Register",
		Description: "Domain registrar and DNS hosting provider.",
	}, RegisterProvider)
}
