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
	_ "github.com/StackExchange/dnscontrol/v4/providers/netlify"

	"git.happydns.org/happyDomain/internal/adapters"
	"git.happydns.org/happyDomain/model"
)

type NetlifyAPI struct {
	Token string `json:"token,omitempty" happydomain:"label=Netlify Access Token,placeholder=xxxxxxxxxx,required,description=Get your token on https://app.netlify.com/user/applications#personal-access-tokens."`
	Slug  string `json:"slug,omitempty" happydomain:"label=Account Slug,description=Optional account slug (help us to understand how it is used)."`
}

func (s *NetlifyAPI) DNSControlName() string {
	return "NETLIFY"
}

func (s *NetlifyAPI) InstantiateProvider() (happydns.ProviderActuator, error) {
	return adapter.NewDNSControlProviderAdapter(s)
}

func (s *NetlifyAPI) ToDNSControlConfig() (map[string]string, error) {
	config := map[string]string{
		"token": s.Token,
	}

	if len(s.Slug) > 0 {
		config["slug"] = s.Slug
	}

	return config, nil
}

func init() {
	adapter.RegisterDNSControlProviderAdapter(func() happydns.ProviderBody {
		return &NetlifyAPI{}
	}, happydns.ProviderInfos{
		Name:        "Netlify",
		Description: "US remote-first cloud computing company.",
	}, RegisterProvider)
}
