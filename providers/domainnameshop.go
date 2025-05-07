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
	_ "github.com/StackExchange/dnscontrol/v4/providers/domainnameshop"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type DomainnameshopAPI struct {
	Token  string `json:"token,omitempty" happydomain:"label=Token,placeholder=your-domainnameshop-token,required,description=Domainnameshop API Token."`
	Secret string `json:"secret,omitempty" happydomain:"label=Secret,placeholder=your-domainnameshop-secret,required,description=Domainnameshop API Secret."`
}

func (s *DomainnameshopAPI) DNSControlName() string {
	return "DOMAINNAMESHOP"
}

func (s *DomainnameshopAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *DomainnameshopAPI) ToDNSControlConfig() (map[string]string, error) {
	return map[string]string{
		"token":  s.Token,
		"secret": s.Secret,
	}, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &DomainnameshopAPI{}
	}, happydns.ProviderInfos{
		Name:        "Domeneshop AS",
		Description: "Norwegian registrar and hosting company",
	}, RegisterProvider)
}
