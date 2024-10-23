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
	"github.com/StackExchange/dnscontrol/v4/providers"
	_ "github.com/StackExchange/dnscontrol/v4/providers/hedns"
)

type HEDNSAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=xxxxxxxx,required,description=The username you usually use to log on HE services."`
	Password string `json:"password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,description=The password associated with you HE account."`
	TOTP     string `json:"totp,omitempty" happydomain:"label=TOTP Key,placeholder=xxxxxxxx,description=If you enabled two factor authentication, you need to paste here your TOTP key."`
}

func (s *HEDNSAPI) NewDNSServiceProvider() (providers.DNSServiceProvider, error) {
	config := map[string]string{
		"username": s.Username,
		"password": s.Password,
	}

	if s.TOTP != "" {
		config["totp-key"] = s.TOTP
	}

	return providers.CreateDNSProvider(s.DNSControlName(), config, nil)
}

func (s *HEDNSAPI) DNSControlName() string {
	return "HEDNS"
}

func init() {
	RegisterProvider(func() Provider {
		return &HEDNSAPI{}
	}, ProviderInfos{
		Name:        "Hurricane Electric",
		Description: "American Internet service provider.",
	})
}
