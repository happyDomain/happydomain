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
	_ "github.com/StackExchange/dnscontrol/v4/providers/hedns"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type HEDNSAPI struct {
	Username string `json:"username,omitempty" happydomain:"label=Username,placeholder=xxxxxxxx,required,description=The username you usually use to log on HE services."`
	Password string `json:"password,omitempty" happydomain:"label=Password,placeholder=xxxxxxxx,required,description=The password associated with you HE account."`
	TOTP     string `json:"totp,omitempty" happydomain:"label=TOTP Key,placeholder=xxxxxxxx,description=If you enabled two factor authentication, you need to paste here your TOTP key."`
}

func (s *HEDNSAPI) DNSControlName() string {
	return "HEDNS"
}

func (s *HEDNSAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *HEDNSAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"username": s.Username,
		"password": s.Password,
	}

	if s.TOTP != "" {
		config["totp-key"] = s.TOTP
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &HEDNSAPI{}
	}, happydns.ProviderInfos{
		Name:        "Hurricane Electric",
		Description: "American Internet service provider.",
	}, RegisterProvider)
}
