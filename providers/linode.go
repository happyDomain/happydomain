// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2024 happyDomain
// Authors: David Dernoncourt, et al.
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
	_ "github.com/StackExchange/dnscontrol/v4/providers/linode"

	"git.happydns.org/happyDomain/adapters"
	"git.happydns.org/happyDomain/model"
)

type LinodeAPI struct {
	Token string `json:"token,omitempty" happydomain:"label=Token,placeholder=xxxxxxxx,required,description=Your Linode Personal Access Token."`
}

func (s *LinodeAPI) DNSControlName() string {
	return "LINODE"
}

func (s *LinodeAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *LinodeAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"token": s.Token,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &LinodeAPI{}
	}, happydns.ProviderInfos{
		Name:        "Linode, LLC",
		Description: "American cloud hosting company, based in Philadelphia.",
	}, RegisterProvider)
}
