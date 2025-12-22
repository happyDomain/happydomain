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
	_ "github.com/StackExchange/dnscontrol/v4/providers/cnr"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type CNRAPI struct {
	APILogin    string `json:"apilogin,omitempty" happydomain:"label=API Login,required"`
	APIPassword string `json:"apipassword,omitempty" happydomain:"label=API Password,required,secret"`
	APIEntity   string `json:"apientity,omitempty" happydomain:"label=API Entity,choices=OTE;LIVE,default=LIVE,required,description=Use OTE for testing or LIVE for production"`
}

func (s *CNRAPI) DNSControlName() string {
	return "CNR"
}

func (s *CNRAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *CNRAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"apilogin":    s.APILogin,
		"apipassword": s.APIPassword,
		"apientity":   s.APIEntity,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &CNRAPI{}
	}, happydns.ProviderInfos{
		Name:        "CentralNic Reseller (CNR)",
		Description: "DNS and domain management through CentralNic Reseller platform.",
	}, RegisterProvider)
}
