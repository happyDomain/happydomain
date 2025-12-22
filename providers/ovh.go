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
	"flag"

	_ "github.com/StackExchange/dnscontrol/v4/providers/ovh"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

var (
	appKey    string
	appSecret string
)

type OVHAPI struct {
	Endpoint    string `json:"endpoint,omitempty" happydomain:"label=Endpoint,placeholder=ovh-eu,description=API endpoint (ovh-eu, ovh-ca, ovh-us, etc.)"`
	ConsumerKey string `json:"consumerkey,omitempty" happydomain:"label=Consumer Key,required,secret,description=OVH Consumer Key obtained from API token generation"`
}

func (s *OVHAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *OVHAPI) DNSControlName() string {
	return "OVH"
}

func (s *OVHAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"app-key":        appKey,
		"app-secret-key": appSecret,
		"consumer-key":   s.ConsumerKey,
	}

	return config, nil
}

func init() {
	flag.StringVar(&appKey, "ovh-application-key", "", "Application Key for using the OVH API")
	flag.StringVar(&appSecret, "ovh-application-secret", "", "Application Secret for using the OVH API")

	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &OVHAPI{}
	}, happydns.ProviderInfos{
		Name:        "OVH",
		Description: "European cloud and hosting provider with DNS services.",
	}, RegisterProvider)
}
