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
	"flag"

	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/ovh"
)

var (
	appKey    string
	appSecret string
)

type OVHAPI struct {
	Endpoint    string `json:"endpoint,omitempty"`
	ConsumerKey string `json:"consumerkey,omitempty" happydomain:"required"`
}

func (s *OVHAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"app-key":        appKey,
		"app-secret-key": appSecret,
		"consumer-key":   s.ConsumerKey,
	}
	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *OVHAPI) DNSControlName() string {
	return "OVH"
}

func init() {
	flag.StringVar(&appKey, "ovh-application-key", "", "Application Key for using the OVH API")
	flag.StringVar(&appSecret, "ovh-application-secret", "", "Application Secret for using the OVH API")

	RegisterProvider(func() Provider {
		return &OVHAPI{}
	}, ProviderInfos{
		Name:        "OVH",
		Description: "European hosting provider.",
	})
}
